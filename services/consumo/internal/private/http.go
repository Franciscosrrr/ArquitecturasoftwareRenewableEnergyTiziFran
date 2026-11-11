package private

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"math"
	"regexp"
	"renewable.local/consumo/internal/application"
	"renewable.local/platform"
	"strings"
)

/**
 * Congela entrada validada, fichas utilizadas y resultado para reproducir una evaluación.
 *
 * @method Snapshot
 * @returns {Snapshot} Representación tipada usada por este componente.
 */
type Snapshot struct {
	Consumption application.Result  `json:"consumo"`
	Fichas      []application.Ficha `json:"fichas"`
	Input       json.RawMessage     `json:"entrada"`
	Place       Place               `json:"lugar"`
}

/**
 * Conecta persistencia propia y resolución de catálogo para confirmar evaluaciones HTTP.
 *
 * @method Service
 * @returns {Service} Representación tipada usada por este componente.
 */
type Service struct {
	Store   *Store
	Catalog application.Resolver
}

/**
 * Registra las fichas efectivamente resueltas mientras calcula el snapshot.
 *
 * @method captureResolver
 * @returns {captureResolver} Representación tipada usada por este componente.
 */
type captureResolver struct {
	Source application.Resolver
	Items  []application.Ficha
}

/**
 * Obtiene fichas canónicas activas para calcular con sus versiones exactas.
 *
 * @method Resolve
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {[]string} ids - Identificadores únicos de las fichas a recuperar en lote.
 * @returns {[]application.Ficha, error} Fichas activas solicitadas o error de validación, ausencia o dependencia.
 */
func (r *captureResolver) Resolve(ctx context.Context, ids []string) ([]application.Ficha, error) {
	items, err := r.Source.Resolve(ctx, ids)
	r.Items = items
	return items, err
}

/**
 * Valida rangos y precisión decimal de los supuestos solares antes de persistir.
 *
 * @method validOptions
 * @param {SolarOptions} o - HSP, rendimiento, cobertura y superficie del dimensionamiento.
 * @returns {bool} true si los supuestos solares cumplen límites y precisión; false de otro modo.
 */
func validOptions(o SolarOptions) bool {
	for _, v := range []float64{o.Sun, o.Yield, o.Coverage} {
		if math.IsInf(v, 0) || math.Abs(v*1e6-math.Round(v*1e6)) > .00001 {
			return false
		}
	}
	if o.Area != nil && math.Abs(*o.Area*1e6-math.Round(*o.Area*1e6)) > .00001 {
		return false
	}
	return o.Sun >= .5 && o.Sun <= 8 && o.Yield >= .1 && o.Yield <= 1 && o.Coverage > 0 && o.Coverage <= 1 && (o.Area == nil || (*o.Area > 0 && *o.Area <= 1000000)) && !math.IsNaN(o.Sun) && !math.IsNaN(o.Yield) && !math.IsNaN(o.Coverage)
}

/**
 * Construye un evento con propietario, snapshot y correlación derivados del servidor.
 *
 * @method newEvent
 * @param {*gin.Context} c - Solicitud HTTP en curso.
 * @param {string} owner - Identificador del propietario autenticado.
 * @param {SolarOptions} options - Supuestos solares explícitos.
 * @param {json.RawMessage} snapshot - Datos congelados para reproducir el resultado.
 * @returns {Event} Evento con snapshot y correlación definidos por el servidor.
 */
func newEvent(c *gin.Context, owner string, options SolarOptions, snapshot json.RawMessage) Event {
	carrier := propagation.MapCarrier{}
	otel.GetTextMapPropagator().Inject(c.Request.Context(), carrier)
	return Event{ID: platform.RandomID(), StudyID: platform.RandomID(), Type: "ConsumoEvaluado.v1", Owner: owner, Solar: options, Snapshot: snapshot, Traceparent: carrier["traceparent"], RequestID: platform.RequestID(c.Request.Context())}
}

