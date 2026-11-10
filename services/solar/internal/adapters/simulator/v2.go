package simulator

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"renewable.local/platform"
	"renewable.local/solar/internal/domain"
)

/**
 * Expone la simulación comercial por SKU sin reservar ni comprar stock.
 *
 * @method registerV2
 * @param {*gin.Engine} r - Router o receptor de operaciones HTTP.
 * @param {string} key - Clave de idempotencia o identidad técnica.
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func registerV2(r *gin.Engine, key string) {
	r.POST("/api/v2/cotizaciones", func(c *gin.Context) {
		if !platform.CredentialOK(c.GetHeader("Authorization"), "Bearer "+key) {
			platform.Error(c, 401, "NO_AUTENTICADO", "Credencial comercial inválida.", nil)
			return
		}
		var input struct {
			Watts int64  `json:"potenciaObjetivoW"`
			SKU   string `json:"sku"`
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
		d := json.NewDecoder(c.Request.Body)
		d.DisallowUnknownFields()
		if d.Decode(&input) != nil || d.Decode(new(any)) != io.EOF || input.Watts < 1 || input.Watts > 10000000 {
			platform.Error(c, 422, "VALIDACION", "Solicitud comercial inválida.", nil)
			return
		}
		power, price, stock := int64(450), int64(25000000), int64(1000)
		switch input.SKU {
		case "PANEL-SIM-450":
		case "PANEL-SIM-550":
			power = 550
			price = 30000000
			stock = 100
		default:
			platform.Error(c, 404, "SIN_OFERTA", "Modelo sin oferta.", nil)
			return
		}
		quantity := (input.Watts + power - 1) / power
		if quantity > stock {
			platform.Error(c, 409, "STOCK_INSUFICIENTE", "Stock insuficiente.", nil)
			return
		}
		c.JSON(200, domain.Quote{ID: "sim-modelo-" + input.SKU, SKU: input.SKU, PanelWatts: power, Quantity: quantity, Stock: stock, UnitCents: price, TotalCents: price * quantity, Currency: "ARS"})
	})
}
