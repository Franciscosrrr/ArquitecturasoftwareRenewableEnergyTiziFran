package external

import (
	"context"
	"encoding/json"
	"io"
	"math/big"
	"mime"
	"net/http"
	"net/url"
	"renewable.local/platform"
	"renewable.local/solar/internal/domain"
	"strconv"
	"strings"
	"sync"
	"time"
)

/**
 * Recupera fichas propias; el vendedor comercial no es autoridad de potencia ni superficie.
 */
type PanelCatalog interface {
	/**
	 * Recupera el catálogo técnico conservando su procedencia.
	 *
	 * @method Panels
	 * @param {context.Context} ctx - Presupuesto de lectura.
	 * @param {bool} all - Incluye inactivos cuando es verdadero.
	 * @returns {[]domain.Panel, error} Fichas o indisponibilidad.
	 */
	Panels(context.Context, bool) ([]domain.Panel, error)
}

/**
 * Adapta Comparador de Proveedores 1.1.0 con circuito, cuota y decimales exactos.
 */
type Comparator struct {
	Client   *Client
	Catalog  PanelCatalog
	mu       sync.Mutex
	failures int
	until    time.Time
	probe    bool
}

/**
 * Relaciona identidades locales históricas con SKUs acordados, sin parsear nombres.
 *
 * @method ProviderSKU
 * @param {string} id - Identidad de ficha de nuestro catálogo.
 * @returns {string} SKU comercial; vacío cuando no existe una equivalencia acordada.
 */
func ProviderSKU(id string) string {
	switch id {
	case "PANEL-SIM-450":
		return "PAN-450W-MONO"
	case "PANEL-SIM-550":
		return "PAN-550W-MONO"
	case "PAN-100W-FLEX", "PAN-330W-POLI", "PAN-450W-MONO", "PAN-550W-MONO", "PAN-600W-BIF":
		return id
	}
	return ""
}

/**
 * Convierte un decimal de pesos a centavos sin utilizar coma flotante ni redondear entradas.
 *
 * @method moneyCents
 * @param {domain.DecimalNumber} value - Importe JSON con hasta dos decimales efectivos.
 * @returns {int64, error} Centavos exactos o contrato incompatible.
 */
func moneyCents(value domain.DecimalNumber) (int64, error) {
	if len(value) == 0 || len(value) > 32 {
		return 0, domain.ErrContract
	}
	r, ok := new(big.Rat).SetString(string(value))
	if !ok || r.Sign() < 0 {
		return 0, domain.ErrContract
	}
	r.Mul(r, big.NewRat(100, 1))
	if !r.IsInt() || !r.Num().IsInt64() || r.Num().Int64() > 1000000000000000 {
		return 0, domain.ErrContract
	}
	return r.Num().Int64(), nil
}

/**
 * Conserva Retry-After y abre el circuito tras tres fallos; admite una prueba de recuperación.
 *
 * @method finish
 * @param {error} err - Resultado saneado de la interacción.
 * @param {time.Duration} retry - Espera mínima anunciada por el proveedor.
 * @returns {void} Actualiza únicamente el estado técnico local.
 */
func (c *Comparator) finish(err error, retry time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	half := c.probe
	c.probe = false
	if err == nil || err == domain.ErrStock {
		c.failures = 0
		c.until = time.Time{}
		return
	}
	c.failures++
	if half || c.failures >= 3 || retry > 0 {
		if retry < 10*time.Second {
			retry = 10 * time.Second
		}
		c.until = time.Now().Add(retry)
	}
}

/**
 * Consulta el ranking para el SKU y cantidad exactos; una lista vacía significa falta de oferta.
 *
 * @method QuoteModel
 * @param {context.Context} ctx - Cancelación y presupuesto compartido del estudio.
 * @param {int64} watts - Potencia objetivo calculada en nuestro dominio.
 * @param {string} id - Ficha local, cuya potencia no se obtiene del nombre comercial.
 * @returns {domain.Quote, error} Oferta comercial, falta de stock o fallo saneado.
 */
