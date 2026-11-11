// Package platform contains technical plumbing only; domain models stay in each service.
package platform

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

/**
 * Lee una variable de entorno con valor alternativo.
 *
 * @method Env
 * @param {string} key - Nombre de la variable.
 * @param {string} fallback - Valor usado si no está definida.
 * @returns {string} Valor de configuración.
 */
func Env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

/**
 * Exige un secreto de al menos treinta y dos caracteres.
 *
 * @method Secret
 * @param {string} key - Nombre de la variable secreta.
 * @returns {string} Secreto válido; termina el proceso si falta.
 */
func Secret(key string) string {
	v := os.Getenv(key)
	if len(v) < 32 {
		slog.Error("Configuración ausente o demasiado corta", "variable", key)
		os.Exit(1)
	}
	return v
}

/**
 * Configura logs JSON y trazas OpenTelemetry del servicio.
 *
 * @method Init
 * @param {string} service - Nombre estable del servicio.
 * @returns {func()} Cierre del exportador con presupuesto acotado.
 */
func Init(service string) func() {
	logCtx, logCancel := context.WithCancel(context.Background())
	slog.SetDefault(slog.New(slog.NewJSONHandler(NewLogWriter(logCtx, os.Stdout, service), nil)).With("service", service, "instance", Env("INSTANCE_ID", service)))
	gin.SetMode(gin.ReleaseMode)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	opts := []sdktrace.TracerProviderOption{sdktrace.WithResource(resource.NewSchemaless(attribute.String("service.name", service))), sdktrace.WithSampler(sdktrace.AlwaysSample())}
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" {
		ex, err := otlptracehttp.New(context.Background())
		if err != nil {
			slog.Error("No se pudo configurar telemetría")
			os.Exit(1)
		}
		opts = append(opts, sdktrace.WithBatcher(ex))
	}
	tp := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(tp)
	return func() {
		defer logCancel()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tp.Shutdown(ctx)
	}
}

/**
 * Obtiene el identificador W3C del span activo.
 *
 * @method TraceID
 * @param {context.Context} ctx - Contexto instrumentado.
 * @returns {string} Identificador de traza hexadecimal.
 */
func TraceID(ctx context.Context) string { return trace.SpanContextFromContext(ctx).TraceID().String() }

/**
 * Distingue la correlación HTTP dentro del contexto sin colisiones con otras claves.
 *
 * @method requestIDKey
 * @returns {requestIDKey} Representación tipada usada por este componente.
 */
type requestIDKey struct{}

/**
 * Lee el identificador de correlación validado de la solicitud.
 *
 * @method RequestID
 * @param {context.Context} ctx - Contexto de la solicitud.
 * @returns {string} Identificador o cadena vacía si falta.
 */
func RequestID(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey{}).(string)
	return value
}

/**
 * Escribe un error de negocio con el sobre público estable.
 *
 * @method Error
 * @param {*gin.Context} c - Solicitud HTTP.
 * @param {int} status - Estado HTTP.
 * @param {string} code - Código estable del contrato.
 * @param {string} message - Mensaje público sin secretos.
 * @param {any} details - Detalles de validación o nil.
 * @returns {void} Escribe JSON sin exponer detalles internos.
 */
func Error(c *gin.Context, status int, code, message string, details any) {
	if details == nil {
		details = []any{}
	}
	c.JSON(status, gin.H{"codigo": code, "mensaje": message, "traceId": TraceID(c.Request.Context()), "detalles": details})
}

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

/**
 * Configura correlación, logs y recuperación de errores HTTP.
 *
 * @method Router
 * @returns {*gin.Engine} Router sin proxies de confianza implícitos.
 */
func Router() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(MetricsMiddleware)
	_ = r.SetTrustedProxies(nil)
	r.Use(func(c *gin.Context) {
		c.Header("Cache-Control", "no-store")
		id := c.GetHeader("X-Request-Id")
		if id == "" {
			id = TraceID(c.Request.Context())
		}
		if !requestIDPattern.MatchString(id) {
			Error(c, 422, "VALIDACION", "X-Request-Id inválido", nil)
			c.Abort()
			return
		}
		c.Header("X-Request-Id", id)
		c.Request.Header.Set("X-Request-Id", id)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), requestIDKey{}, id))
		start := time.Now()
		defer func() {
			if recover() != nil {
				Error(c, 500, "ERROR_INTERNO", "No fue posible completar la operación.", nil)
				c.Abort()
			}
			route := c.FullPath()
			if route == "" {
				route = "ruta-desconocida"
			}
			slog.Info("http.request", "route", route, "method", c.Request.Method, "status", c.Writer.Status(), "duration_ms", time.Since(start).Milliseconds(), "request_id", id, "trace_id", TraceID(c.Request.Context()), "span_id", trace.SpanContextFromContext(c.Request.Context()).SpanID().String())
		}()
		c.Next()
	})
	return r
}

/**
 * Crea spans HTTP propagando contexto W3C.
 *
 * @method Instrument
 * @param {string} service - Nombre del servicio.
 * @param {http.Handler} h - Adaptador HTTP del servicio.
 * @returns {http.Handler} Adaptador instrumentado.
 */
func Instrument(service string, h http.Handler) http.Handler { return otelhttp.NewHandler(h, service) }

/**
 * Registra liveness y readiness protegidos por secreto técnico.
 *
 * @method Health
 * @param {*gin.Engine} r - Router del servicio.
 * @param {string} token - Secreto de healthchecks.
 * @param {func(context.Context) error} ready - Verificación de dependencias indispensables.
 * @returns {void} Registra rutas de salud sin exponerlas anónimamente.
 */
