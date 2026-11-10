package platform

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"strconv"
	"time"
)

var requests = promauto.NewCounterVec(prometheus.CounterOpts{Name: "renewable_http_requests_total", Help: "Solicitudes por ruta normalizada y estado."}, []string{"route", "method", "status"})
var latency = promauto.NewHistogramVec(prometheus.HistogramOpts{Name: "renewable_http_duration_seconds", Help: "Duración HTTP.", Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2, 4, 8}}, []string{"route"})
var Operations = promauto.NewCounterVec(prometheus.CounterOpts{Name: "renewable_operations_total", Help: "Resultados técnicos y de dominio sin identificadores personales."}, []string{"component", "outcome"})
var State = promauto.NewGaugeVec(prometheus.GaugeOpts{Name: "renewable_state", Help: "Estado y pendientes de componentes."}, []string{"component", "state"})

/**
 * Mide duración, estado y ruta normalizada sin usar identificadores personales como etiquetas.
 *
 * @method MetricsMiddleware
 * @param {*gin.Context} c - Solicitud HTTP en curso.
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func MetricsMiddleware(c *gin.Context) {
	start := time.Now()
	c.Header("X-Instance-ID", Env("INSTANCE_ID", "local"))
	c.Next()
	route := c.FullPath()
	if route == "" {
		route = "desconocida"
	}
	if route == "/metrics" {
		return
	}
	requests.WithLabelValues(route, c.Request.Method, strconv.Itoa(c.Writer.Status())).Inc()
	latency.WithLabelValues(route).Observe(time.Since(start).Seconds())
}

/**
 * Expone el scraping Prometheus solo con credencial técnica válida.
 *
 * @method Metrics
 * @param {*gin.Engine} r - Router o receptor de operaciones HTTP.
 * @param {string} token - Credencial técnica del servicio.
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func Metrics(r *gin.Engine, token string) {
	h := promhttp.Handler()
	r.GET("/metrics", func(c *gin.Context) {
		if !CredentialOK(c.GetHeader("Authorization"), "Bearer "+token) {
			c.Status(http.StatusUnauthorized)
			return
		}
		h.ServeHTTP(c.Writer, c.Request)
	})
}
