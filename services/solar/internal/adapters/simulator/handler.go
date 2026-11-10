package simulator

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"mime"
	"net/http"
	"renewable.local/platform"
	"renewable.local/solar/internal/domain"
)

/**
 * Publica un vendedor ficticio explícito para ensayar cotizaciones sin compras.
 *
 * @method NewHandler
 * @param {string} healthToken - Secreto técnico de salud.
 * @param {string} apiKey - Credencial comercial del cliente Solar.
 * @returns {http.Handler} API simulada con stock y precios didácticos.
 */
func NewHandler(healthToken, apiKey string) http.Handler {
	r := platform.Router()
	platform.Health(r, healthToken, func(context.Context) error { return nil })
	r.POST("/api/v1/cotizaciones", func(c *gin.Context) {
		supplied := sha256.Sum256([]byte(c.GetHeader("Authorization")))
		expected := sha256.Sum256([]byte("Bearer " + apiKey))
		if len(apiKey) < 32 || subtle.ConstantTimeCompare(supplied[:], expected[:]) != 1 {
			platform.Error(c, 401, "NO_AUTENTICADO", "Credencial comercial inválida.", nil)
			return
		}
		media, _, _ := mime.ParseMediaType(c.GetHeader("Content-Type"))
		if media != "application/json" {
			platform.Error(c, 415, "TIPO_CONTENIDO_NO_SOPORTADO", "Utilice JSON.", nil)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
		decoder := json.NewDecoder(c.Request.Body)
		decoder.DisallowUnknownFields()
		var input struct {
			Watts int64 `json:"potenciaObjetivoW"`
		}
		if decoder.Decode(&input) != nil || decoder.Decode(new(any)) != io.EOF || input.Watts < 1 || input.Watts > 10000000 {
			platform.Error(c, 422, "VALIDACION", "Potencia fuera de rango.", nil)
			return
		}
		quantity := (input.Watts + 449) / 450
		if quantity > 1000 {
			platform.Error(c, 409, "STOCK_INSUFICIENTE", "Stock insuficiente.", nil)
			return
		}
		c.JSON(200, domain.Quote{ID: fmt.Sprintf("sim-%d", input.Watts), SKU: "PANEL-SIM-450", PanelWatts: 450, Quantity: quantity, Stock: 1000, UnitCents: 25000000, TotalCents: quantity * 25000000, Currency: "ARS"})
	})
	registerV2(r, apiKey)
	return platform.Instrument("proveedor-paneles-sim", r)
}