/**
 * Traduce errores de dominio y dependencias a respuestas públicas sin datos internos.
 *
 * @method fail
 * @param {*gin.Context} c - Solicitud HTTP en curso.
 * @param {error} err - Error de dominio o infraestructura que debe traducirse a una respuesta pública.
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func fail(c *gin.Context, err error) {
	if errors.Is(err, ErrPlaceInUse) {
		platform.Error(c, 409, "LUGAR_CON_CONSUMOS", "Este lugar tiene consumos guardados. Movelos a otro lugar o eliminalos antes.", nil)
		return
	}
	status, code := 503, "SERVICIO_NO_DISPONIBLE"
	var validation *application.ValidationError
	if errors.As(err, &validation) {
		status, code = 422, "VALIDACION"
	}
	if errors.Is(err, ErrNotFound) {
		status, code = 404, "NO_ENCONTRADO"
	}
	var sqlError *mysql.MySQLError
	if errors.Is(err, ErrConflict) || errors.As(err, &sqlError) && (sqlError.Number == 1062 || sqlError.Number == 1213) {
		status, code = 409, "CONFLICTO"
	}
	platform.Error(c, status, code, "No se pudo completar la operación; revise versión, permisos y dependencias.", nil)
}

/**
 * Registra operaciones HTTP y aplica autorización en el adaptador del servicio.
 *
 * @method Register
 * @param {*gin.Engine} r - Router o receptor de operaciones HTTP.
 * @param {string} token - Credencial técnica del servicio.
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func (s Service) Register(r *gin.Engine, token string) {
	api := r.Group("/api/v1", platform.Internal(token, false))
	s.registerLifecycle(api)
	api.GET("/lugares", func(c *gin.Context) {
		items, err := s.Store.Places(c.Request.Context(), platform.ActorID(c))
		if err != nil {
			fail(c, err)
			return
		}
		c.JSON(200, items)
	})
	savePlace := func(c *gin.Context) {
		var p Place
		if !platform.Decode(c, &p) {
			return
		}
		if strings.TrimSpace(p.Name) == "" || len(p.Name) > 100 || len(p.Location) > 200 || (p.Area != nil && (*p.Area <= 0 || *p.Area > 1000000)) {
			platform.Error(c, 422, "VALIDACION", "Nombre, ubicación o superficie inválidos.", nil)
			return
		}
		if c.Request.Method == "POST" {
			p.ID = ""
		} else {
			p.ID = c.Param("id")
			if p.Version < 1 {
				platform.Error(c, 422, "VALIDACION", "Se requiere versión.", nil)
				return
			}
		}
		saved, err := s.Store.SavePlace(c.Request.Context(), platform.ActorID(c), p)
		if err != nil {
			fail(c, err)
			return
		}
		status := 200
		if c.Request.Method == "POST" {
			status = 201
		}
		c.JSON(status, saved)
	}
	api.POST("/lugares", savePlace)
	api.PUT("/lugares/:id", savePlace)
	api.GET("/configuraciones", func(c *gin.Context) {
		items, err := s.Store.Configs(c.Request.Context(), platform.ActorID(c))
		if err != nil {
			fail(c, err)
			return
		}
		c.JSON(200, items)
	})
	api.GET("/configuraciones/:id", func(c *gin.Context) {
		saved, err := s.Store.Config(c.Request.Context(), platform.ActorID(c), c.Param("id"))
		if err != nil {
			fail(c, err)
			return
		}
		c.JSON(200, saved)
	})
	saveConfig := func(c *gin.Context) {
		var input struct {
			Name        string          `json:"nombre"`
			PlaceID     string          `json:"lugarId"`
			Version     int64           `json:"version"`
			Consumption json.RawMessage `json:"consumo"`
			Archived    bool            `json:"archivada"`
		}
		if !platform.Decode(c, &input) {
			return
		}
		if strings.TrimSpace(input.Name) == "" || len(input.Name) > 100 {
			platform.Error(c, 422, "VALIDACION", "Nombre inválido.", nil)
			return
		}
		owner := platform.ActorID(c)
		place, err := s.Store.Place(c.Request.Context(), owner, input.PlaceID)
		if err != nil || place.Archived {
			fail(c, ErrNotFound)
			return
		}
		capture := &captureResolver{Source: s.Catalog}
		result, err := application.CalculateCatalog(c.Request.Context(), input.Consumption, platform.TraceID(c.Request.Context()), capture)
		if err != nil {
			fail(c, err)
			return
		}
		fichas := capture.Items
		snapshot, _ := json.Marshal(Snapshot{result, fichas, input.Consumption, place})
		cfg := Config{PlaceID: input.PlaceID, Name: input.Name, Input: input.Consumption, Snapshot: snapshot, Archived: input.Archived, Version: input.Version}
		if c.Request.Method == "PUT" {
			cfg.ID = c.Param("id")
		}
		saved, err := s.Store.SaveConfig(c.Request.Context(), owner, cfg)
		if err != nil {
			fail(c, err)
			return
		}
		status := 200
		if cfg.ID == "" {
			status = 201
		}
		c.JSON(status, saved)
	}
	api.POST("/configuraciones", saveConfig)
	api.PUT("/configuraciones/:id", saveConfig)
	api.POST("/configuraciones/:id/duplicar", func(c *gin.Context) {
		saved, err := s.Store.Config(c.Request.Context(), platform.ActorID(c), c.Param("id"))
		if err != nil {
			fail(c, err)
			return
		}
		saved.ID = ""
		saved.Name = "Copia de " + saved.Name
		if len(saved.Name) > 100 {
			saved.Name = saved.Name[:100]
		}
		saved.Archived = false
		copy, err := s.Store.SaveConfig(c.Request.Context(), platform.ActorID(c), saved)
		if err != nil {
			fail(c, err)
			return
		}
		c.JSON(201, copy)
	})
	keyPattern := regexp.MustCompile(`^[A-Za-z0-9_-]{8,128}$`)
	api.POST("/configuraciones/:id/evaluaciones", func(c *gin.Context) {
		var input struct {
			Version      int64        `json:"version"`
			PlaceVersion int64        `json:"versionLugar"`
			Solar        SolarOptions `json:"solar"`
		}
		if !platform.Decode(c, &input) {
			return
		}
		if !validOptions(input.Solar) || input.Version < 1 || input.PlaceVersion < 1 || !keyPattern.MatchString(c.GetHeader("Idempotency-Key")) {
			platform.Error(c, 422, "VALIDACION", "Se requieren versiones, parámetros solares e Idempotency-Key de 8 a 128 caracteres.", nil)
			return
		}
		owner, key := platform.ActorID(c), c.GetHeader("Idempotency-Key")
		canonical, _ := json.Marshal(input)
		hash := platform.Digest(c.Param("id") + string(canonical))
		id, err := s.Store.Existing(c.Request.Context(), owner, key, hash)
		if err != nil {
			fail(c, err)
			return
		}
		if id != "" {
			e, err := s.Store.Evaluation(c.Request.Context(), owner, id)
			if err != nil {
				fail(c, err)
				return
			}
			c.JSON(200, e)
			return
		}
		saved, err := s.Store.Config(c.Request.Context(), owner, c.Param("id"))
		if err != nil {
			fail(c, err)
			return
		}
		if saved.Version != input.Version || saved.Archived {
			fail(c, ErrConflict)
			return
		}
		var snapshot Snapshot
		if json.Unmarshal(saved.Snapshot, &snapshot) != nil {
			fail(c, ErrConflict)
			return
		}
		ids := []string{}
		versions := map[string]int64{}
		for _, f := range snapshot.Fichas {
			ids = append(ids, f.ID)
			versions[f.ID] = f.Version
		}
		fresh, err := s.Catalog.Resolve(c.Request.Context(), ids)
		if err != nil {
			fail(c, err)
			return
		}
		if len(fresh) != len(ids) {
			fail(c, ErrConflict)
			return
		}
		for _, f := range fresh {
			if versions[f.ID] != f.Version {
				fail(c, ErrConflict)
				return
			}
		}
		place, err := s.Store.Place(c.Request.Context(), owner, saved.PlaceID)
		if err != nil {
			fail(c, err)
			return
		}
		if input.Solar.Area == nil {
			input.Solar.Area = place.Area
		}
		event := newEvent(c, owner, input.Solar, saved.Snapshot)
		id, err = s.Store.Confirm(c.Request.Context(), owner, key, hash, saved, input.PlaceVersion, event)
		if err != nil {
			fail(c, err)
			return
		}
		e, err := s.Store.Evaluation(c.Request.Context(), owner, id)
		if err != nil {
			fail(c, err)
			return
		}
		c.JSON(201, e)
	})
	api.GET("/evaluaciones", func(c *gin.Context) {
		items, err := s.Store.History(c.Request.Context(), platform.ActorID(c), c.Query("configuracionId"))
		if err != nil {
			fail(c, err)
			return
		}
		c.JSON(200, items)
	})
	api.GET("/evaluaciones/:id", func(c *gin.Context) {
		e, err := s.Store.Evaluation(c.Request.Context(), platform.ActorID(c), c.Param("id"))
		if err != nil {
			fail(c, err)
			return
		}
		c.JSON(200, e)
	})
	api.POST("/evaluaciones/:id/estudios", func(c *gin.Context) {
		var options SolarOptions
		if !platform.Decode(c, &options) {
			return
		}
		if !validOptions(options) || !keyPattern.MatchString(c.GetHeader("Idempotency-Key")) {
			platform.Error(c, 422, "VALIDACION", "Parámetros o Idempotency-Key inválidos.", nil)
			return
		}
		event := newEvent(c, platform.ActorID(c), options, nil)
		id, err := s.Store.AddStudy(c.Request.Context(), platform.ActorID(c), c.Param("id"), c.GetHeader("Idempotency-Key"), event)
		if err != nil {
			fail(c, err)
			return
		}
		c.JSON(202, gin.H{"estudioId": id, "estado": "PENDIENTE"})
	})
}
