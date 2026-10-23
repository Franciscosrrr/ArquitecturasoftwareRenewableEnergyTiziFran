package transport

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"renewable.local/platform"
	"strings"
	"time"
)

type Config struct{ ConsumoURL, CatalogURL, WebDir, DocsDir string }

func NewHandler(token string, configs ...Config) http.Handler {
	r := platform.Router()
	var cfg Config
	var ready func(context.Context) error
	if len(configs) > 0 {
		cfg = configs[0]
		ready = func(ctx context.Context) error {
			for _, base := range []string{cfg.ConsumoURL, cfg.CatalogURL} {
				req, _ := http.NewRequestWithContext(ctx, "GET", base+"/health/ready", nil)
				req.Header.Set("Authorization", "Bearer "+token)
				res, err := platform.Client(time.Second).Do(req)
				if err != nil {
					return err
				}
				res.Body.Close()
				if res.StatusCode != 200 {
					return fmt.Errorf("servicio no disponible")
				}
			}
			return nil
		}
	}
	platform.Health(r, token, ready)
	limiter := platform.NewLimiter()
	proxy := func(targetURL string) gin.HandlerFunc {
		target, err := url.Parse(targetURL)
		if err != nil {
			panic("URL de servicio inválida")
		}
		p := &httputil.ReverseProxy{Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			pr.SetXForwarded()
			pr.Out.Header.Del("Authorization")
			pr.Out.Header.Del("Cookie")
		}, Transport: otelhttp.NewTransport(&http.Transport{Proxy: http.ProxyFromEnvironment, DialContext: (&net.Dialer{Timeout: 500 * time.Millisecond, KeepAlive: 30 * time.Second}).DialContext, ResponseHeaderTimeout: 6 * time.Second, MaxIdleConnsPerHost: 20}), ErrorHandler: func(w http.ResponseWriter, req *http.Request, err error) {
			platform.JSON(w, 503, map[string]any{"codigo": "SERVICIO_NO_DISPONIBLE", "mensaje": "Servicio no disponible. Reintente más tarde.", "traceId": platform.TraceID(req.Context()), "detalles": []any{}})
		}}
		return func(c *gin.Context) {
			key := c.ClientIP()
			if c.FullPath() == "/api/v1/estimaciones-consumo" {
				sum := sha256.Sum256([]byte(c.GetHeader("X-API-Key")))
				key = fmt.Sprintf("m2m:%x", sum[:])
			}
			if !limiter.Allow(key) {
				c.Header("Retry-After", "1")
				platform.Error(c, 429, "LIMITE_DE_TRAFICO", "Se alcanzó el límite de solicitudes.", nil)
				return
			}
			ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
			defer cancel()
			p.ServeHTTP(c.Writer, c.Request.WithContext(ctx))
		}
	}
	if len(configs) > 0 {
		for _, path := range []string{"/api/v1/publico/consumo", "/api/v1/publico/consumo-catalogo", "/api/v1/estimaciones-consumo"} {
			r.POST(path, proxy(cfg.ConsumoURL))
		}
		r.GET("/api/v1/publico/electrodomesticos", proxy(cfg.CatalogURL))
		r.GET("/api/v1/publico/electrodomesticos/:id", proxy(cfg.CatalogURL))
		r.GET("/openapi.json", func(c *gin.Context) { c.File(filepath.Join(cfg.DocsDir, "v1", "openapi.json")) })
		r.GET("/guest-openapi.json", func(c *gin.Context) { c.File(filepath.Join(cfg.DocsDir, "guest-v1", "openapi.json")) })
		r.GET("/catalogo-openapi.json", func(c *gin.Context) { c.File(filepath.Join(cfg.DocsDir, "catalog-v1", "openapi.json")) })
	}
	r.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") || strings.HasPrefix(c.Request.URL.Path, "/internal/") {
			if c.GetHeader("Authorization") == "" && !strings.HasPrefix(c.Request.URL.Path, "/api/v1/publico/") {
				platform.Error(c, 401, "NO_AUTENTICADO", "Se requiere iniciar sesión para esta operación.", nil)
				return
			}
			c.JSON(501, gin.H{"mensaje": "Funcionalidad fuera del alcance operativo de entrega 2"})
			return
		}
		if len(configs) == 0 {
			c.JSON(501, gin.H{"mensaje": "Negocio no implementado"})
			return
		}
		clean := filepath.Clean("/" + c.Request.URL.Path)
		file := filepath.Join(cfg.WebDir, clean)
		if info, err := os.Stat(file); err == nil && !info.IsDir() {
			c.File(file)
			return
		}
		if c.Request.Method == http.MethodGet && c.Request.URL.Path == "/" {
			c.File(filepath.Join(cfg.WebDir, "index.html"))
			return
		}
		c.Status(404)
	})
	return platform.Instrument("gateway", r)
}
