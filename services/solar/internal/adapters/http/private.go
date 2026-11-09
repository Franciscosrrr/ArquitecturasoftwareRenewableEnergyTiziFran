package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"io"
	"mime"
	"net/http"
	"regexp"
	"renewable.local/platform"
	"renewable.local/solar/internal/adapters/persistence"
	"renewable.local/solar/internal/adapters/search"
	"renewable.local/solar/internal/domain"
	"renewable.local/solar/internal/worker"
	"strings"
	"time"
)

/**
 * Registra catálogos, estudios propios y mantenimiento administrativo de Solar.
 *
 * @method registerPrivate
 * @param {*gin.Engine} r - Router o receptor de operaciones HTTP.
 * @param {*persistence.Store} store - Valor de store requerido por la operación.
 * @param {string} token - Credencial técnica del servicio.
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func registerPrivate(r *gin.Engine, store *persistence.Store, token string) {
	panelSearch := search.New(store)
	go panelSearch.Run(context.Background())
	r.GET("/api/v1/publico/paneles/busqueda", func(c *gin.Context) {
		query, err := search.Parse(c.Request.URL.Query())
		if err != nil {
			platform.Error(c, 422, "VALIDACION", err.Error(), nil)
			return
		}
		page, source, err := panelSearch.List(c.Request.Context(), query)
		if err != nil {
			platform.Error(c, 503, "SERVICIO_NO_DISPONIBLE", "No pudimos consultar los paneles. Volvé a intentar.", nil)
			return
		}
		c.Header("X-Panel-Source", source)
		c.JSON(200, page)
	})
	r.GET("/api/v1/publico/paneles", func(c *gin.Context) {
		items, err := store.Panels(c.Request.Context(), false)
		if err != nil {
			platform.Error(c, 503, "SERVICIO_NO_DISPONIBLE", "Paneles no disponibles.", nil)
			return
		}
		c.JSON(200, items)
	})
	api := r.Group("/api/v1", platform.Internal(token, false))
	api.GET("/estudios", func(c *gin.Context) {
		items, err := store.List(c.Request.Context(), platform.ActorID(c), c.Query("evaluacionId"))
		if err != nil {
			platform.Error(c, 503, "SERVICIO_NO_DISPONIBLE", "Estudios no disponibles.", nil)
			return
		}
		c.JSON(200, items)
	})
	api.GET("/estudios/:id", func(c *gin.Context) {
		item, err := store.Get(c.Request.Context(), platform.ActorID(c), c.Param("id"))
		if errors.Is(err, mongo.ErrNoDocuments) {
			platform.Error(c, 404, "NO_ENCONTRADO", "Estudio pendiente o inexistente.", nil)
			return
		}
		if err != nil {
			platform.Error(c, 503, "SERVICIO_NO_DISPONIBLE", "Estudio no disponible.", nil)
			return
		}
		c.JSON(200, item)
	})
	admin := r.Group("/api/v1/admin", platform.Internal(token, true))
	admin.POST("/paneles/reindexar", func(c *gin.Context) {
		if err := store.ReindexPanels(c.Request.Context()); err != nil {
			platform.Error(c, 503, "SERVICIO_NO_DISPONIBLE", "No se pudo actualizar la búsqueda de paneles.", nil)
			return
		}
		c.JSON(202, gin.H{"estado": "PENDIENTE"})
	})
	admin.GET("/paneles", func(c *gin.Context) {
		items, err := store.Panels(c.Request.Context(), true)
		if err != nil {
			platform.Error(c, 503, "SERVICIO_NO_DISPONIBLE", "Catálogo no disponible.", nil)
			return
		}
		c.JSON(200, items)
	})
	save := func(c *gin.Context) {
		var p domain.Panel
		if !platform.Decode(c, &p) {
			return
		}
		if c.Request.Method == "PUT" {
			p.ID = c.Param("id")
		}
		if !regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`).MatchString(p.ID) || strings.TrimSpace(p.Name) == "" || len(p.Name) > 120 || p.Watts < 1 || p.Watts > 2000 || p.AreaMilli < 1 || p.AreaMilli > 100000 || p.UnitCents < 0 || p.UnitCents > 1000000000 || (p.Currency != "ARS" && p.Currency != "USD") || (c.Request.Method == "PUT" && p.Version < 1) {
			platform.Error(c, 422, "VALIDACION", "Ficha de panel inválida.", nil)
			return
		}
		saved, err := store.SavePanel(c.Request.Context(), p, c.Request.Method == "POST")
		if errors.Is(err, persistence.ErrConflict) {
			platform.Error(c, 409, "CONFLICTO", "Panel existente o versión obsoleta.", nil)
			return
		}
		if err != nil {
			platform.Error(c, 503, "SERVICIO_NO_DISPONIBLE", "Panel no guardado.", nil)
			return
		}
		status := 200
		if c.Request.Method == "POST" {
			status = 201
		}
		c.JSON(status, saved)
	}
	admin.POST("/paneles", save)
	admin.PUT("/paneles/:id", save)
	admin.GET("/mensajeria", func(c *gin.Context) {
		b, err := platform.OpenBus()
		if err != nil {
			platform.Error(c, 503, "SERVICIO_NO_DISPONIBLE", "Broker no disponible.", nil)
			return
		}
		defer b.Close()
		counts := map[string]int{}
		for _, suffix := range []string{"", ".retry", ".dlq"} {
			q, err := b.Ch.QueueInspect("solar.estudios" + suffix)
			if err != nil {
				platform.Error(c, 503, "SERVICIO_NO_DISPONIBLE", "Cola no disponible.", nil)
				return
			}
			counts[q.Name] = q.Messages
			platform.State.WithLabelValues("solar.estudios", suffix+"pendientes").Set(float64(q.Messages))
		}
		c.JSON(200, counts)
	})
	admin.POST("/mensajeria/reprocesar", func(c *gin.Context) {
		var input struct {
			Limit int `json:"limite"`
		}
		if !platform.Decode(c, &input) {
			return
		}
		if input.Limit < 1 || input.Limit > 50 {
			platform.Error(c, 422, "VALIDACION", "Límite entre 1 y 50.", nil)
			return
		}
		b, err := platform.OpenBus()
		if err != nil {
			platform.Error(c, 503, "SERVICIO_NO_DISPONIBLE", "Broker no disponible.", nil)
			return
		}
		defer b.Close()
		replayed := 0
		invalid := 0
		for i := 0; i < input.Limit; i++ {
			d, ok, err := b.Ch.Get("solar.estudios.dlq", false)
			if err != nil {
				platform.Error(c, 503, "SERVICIO_NO_DISPONIBLE", "No se pudo revisar DLQ.", nil)
				return
			}
			if !ok {
				break
			}
			if _, err = worker.Parse(d.Body); err != nil {
				// Se conserva sin ACK hasta cerrar este canal; así no bloquea mensajes válidos posteriores.
				invalid++
				continue
			}
			ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
			err = b.Publish(ctx, "renewable.events", "consumo.evaluado.v1", d.Body, nil, "")
			cancel()
			if err != nil {
				_ = d.Nack(false, true)
				platform.Error(c, 503, "SERVICIO_NO_DISPONIBLE", "No se confirmó la republicación.", nil)
				return
			}
			_ = d.Ack(false)
			replayed++
		}
		platform.Operations.WithLabelValues("dlq", "reprocesado").Add(float64(replayed))
		c.JSON(200, gin.H{"republicados": replayed, "invalidosConservados": invalid})
	})
	guestLimit := platform.NewRateLimiter(10.0/60, 2)
	r.POST("/api/v1/publico/recomendaciones-solares", func(c *gin.Context) {
		if !guestLimit.Allow(c.ClientIP()) {
			c.Header("Retry-After", "6")
			platform.Error(c, 429, "LIMITE_DE_TRAFICO", "Se alcanzó la cuota anónima solar.", nil)
			return
		}
		var input struct {
			Daily    domain.DecimalNumber  `json:"promedioDiarioKWh"`
			Coverage domain.DecimalNumber  `json:"coberturaObjetivo"`
			Yield    domain.DecimalNumber  `json:"rendimientoGlobal"`
			Area     *domain.DecimalNumber `json:"superficieUtilM2"`
			Resource struct {
				Type      string               `json:"tipo"`
				HSP       domain.DecimalNumber `json:"hsp"`
				Source    string               `json:"fuente"`
				Latitude  *float64             `json:"latitud"`
				Longitude *float64             `json:"longitud"`
			} `json:"recursoSolar"`
		}
		media, _, mediaErr := mime.ParseMediaType(c.GetHeader("Content-Type"))
		if mediaErr != nil || media != "application/json" {
			platform.Error(c, 415, "TIPO_CONTENIDO_NO_SOPORTADO", "Se requiere application/json.", nil)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 262144)
		raw, readErr := io.ReadAll(c.Request.Body)
		if readErr != nil {
			platform.Error(c, 413, "CUERPO_DEMASIADO_GRANDE", "Cuerpo demasiado grande.", nil)
			return
		}
		if !json.Valid(raw) {
			platform.Error(c, 400, "FORMATO_INVALIDO", "JSON inválido.", nil)
			return
		}
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&input) != nil {
			platform.Error(c, 422, "VALIDACION", "Solicitud incompatible.", nil)
			return
		}
		if input.Resource.Type != "MANUAL" && input.Resource.Type != "UBICACION" {
			platform.Error(c, 422, "VALIDACION", "Tipo de recurso inválido.", nil)
			return
		}
		if input.Resource.Type == "MANUAL" && (input.Resource.Latitude != nil || input.Resource.Longitude != nil) {
			platform.Error(c, 422, "VALIDACION", "Recurso manual no admite coordenadas.", nil)
			return
		}
		if input.Resource.Type == "UBICACION" && (input.Resource.Latitude == nil || input.Resource.Longitude == nil || *input.Resource.Latitude < -90 || *input.Resource.Latitude > 90 || *input.Resource.Longitude < -180 || *input.Resource.Longitude > 180 || input.Resource.Source != "" || input.Resource.HSP != "") {
			platform.Error(c, 422, "VALIDACION", "Recurso por ubicación inválido.", nil)
			return
		}
		if input.Resource.Type != "MANUAL" {
			platform.Error(c, 503, "SERVICIO_NO_DISPONIBLE", "No hay una fuente de irradiación por ubicación acordada; indique un recurso manual explícito.", nil)
			return
		}
		if input.Resource.Source == "" || len(input.Resource.Source) > 200 {
			platform.Error(c, 422, "VALIDACION", "Indique la fuente del supuesto manual.", nil)
			return
		}
		panels, err := store.Panels(c.Request.Context(), false)
		if err != nil {
			platform.Error(c, 503, "SERVICIO_NO_DISPONIBLE", "Catálogo solar no disponible.", nil)
			return
		}
		result, err := domain.Recommend(json.Number(input.Daily), domain.StudyOptions{Sun: input.Resource.HSP, Yield: input.Yield, Coverage: input.Coverage, Area: input.Area}, panels)
		if err != nil {
			platform.Error(c, 422, "VALIDACION", "Supuestos o dimensionamiento fuera de rango.", nil)
			return
		}
		total := len(result.Candidates)
		if total > 20 {
			result.Candidates = result.Candidates[:20]
		}
		items := []gin.H{}
		for _, a := range result.Candidates {
			items = append(items, gin.H{"panelId": a.Panel.ID, "modelo": a.Panel.Name, "versionFicha": a.Panel.Version, "cantidad": a.Quantity, "generacionDiariaKWh": a.Generation, "coberturaAlcanzada": a.Coverage, "superficieRequeridaM2": a.Area, "cumpleObjetivo": a.Feasible})
		}
		c.JSON(200, gin.H{"estado": result.Status, "versionAlgoritmo": "solar-v1", "alternativas": items, "totalAlternativas": total, "alternativasTruncadas": total > 20, "criterioSeleccion": result.Criteria, "fuenteRecurso": "Manual: " + input.Resource.Source, "hsp": json.Number(input.Resource.HSP), "supuestos": []string{result.Note, "Precios de catálogo referenciales, sin incluir instalación ni baterías.", "Orientación e inclinación no verificadas; superficie efectiva incluye reserva didáctica de disposición."}, "traceId": platform.TraceID(c.Request.Context())})
	})
}
