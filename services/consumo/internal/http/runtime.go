package transport

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"io"
	"mime"
	"net/http"
	"renewable.local/consumo/internal/application"
	"renewable.local/consumo/internal/private"
	"renewable.local/platform"
)

/**
 * Configura los destinos, credenciales y dependencias del adaptador HTTP de este servicio.
 *
 * @method Config
 * @returns {Config} Representación tipada usada por este componente.
 */
type Config struct {
	APIKey, DeniedKey string
	Catalog           application.Resolver
	Private           *private.Service
	InternalToken     string
}

/**
 * Conecta rutas HTTP, salud y dependencias de negocio del servicio.
 *
 * @method NewHandler
 * @param {string} token - Credencial técnica del servicio.
 * @param {...Config} configs - Configuración opcional de dependencias y rutas del adaptador HTTP.
 * @returns {http.Handler} Adaptador HTTP con las rutas, controles y dependencias configurados.
 */
func NewHandler(token string, configs ...Config) http.Handler {
	r := platform.Router()
	var cfg Config
	var ready func(context.Context) error
	if len(configs) > 0 {
		cfg = configs[0]
		ready = func(context.Context) error { return nil }
		if cfg.Private != nil {
			ready = cfg.Private.Store.DB.PingContext
		}
	}
	platform.Health(r, token, ready)
	if cfg.Private != nil {
		cfg.Private.Register(r, cfg.InternalToken)
	}
	limiter := platform.NewLimiter()
	handler := func(c *gin.Context) {
		if c.FullPath() == "/api/v1/estimaciones-consumo" {
			supplied := sha256.Sum256([]byte(c.GetHeader("X-API-Key")))
			expected := sha256.Sum256([]byte(cfg.APIKey))
			denied := sha256.Sum256([]byte(cfg.DeniedKey))
			if len(cfg.DeniedKey) >= 32 && subtle.ConstantTimeCompare(supplied[:], denied[:]) == 1 {
				platform.Error(c, 403, "SIN_PERMISO", "El cliente no tiene permiso para estimar consumos.", nil)
				return
			}
			if len(cfg.APIKey) < 32 || subtle.ConstantTimeCompare(supplied[:], expected[:]) != 1 {
				platform.Error(c, 401, "NO_AUTENTICADO", "Se requiere una credencial M2M válida.", nil)
				return
			}
		}
		identity := c.ClientIP() + c.FullPath()
		if c.FullPath() == "/api/v1/estimaciones-consumo" {
			identity = "m2m"
		}
		if !limiter.Allow(identity) {
			c.Header("Retry-After", "1")
			platform.Error(c, 429, "LIMITE_DE_TRAFICO", "Se alcanzó el límite de solicitudes.", nil)
			return
		}
		media, _, _ := mime.ParseMediaType(c.GetHeader("Content-Type"))
		if media != "application/json" {
			platform.Error(c, 415, "TIPO_CONTENIDO_NO_SOPORTADO", "Utilice Content-Type application/json.", nil)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 262144)
		raw, err := io.ReadAll(c.Request.Body)
		if err != nil {
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				platform.Error(c, 413, "CUERPO_DEMASIADO_GRANDE", "La solicitud supera 256 KiB.", nil)
			} else {
				platform.Error(c, 400, "FORMATO_INVALIDO", "Cuerpo incompleto.", nil)
			}
			return
		}
		if !json.Valid(raw) {
			platform.Error(c, 400, "FORMATO_INVALIDO", "El cuerpo no contiene JSON válido.", nil)
			return
		}
		var result application.Result
		if c.FullPath() == "/api/v1/publico/consumo-catalogo" {
			if cfg.Catalog == nil {
				platform.Error(c, 503, "SERVICIO_NO_DISPONIBLE", "Catálogo no disponible.", nil)
				return
			}
			result, err = application.CalculateCatalog(c.Request.Context(), raw, platform.TraceID(c.Request.Context()), cfg.Catalog)
		} else {
			result, err = application.Calculate(c.Request.Context(), raw, platform.TraceID(c.Request.Context()))
		}
		if err != nil {
			var validation *application.ValidationError
			if errors.As(err, &validation) {
				platform.Error(c, 422, "VALIDACION", "Los datos de consumo no cumplen las reglas de la operación.", []application.Detail{validation.Detail})
			} else {
				platform.Error(c, 503, "SERVICIO_NO_DISPONIBLE", "Catálogo no disponible. Reintente más tarde.", nil)
			}
			return
		}
		c.JSON(200, result)
	}
	r.POST("/api/v1/estimaciones-consumo", handler)
	r.POST("/api/v1/publico/consumo", handler)
	r.POST("/api/v1/publico/consumo-catalogo", handler)
	r.NoRoute(func(c *gin.Context) {
		c.JSON(501, gin.H{"mensaje": "Funcionalidad fuera del alcance operativo de entrega 2"})
	})
	return platform.Instrument("consumo", r)
}
