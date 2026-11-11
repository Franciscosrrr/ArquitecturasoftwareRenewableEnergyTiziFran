package transport

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"io"
	"mime"
	"net/http"
	"renewable.local/platform"
	"renewable.local/solar/internal/adapters/persistence"
	"renewable.local/solar/internal/application"
	"renewable.local/solar/internal/domain"
	"time"
)

/**
 * Construye las rutas solares y mantiene la salud separada del proveedor externo.
 *
 * @method NewHandler
 * @param {string} token - Secreto independiente de healthchecks.
 * @param {...application.Service} services - Caso de uso y adaptadores configurados.
 * @returns {http.Handler} API instrumentada; sin configuración conserva el esqueleto.
 */
func NewHandler(token string, services ...application.Service) http.Handler {
	return newHandler(token, services, nil, "")
}

/**
 * Construye el adaptador Solar con su repositorio real y rutas privadas.
 *
 * @method NewFullHandler
 * @param {string} token - Credencial técnica del servicio.
 * @param {application.Service} service - Nombre estable del servicio para trazas, logs y métricas.
 * @param {*persistence.Store} store - Repositorio canónico del componente, que conserva la propiedad de sus datos.
 * @param {string} internal - Secreto técnico usado para las operaciones entre servicios.
 * @returns {http.Handler} Adaptador HTTP con las rutas, controles y dependencias configurados.
 */
func NewFullHandler(token string, service application.Service, store *persistence.Store, internal string) http.Handler {
	return newHandler(token, []application.Service{service}, store, internal)
}

/**
 * Conecta rutas, salud y casos de uso con la configuración de Solar.
 *
 * @method newHandler
 * @param {string} token - Credencial técnica del servicio.
 * @param {[]application.Service} services - Casos de uso configurados para las rutas solares.
 * @param {*persistence.Store} store - Repositorio canónico del componente, que conserva la propiedad de sus datos.
 * @param {string} internal - Secreto técnico usado para las operaciones entre servicios.
 * @returns {http.Handler} Adaptador HTTP con las rutas, controles y dependencias configurados.
 */
func newHandler(token string, services []application.Service, store *persistence.Store, internal string) http.Handler {
	r := platform.Router()
	var ready func(context.Context) error
	if len(services) > 0 {
		ready = func(context.Context) error { return nil }
	}
	if store != nil {
		ready = store.Ready
		if len(services) > 0 {
			if supplier, ok := services[0].Supplier.(application.ModelSupplier); ok {
				registerPrivate(r, store, internal, supplier)
			} else {
				registerPrivate(r, store, internal)
			}
		} else {
			registerPrivate(r, store, internal)
		}
	}
	platform.Health(r, token, ready)
	if len(services) > 0 {
		limiter := platform.NewLimiter()
		r.POST("/api/v1/publico/solar/cotizaciones", func(c *gin.Context) {
			if !limiter.Allow(c.ClientIP()) {
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
			decoder := json.NewDecoder(c.Request.Body)
			decoder.DisallowUnknownFields()
			var input application.Request
			err := decoder.Decode(&input)
			if errors.Is(err, domain.ErrValidation) {
				platform.Error(c, 422, "VALIDACION", "Los parámetros solares deben ser números con hasta seis decimales.", nil)
				return
			}
			var maxErr *http.MaxBytesError
			if errors.As(err, &maxErr) {
				platform.Error(c, 413, "CUERPO_DEMASIADO_GRANDE", "La solicitud supera 256 KiB.", nil)
				return
			}
			if err != nil {
				platform.Error(c, 400, "FORMATO_INVALIDO", "JSON inválido o campos desconocidos.", nil)
				return
			}
			err = decoder.Decode(new(any))
			if errors.As(err, &maxErr) {
				platform.Error(c, 413, "CUERPO_DEMASIADO_GRANDE", "La solicitud supera 256 KiB.", nil)
				return
			}
			if err != io.EOF {
				platform.Error(c, 400, "FORMATO_INVALIDO", "Se requiere un único objeto JSON.", nil)
				return
			}
			ctx, cancel := context.WithTimeout(c.Request.Context(), 4*time.Second)
			defer cancel()
			result, err := services[0].Execute(ctx, input, platform.TraceID(ctx))
			switch {
			case errors.Is(err, domain.ErrValidation):
				platform.Error(c, 422, "VALIDACION", "Revise los equipos, el consumo positivo y los parámetros solares.", nil)
			case errors.Is(err, domain.ErrStock):
				platform.Error(c, 409, "STOCK_INSUFICIENTE", "El vendedor no tiene stock suficiente para esta instalación.", nil)
			case errors.Is(err, domain.ErrContract):
				platform.Error(c, 502, "CONTRATO_PROVEEDOR_INVALIDO", "No se pudo validar la oferta del vendedor.", nil)
			case err != nil:
				platform.Error(c, 503, "PROVEEDOR_NO_DISPONIBLE", "No pudimos obtener la cotización. Tu estimación de consumo sigue disponible; volvé a intentar.", nil)
			default:
				c.JSON(200, result)
			}
		})
	}
	r.NoRoute(func(c *gin.Context) { c.JSON(501, gin.H{"mensaje": "Ruta no implementada"}) })
	return platform.Instrument("solar", r)
}