func Health(r *gin.Engine, token string, ready func(context.Context) error) {
	Metrics(r, token)
	r.Any("/health/*path", func(c *gin.Context) {
		supplied := sha256.Sum256([]byte(c.GetHeader("Authorization")))
		expected := sha256.Sum256([]byte("Bearer " + token))
		if len(token) < 32 || subtle.ConstantTimeCompare(supplied[:], expected[:]) != 1 {
			c.Header("WWW-Authenticate", `Bearer realm="healthcheck"`)
			c.JSON(401, gin.H{"mensaje": "No autenticado"})
			return
		}
		if c.Request.Method != http.MethodGet {
			c.Status(405)
			return
		}
		switch c.Param("path") {
		case "/live":
			c.JSON(200, gin.H{"estado": "proceso-iniciado"})
		case "/ready":
			ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
			defer cancel()
			if ready == nil {
				c.JSON(503, gin.H{"estado": "negocio-no-implementado"})
				return
			}
			if ready(ctx) != nil {
				c.JSON(503, gin.H{"estado": "dependencia-no-disponible"})
				return
			}
			c.JSON(200, gin.H{"estado": "disponible"})
		default:
			c.Status(404)
		}
	})
}

/**
 * Ejecuta HTTP con límites y apagado ordenado por señal.
 *
 * @method Serve
 * @param {http.Handler} handler - Adaptador de aplicación.
 * @returns {void} Sirve solicitudes hasta la señal de cierre.
 */
func Serve(handler http.Handler) {
	server := &http.Server{Addr: Env("ADDR", ":8080"), Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 12 * time.Second, IdleTimeout: 30 * time.Second, MaxHeaderBytes: 16 * 1024}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	done := make(chan error, 1)
	go func() { done <- server.ListenAndServe() }()
	select {
	case err := <-done:
		if !errors.Is(err, http.ErrServerClosed) {
			slog.Error("No se pudo iniciar HTTP")
			os.Exit(1)
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}
}

/**
 * Construye HTTP saliente con trazas y tiempos de espera acotados.
 *
 * @method Client
 * @param {time.Duration} timeout - Presupuesto total de llamada.
 * @returns {*http.Client} Cliente reutilizable con pool de conexiones.
 */
func Client(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout, Transport: sharedTransport}
}

var sharedTransport = otelhttp.NewTransport(&http.Transport{Proxy: http.ProxyFromEnvironment, DialContext: (&net.Dialer{Timeout: 500 * time.Millisecond, KeepAlive: 30 * time.Second}).DialContext, MaxIdleConns: 100, MaxIdleConnsPerHost: 10, IdleConnTimeout: 60 * time.Second, ResponseHeaderTimeout: 2 * time.Second})

/**
 * Conserva tokens y última reposición de una identidad del limitador en memoria.
 *
 * @method bucket
 * @returns {bucket} Representación tipada usada por este componente.
 */
type bucket struct {
	tokens float64
	last   time.Time
}

/**
 * Conserva buckets locales acotados con exclusión mutua y reloj de prueba.
 */
type Limiter struct {
	mu       sync.Mutex
	capacity float64
	rate     float64
	entries  map[string]bucket
	now      func() time.Time
}

/**
 * Crea un token bucket por identidad con capacidad de diez.
 *
 * @method NewLimiter
 * @returns {*Limiter} Limitador local con reposición de un token por segundo.
 */
func NewLimiter() *Limiter {
	return &Limiter{entries: map[string]bucket{}, now: time.Now, capacity: 10, rate: 1}
}

// Entries expire and the map has a hard bound; no identity/body is logged.
/**
 * Consume un token y limita la memoria de identidades.
 *
 * @method Allow
 * @param {string} key - Identidad técnica de cliente o hash de clave.
 * @returns {bool} Indica si la solicitud puede continuar.
 */
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	b, ok := l.entries[key]
	if !ok {
		if len(l.entries) >= 4096 {
			for k, v := range l.entries {
				if now.Sub(v.last) > time.Minute {
					delete(l.entries, k)
				}
			}
			if len(l.entries) >= 4096 {
				return false
			}
		}
		b = bucket{l.capacity, now}
	}
	b.tokens = min(l.capacity, b.tokens+now.Sub(b.last).Seconds()*l.rate)
	b.last = now
	allowed := b.tokens >= 1
	if allowed {
		b.tokens--
	}
	l.entries[key] = b
	return allowed
}

/**
 * Escribe JSON con tipo de contenido y política sin caché.
 *
 * @method JSON
 * @param {http.ResponseWriter} w - Respuesta HTTP.
 * @param {int} status - Estado a enviar.
 * @param {any} v - Valor serializable.
 * @returns {void} Escribe la respuesta JSON.
 */
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

/**
 * Construye una cuota local con reposición por segundo y ráfaga explícitas.
 *
 * @method NewRateLimiter
 * @param {float64} rate - Tokens repuestos por segundo para cada identidad.
 * @param {int} burst - Capacidad máxima de tokens disponibles en una ráfaga.
 * @returns {*Limiter} Limitador local por identidad con reposición y ráfaga configuradas.
 */
func NewRateLimiter(rate float64, burst int) *Limiter {
	if rate <= 0 || burst < 1 {
		panic("cuota inválida")
	}
	return &Limiter{entries: map[string]bucket{}, now: time.Now, capacity: float64(burst), rate: rate}
}
