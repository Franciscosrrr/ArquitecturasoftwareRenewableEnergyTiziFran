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

func Env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func Secret(key string) string {
	v := os.Getenv(key)
	if len(v) < 32 {
		slog.Error("Configuración ausente o demasiado corta", "variable", key)
		os.Exit(1)
	}
	return v
}
func Init(service string) func() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", service))
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
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = tp.Shutdown(ctx)
	}
}
func TraceID(ctx context.Context) string { return trace.SpanContextFromContext(ctx).TraceID().String() }

type requestIDKey struct{}

func RequestID(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey{}).(string)
	return value
}
func Error(c *gin.Context, status int, code, message string, details any) {
	if details == nil {
		details = []any{}
	}
	c.JSON(status, gin.H{"codigo": code, "mensaje": message, "traceId": TraceID(c.Request.Context()), "detalles": details})
}

var requestIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

func Router() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
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
func Instrument(service string, h http.Handler) http.Handler { return otelhttp.NewHandler(h, service) }
func Health(r *gin.Engine, token string, ready func(context.Context) error) {
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
func Client(timeout time.Duration) *http.Client {
	return &http.Client{Timeout: timeout, Transport: otelhttp.NewTransport(&http.Transport{Proxy: http.ProxyFromEnvironment, DialContext: (&net.Dialer{Timeout: 500 * time.Millisecond, KeepAlive: 30 * time.Second}).DialContext, MaxIdleConnsPerHost: 10, ResponseHeaderTimeout: 2 * time.Second})}
}

type bucket struct {
	tokens float64
	last   time.Time
}
type Limiter struct {
	mu      sync.Mutex
	entries map[string]bucket
}

func NewLimiter() *Limiter { return &Limiter{entries: map[string]bucket{}} }

// Entries expire and the map has a hard bound; no identity/body is logged.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
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
		b = bucket{10, now}
	}
	b.tokens = min(10, b.tokens+now.Sub(b.last).Seconds())
	b.last = now
	allowed := b.tokens >= 1
	if allowed {
		b.tokens--
	}
	l.entries[key] = b
	return allowed
}
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
