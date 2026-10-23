package transport

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http"
	"regexp"
	"renewable.local/electrodomesticos/internal/repository"
	"renewable.local/platform"
	"strconv"
	"unicode/utf8"
)

type Config struct {
	Store         *repository.Store
	InternalToken string
}

func NewHandler(token string, configs ...Config) http.Handler {
	r := platform.Router()
	var cfg Config
	var ready func(context.Context) error
	if len(configs) > 0 {
		cfg = configs[0]
		ready = cfg.Store.DB.PingContext
	}
	platform.Health(r, token, ready)
	fail := func(c *gin.Context) {
		platform.Error(c, 503, "SERVICIO_NO_DISPONIBLE", "Catálogo no disponible.", nil)
	}
	r.GET("/api/v1/publico/electrodomesticos", func(c *gin.Context) {
		if cfg.Store == nil {
			fail(c)
			return
		}
		q := repository.Query{Text: c.Query("q"), Category: c.Query("categoria"), Mode: c.Query("modo"), Sort: c.DefaultQuery("orden", "nombre"), Page: 1, Size: 20}
		var err error
		if v, ok := c.GetQuery("pagina"); ok {
			q.Page, err = strconv.Atoi(v)
			if err != nil {
				q.Page = 0
			}
		}
		if v, ok := c.GetQuery("tamano"); ok {
			q.Size, err = strconv.Atoi(v)
			if err != nil {
				q.Size = 0
			}
		}
		if q.Page < 1 || q.Page > 10000 || q.Size < 1 || q.Size > 100 || utf8.RuneCountInString(q.Text) > 100 || utf8.RuneCountInString(q.Category) > 64 || (q.Mode != "" && q.Mode != "POTENCIA" && q.Mode != "CICLOS") || (q.Sort != "nombre" && q.Sort != "potencia" && q.Sort != "-potencia") {
			platform.Error(c, 422, "VALIDACION", "Filtros o paginación inválidos.", nil)
			return
		}
		page, err := cfg.Store.List(c.Request.Context(), q)
		if err != nil {
			fail(c)
			return
		}
		c.JSON(200, page)
	})
	idPattern := regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)
	r.GET("/api/v1/publico/electrodomesticos/:id", func(c *gin.Context) {
		if cfg.Store == nil {
			fail(c)
			return
		}
		id := c.Param("id")
		if !idPattern.MatchString(id) {
			c.Status(404)
			return
		}
		items, err := cfg.Store.Resolve(c.Request.Context(), []string{id})
		if err != nil {
			fail(c)
			return
		}
		if len(items) == 0 {
			c.JSON(404, gin.H{"mensaje": "Ficha no disponible"})
			return
		}
		c.JSON(200, items[0])
	})
	r.POST("/internal/v1/electrodomesticos/resolver", func(c *gin.Context) {
		supplied := sha256.Sum256([]byte(c.GetHeader("Authorization")))
		expected := sha256.Sum256([]byte("Bearer " + cfg.InternalToken))
		if len(cfg.InternalToken) < 32 || subtle.ConstantTimeCompare(supplied[:], expected[:]) != 1 {
			platform.Error(c, 401, "NO_AUTENTICADO", "No autenticado", nil)
			return
		}
		if cfg.Store == nil {
			fail(c)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16384)
		var body struct {
			IDs []string `json:"ids"`
		}
		d := json.NewDecoder(c.Request.Body)
		d.DisallowUnknownFields()
		if d.Decode(&body) != nil || len(body.IDs) < 1 || len(body.IDs) > 100 {
			platform.Error(c, 422, "VALIDACION", "Identificadores inválidos.", nil)
			return
		}
		for _, id := range body.IDs {
			if !idPattern.MatchString(id) {
				platform.Error(c, 422, "VALIDACION", "Identificadores inválidos.", nil)
				return
			}
		}
		items, err := cfg.Store.Resolve(c.Request.Context(), body.IDs)
		if err != nil {
			fail(c)
			return
		}
		c.JSON(200, items)
	})
	r.NoRoute(func(c *gin.Context) {
		c.JSON(501, gin.H{"mensaje": "Funcionalidad fuera del alcance operativo de entrega 2"})
	})
	return platform.Instrument("electrodomesticos", r)
}
