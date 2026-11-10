package transport

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"renewable.local/gateway/internal/balancing"
	"renewable.local/platform"
	"strings"
	"time"
)

/**
 * Enruta solicitudes, valida sesión y distribuye Solar entre réplicas saludables.
 *
 * @method fullHandler
 * @param {string} token - Credencial técnica del servicio.
 * @param {Config} cfg - Configuración de infraestructura del adaptador.
 * @returns {http.Handler} Resultado de la operación; error si no puede completarse.
 */
func fullHandler(token string, cfg Config) http.Handler {
	r := platform.Router()
	limiter := platform.NewLimiter()
	privateLimiter := platform.NewRateLimiter(5, 20)
	upstreamTransport := otelhttp.NewTransport(&http.Transport{Proxy: http.ProxyFromEnvironment, DialContext: (&net.Dialer{Timeout: 500 * time.Millisecond, KeepAlive: 30 * time.Second}).DialContext, ResponseHeaderTimeout: 6 * time.Second, MaxIdleConns: 100, MaxIdleConnsPerHost: 20, IdleConnTimeout: 60 * time.Second})
	var balance *balancing.Balancer
	urls := []string{}
	if cfg.SolarURLs != "" {
		for _, u := range strings.Split(cfg.SolarURLs, ",") {
			if strings.TrimSpace(u) != "" {
				urls = append(urls, strings.TrimSpace(u))
			}
		}
	} else if cfg.SolarURL != "" {
		urls = []string{cfg.SolarURL}
	}
	if len(urls) > 0 && !cfg.CapabilityOnly {
		balance = balancing.New(context.Background(), urls, token)
	}
	platform.Health(r, token, func(ctx context.Context) error {
		targets := []string{cfg.ConsumoURL}
		if !cfg.CapabilityOnly {
			targets = append(targets, cfg.CatalogURL, cfg.IdentityURL)
		}
		for _, base := range targets {
			if base == "" {
				continue
			}
			req, err := http.NewRequestWithContext(ctx, "GET", base+"/health/ready", nil)
			if err != nil {
				return err
			}
			req.Header.Set("Authorization", "Bearer "+token)
			res, err := platform.Client(time.Second).Do(req)
			if err != nil {
				return err
			}
			res.Body.Close()
			if res.StatusCode != 200 {
				return fmt.Errorf("dependencia no disponible")
			}
		}
		if !cfg.CapabilityOnly && balance != nil && !balance.Ready() {
			return fmt.Errorf("Solar no disponible")
		}
		return nil
	})
	authenticate := func(c *gin.Context) {
		supplied := c.GetHeader("Authorization")
		if !strings.HasPrefix(supplied, "Bearer ") || cfg.IdentityURL == "" {
			platform.Error(c, 401, "NO_AUTENTICADO", "Se requiere iniciar sesión.", nil)
			c.Abort()
			return
		}
		raw, _ := json.Marshal(map[string]string{"token": strings.TrimPrefix(supplied, "Bearer ")})
		ctx, cancel := context.WithTimeout(c.Request.Context(), time.Second)
		defer cancel()
		req, err := http.NewRequestWithContext(ctx, "POST", cfg.IdentityURL+"/internal/v1/identidad/validar", bytes.NewReader(raw))
		if err != nil {
			platform.Error(c, 503, "SERVICIO_NO_DISPONIBLE", "Identidad no disponible.", nil)
			c.Abort()
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+cfg.InternalToken)
		res, err := platform.Client(time.Second).Do(req)
		if err != nil {
			platform.Error(c, 503, "SERVICIO_NO_DISPONIBLE", "Identidad no disponible.", nil)
			c.Abort()
			return
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			status := res.StatusCode
			if status != 401 && status != 403 {
				status = 503
			}
			code := "NO_AUTENTICADO"
			if status == 503 {
				code = "SERVICIO_NO_DISPONIBLE"
			}
			platform.Error(c, status, code, "La sesión no está vigente o identidad no está disponible.", nil)
			c.Abort()
			return
		}
		var actor platform.Actor
		if json.NewDecoder(io.LimitReader(res.Body, 4096)).Decode(&actor) != nil || actor.ID == "" {
			platform.Error(c, 503, "SERVICIO_NO_DISPONIBLE", "Respuesta de identidad inválida.", nil)
			c.Abort()
			return
		}
		c.Set("actor", actor)
		c.Set("access", supplied)
		c.Next()
	}
	proxy := func(base string, private, authCookies, solar bool) gin.HandlerFunc {
		return func(c *gin.Context) {
			target := base
			if solar {
				var err error
				if balance == nil {
					platform.Error(c, 503, "SERVICIO_NO_DISPONIBLE", "Solar no disponible.", nil)
					return
				}
				target, err = balance.Next()
				if err != nil {
					platform.Error(c, 503, "SERVICIO_NO_DISPONIBLE", "No hay réplicas Solar saludables.", nil)
					return
				}
			}
			key := "ip:" + c.ClientIP()
			if private {
				key = "usuario:" + c.GetString("actorID")
				if a, ok := c.Get("actor"); ok {
					key = "usuario:" + a.(platform.Actor).ID
				}
			}
			if c.FullPath() == "/api/v1/estimaciones-consumo" {
				sum := sha256.Sum256([]byte(c.GetHeader("X-API-Key")))
				key = fmt.Sprintf("m2m:%x", sum[:])
			}
			activeLimiter := limiter
			if private {
				activeLimiter = privateLimiter
			}
			if !activeLimiter.Allow(key) {
				c.Header("Retry-After", "1")
				platform.Error(c, 429, "LIMITE_DE_TRAFICO", "Se alcanzó el límite de solicitudes.", nil)
				return
			}
			parsed, err := url.Parse(target)
			if err != nil || parsed.Host == "" {
				platform.Error(c, 503, "SERVICIO_NO_DISPONIBLE", "Destino no disponible.", nil)
				return
			}
			p := &httputil.ReverseProxy{Rewrite: func(pr *httputil.ProxyRequest) {
				pr.SetURL(parsed)
				pr.SetXForwarded()
				for _, name := range []string{"Authorization", "Cookie", "X-Actor-ID", "X-Actor-Role", "X-Access-Token", "X-Cache-Mode"} {
					pr.Out.Header.Del(name)
				}
				if authCookies {
					pr.Out.Header.Set("Cookie", pr.In.Header.Get("Cookie"))
				}
				if private {
					a, _ := c.Get("actor")
					actor := a.(platform.Actor)
					pr.Out.Header.Set("Authorization", "Bearer "+cfg.InternalToken)
					pr.Out.Header.Set("X-Actor-ID", actor.ID)
					pr.Out.Header.Set("X-Actor-Role", actor.Role)
					pr.Out.Header.Set("X-Access-Token", c.GetString("access"))
				}
			}, Transport: upstreamTransport, ErrorHandler: func(w http.ResponseWriter, req *http.Request, err error) {
				platform.JSON(w, 503, map[string]any{"codigo": "SERVICIO_NO_DISPONIBLE", "mensaje": "Servicio no disponible. Reintente más tarde.", "traceId": platform.TraceID(req.Context()), "detalles": []any{}})
			}}
			budget := 8 * time.Second
			if c.FullPath() == "/api/v1/estimaciones-consumo" {
				budget = 4500 * time.Millisecond
			}
			if strings.HasSuffix(c.FullPath(), "/reindexar") {
				budget = 35 * time.Second
			}
			ctx, cancel := context.WithTimeout(c.Request.Context(), budget)
			defer cancel()
			c.Header("X-Gateway-Instance-ID", c.Writer.Header().Get("X-Instance-ID"))
			c.Writer.Header().Del("X-Instance-ID")
			p.ServeHTTP(c.Writer, c.Request.WithContext(ctx))
		}
	}
	r.POST("/api/v1/estimaciones-consumo", proxy(cfg.ConsumoURL, false, false, false))
	if !cfg.CapabilityOnly {
		for _, path := range []string{"/api/v1/publico/consumo", "/api/v1/publico/consumo-catalogo"} {
			r.POST(path, proxy(cfg.ConsumoURL, false, false, false))
		}
		r.GET("/api/v1/publico/electrodomesticos", proxy(cfg.CatalogURL, false, false, false))
		r.GET("/api/v1/publico/electrodomesticos/:id", proxy(cfg.CatalogURL, false, false, false))
		for _, path := range []string{"/api/v1/publico/solar/cotizaciones", "/api/v1/publico/recomendaciones-solares"} {
			r.POST(path, proxy("", false, false, true))
		}
		r.GET("/api/v1/publico/paneles", proxy("", false, false, true))
		r.GET("/api/v1/publico/paneles/busqueda", proxy("", false, false, true))
		for _, path := range []string{"registro", "login", "refresh", "logout"} {
			r.POST("/api/v1/auth/"+path, proxy(cfg.IdentityURL, false, true, false))
		}
		protected := r.Group("/api/v1", authenticate)
		protected.POST("/admin/paneles/reindexar", proxy("", true, false, true))
		for _, resource := range []string{"/lugares", "/configuraciones"} {
			protected.PATCH(resource+"/:id/archivo", proxy(cfg.ConsumoURL, true, false, false))
			protected.DELETE(resource+"/:id", proxy(cfg.ConsumoURL, true, false, false))
		}
		protected.GET("/perfil", proxy(cfg.IdentityURL, true, false, false))
		for _, entry := range []struct{ method, path string }{{"GET", "/lugares"}, {"POST", "/lugares"}, {"PUT", "/lugares/:id"}, {"GET", "/configuraciones"}, {"POST", "/configuraciones"}, {"GET", "/configuraciones/:id"}, {"PUT", "/configuraciones/:id"}, {"POST", "/configuraciones/:id/duplicar"}, {"POST", "/configuraciones/:id/evaluaciones"}, {"GET", "/evaluaciones"}, {"GET", "/evaluaciones/:id"}, {"POST", "/evaluaciones/:id/estudios"}} {
			protected.Handle(entry.method, entry.path, proxy(cfg.ConsumoURL, true, false, false))
		}
		for _, entry := range []struct{ method, path string }{{"GET", "/estudios"}, {"GET", "/estudios/:id"}, {"GET", "/admin/paneles"}, {"POST", "/admin/paneles"}, {"PUT", "/admin/paneles/:id"}, {"GET", "/admin/mensajeria"}, {"POST", "/admin/mensajeria/reprocesar"}} {
			protected.Handle(entry.method, entry.path, proxy("", true, false, true))
		}
		for _, entry := range []struct{ method, path string }{{"GET", "/admin/electrodomesticos"}, {"POST", "/admin/electrodomesticos"}, {"PUT", "/admin/electrodomesticos/:id"}, {"POST", "/admin/catalogo/reindexar"}} {
			protected.Handle(entry.method, entry.path, proxy(cfg.CatalogURL, true, false, false))
		}
		protected.GET("/admin/usuarios", proxy(cfg.IdentityURL, true, false, false))
		protected.PUT("/admin/usuarios/:id/rol", proxy(cfg.IdentityURL, true, false, false))
	}
	for name, folder := range map[string]string{"openapi.json": "v1", "guest-openapi.json": "guest-v1", "catalogo-openapi.json": "catalog-v1", "solar-openapi.json": "solar-v1", "paneles-openapi.json": "panel-search-v1", "privado-openapi.json": "private-v1"} {
		if cfg.CapabilityOnly && folder != "v1" {
			continue
		}
		file := filepath.Join(cfg.DocsDir, folder, "openapi.json")
		r.GET("/"+name, func(c *gin.Context) { c.File(file) })
	}
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") || strings.HasPrefix(c.Request.URL.Path, "/internal/") {
			if c.GetHeader("Authorization") == "" && !strings.HasPrefix(c.Request.URL.Path, "/api/v1/publico/") {
				platform.Error(c, 401, "NO_AUTENTICADO", "Se requiere iniciar sesión.", nil)
				return
			}
			c.Status(404)
			return
		}
		if cfg.CapabilityOnly {
			c.Status(404)
			return
		}
		clean := filepath.Clean("/" + c.Request.URL.Path)
		file := filepath.Join(cfg.WebDir, clean)
		if info, err := os.Stat(file); err == nil && !info.IsDir() {
			c.File(file)
			return
		}
		if c.Request.Method == "GET" && c.Request.URL.Path == "/" {
			c.File(filepath.Join(cfg.WebDir, "index.html"))
			return
		}
		c.Status(404)
	})
	return platform.Instrument("gateway", r)
}
