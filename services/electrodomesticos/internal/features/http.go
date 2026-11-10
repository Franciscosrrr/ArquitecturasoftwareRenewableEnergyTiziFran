package features

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"math"
	"regexp"
	"renewable.local/electrodomesticos/internal/repository"
	"renewable.local/platform"
	"strconv"
	"strings"
	"time"
)

/**
 * Registra operaciones HTTP y aplica autorización en el adaptador del servicio.
 *
 * @method Register
 * @param {*gin.Engine} router - Valor de router requerido por la operación.
 * @param {string} token - Credencial técnica del servicio.
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func (r *Runtime) Register(router *gin.Engine, token string) {
	admin := router.Group("/api/v1/admin/electrodomesticos", platform.Internal(token, true))
	admin.GET("", func(c *gin.Context) {
		docs, err := r.All(c.Request.Context())
		if err != nil {
			platform.Error(c, 503, "SERVICIO_NO_DISPONIBLE", "Catálogo no disponible.", nil)
			return
		}
		c.JSON(200, docs)
	})
	save := func(c *gin.Context) {
		var input struct {
			ID       string  `json:"id"`
			Name     string  `json:"nombre"`
			Category string  `json:"categoria"`
			Mode     string  `json:"modo"`
			Power    float64 `json:"potenciaW"`
			Cycle    float64 `json:"energiaPorCicloKWh"`
			Factor   float64 `json:"factorFuncionamiento"`
			Source   string  `json:"fuente"`
			Version  int64   `json:"version"`
			Active   bool    `json:"activo"`
		}
		if !platform.Decode(c, &input) {
			return
		}
		if c.Request.Method == "PUT" {
			input.ID = c.Param("id")
		}
		validNumber := func(x float64) bool {
			return !math.IsNaN(x) && !math.IsInf(x, 0) && math.Abs(x*1e6-math.Round(x*1e6)) < .00001
		}
		if !regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`).MatchString(input.ID) || strings.TrimSpace(input.Name) == "" || len(input.Name) > 120 || input.Category == "" || len(input.Category) > 64 || len(input.Source) > 500 || !validNumber(input.Power) || !validNumber(input.Cycle) || !validNumber(input.Factor) || (input.Mode != "POTENCIA" && input.Mode != "CICLOS") || (input.Mode == "POTENCIA" && (input.Power <= 0 || input.Power > 1000000 || input.Cycle != 0 || input.Factor <= 0 || input.Factor > 1)) || (input.Mode == "CICLOS" && (input.Cycle <= 0 || input.Cycle > 1000 || input.Power != 0)) || (c.Request.Method == "PUT" && input.Version < 1) {
			platform.Error(c, 422, "VALIDACION", "Ficha inválida.", nil)
			return
		}
		if input.Mode == "CICLOS" {
			input.Factor = 1
		}
		item := repository.Item{ID: input.ID, Name: input.Name, Category: input.Category, Mode: input.Mode, Power: json.Number(strconv.FormatFloat(input.Power, 'f', -1, 64)), Cycle: json.Number(strconv.FormatFloat(input.Cycle, 'f', -1, 64)), Factor: json.Number(strconv.FormatFloat(input.Factor, 'f', -1, 64)), Source: input.Source, Version: input.Version}
		saved, err := r.Save(c.Request.Context(), item, input.Active, c.Request.Method == "POST")
		if err != nil {
			platform.Error(c, 409, "CONFLICTO", "Identificador existente o versión obsoleta.", nil)
			return
		}
		status := 200
		if c.Request.Method == "POST" {
			status = 201
		}
		c.JSON(status, saved)
	}
	admin.POST("", save)
	admin.PUT("/:id", save)
	router.POST("/api/v1/admin/catalogo/reindexar", platform.Internal(token, true), func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()
		if err := r.Rebuild(ctx); err != nil {
			platform.Error(c, 503, "SERVICIO_NO_DISPONIBLE", "Reconstrucción no disponible.", nil)
			return
		}
		c.JSON(200, gin.H{"estado": "INDICE_RECONSTRUIDO"})
	})
}