func (c *Comparator) QuoteModel(ctx context.Context, watts int64, id string) (quote domain.Quote, err error) {
	sku := ProviderSKU(id)
	if sku == "" {
		return quote, domain.ErrStock
	}
	panels, e := c.Catalog.Panels(ctx, false)
	if e != nil {
		return quote, domain.ErrUnavailable
	}
	var panel domain.Panel
	for _, p := range panels {
		if p.ID == id {
			panel = p
			break
		}
	}
	if panel.Watts < 1 || panel.Watts > 2000 || watts < 1 {
		return quote, domain.ErrValidation
	}
	quantity := (watts + panel.Watts - 1) / panel.Watts
	if quantity < 1 || quantity > 1000 {
		return quote, domain.ErrValidation
	}
	c.mu.Lock()
	if !c.until.IsZero() {
		if time.Now().Before(c.until) || c.probe {
			c.mu.Unlock()
			return quote, domain.ErrUnavailable
		}
		c.probe = true
	}
	c.mu.Unlock()
	retry := time.Duration(0)
	defer func() { c.finish(err, retry) }()
	u := c.Client.BaseURL + "/api/v1/proveedores/recomendados?" + url.Values{"producto": {sku}, "cantidad": {strconv.FormatInt(quantity, 10)}, "limite": {"1"}}.Encode()
	req, e := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if e != nil {
		return quote, domain.ErrUnavailable
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-API-Key", c.Client.Token)
	if id := platform.RequestID(ctx); id != "" {
		req.Header.Set("X-Request-Id", id)
	}
	res, e := c.Client.HTTP.Do(req)
	if e != nil {
		return quote, domain.ErrUnavailable
	}
	defer res.Body.Close()
	raw, e := io.ReadAll(io.LimitReader(res.Body, 65537))
	if e != nil {
		return quote, domain.ErrUnavailable
	}
	if len(raw) > 65536 {
		return quote, domain.ErrContract
	}
	media, _, _ := mime.ParseMediaType(res.Header.Get("Content-Type"))
	if res.StatusCode != http.StatusOK {
		if res.StatusCode == 429 {
			seconds, e := strconv.ParseInt(res.Header.Get("Retry-After"), 10, 32)
			if e != nil || seconds < 1 {
				return quote, domain.ErrContract
			}
			retry = time.Duration(seconds) * time.Second
		}
		if res.StatusCode == 429 || res.StatusCode >= 500 {
			return quote, domain.ErrUnavailable
		}
		var problem struct {
			Code string `json:"code"`
		}
		if media == "application/problem+json" && json.Unmarshal(raw, &problem) == nil && res.StatusCode == 404 && problem.Code == "PRODUCTO_NO_ENCONTRADO" {
			return quote, domain.ErrStock
		}
		return quote, domain.ErrContract
	}
	if media != "application/json" || strings.TrimSpace(string(raw)) == "null" {
		return quote, domain.ErrContract
	}
	var offers []struct {
		ProviderID int64                `json:"proveedor_id"`
		Name       string               `json:"proveedor_nombre"`
		Unit       domain.DecimalNumber `json:"precio_unitario"`
		Discount   domain.DecimalNumber `json:"descuento_pct"`
		Applied    *bool                `json:"descuento_aplicado"`
		Freight    domain.DecimalNumber `json:"flete"`
		Stock      int64                `json:"stock_disponible"`
		Total      domain.DecimalNumber `json:"costo_total"`
	}
	if json.Unmarshal(raw, &offers) != nil || len(offers) > 1 {
		return quote, domain.ErrContract
	}
	if len(offers) == 0 {
		return quote, domain.ErrStock
	}
	offer := offers[0]
	if offer.Applied == nil {
		return quote, domain.ErrContract
	}
	unit, e := moneyCents(offer.Unit)
	if e != nil {
		return quote, e
	}
	total, e := moneyCents(offer.Total)
	if e != nil {
		return quote, e
	}
	freight, e := moneyCents(offer.Freight)
	if e != nil {
		return quote, e
	}
	quote = domain.Quote{SKU: id, PanelWatts: panel.Watts, Quantity: quantity, Stock: offer.Stock, UnitCents: unit, TotalCents: total, Currency: "ARS", Source: "externo", ProviderSKU: sku, ProviderID: offer.ProviderID, ProviderName: offer.Name, RequestID: res.Header.Get("X-Request-Id"), FreightCents: freight, Discount: offer.Discount, DiscountApplied: *offer.Applied, ConsultedAt: time.Now().UTC().Format(time.RFC3339)}
	err = quote.Validate(watts)
	return quote, err
}
