package external

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/url"
	"renewable.local/platform"
	"renewable.local/solar/internal/domain"
	"strings"
	"sync"
	"time"
)

/**
 * Encapsula HTTP saliente con límites de tiempo y sin redirecciones de secretos.
 */
type Client struct {
	BaseURL   string
	Token     string
	HTTP      *http.Client
	mu        sync.Mutex
	failures  int
	openUntil time.Time
	probe     bool
}

/**
 * Valida un destino de configuración y configura un cliente acotado.
 *
 * @method NewClient
 * @param {string} base - URL HTTP o HTTPS configurada por el operador.
 * @param {string} token - Credencial del vendedor, nunca enviada al navegador.
 * @param {time.Duration} timeout - Tiempo máximo de una llamada.
 * @returns {*Client, error} Cliente o error de configuración.
 */
func NewClient(base, token string, timeout time.Duration) (*Client, error) {
	u, err := url.Parse(base)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return nil, domain.ErrValidation
	}
	h := platform.Client(timeout)
	h.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	return &Client{BaseURL: strings.TrimRight(base, "/"), Token: token, HTTP: h}, nil
}

/**
 * Envía JSON, limita la respuesta y conserva la correlación de la operación.
 *
 * @method post
 * @param {context.Context} ctx - Contexto cancelable de negocio.
 * @param {string} path - Ruta fija del contrato.
 * @param {[]byte} raw - Cuerpo JSON de la solicitud.
 * @returns {[]byte, int, error} Cuerpo, estado HTTP y error técnico.
 */
func (c *Client) post(ctx context.Context, path string, raw []byte) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(raw))
	if err != nil {
		return nil, 0, domain.ErrUnavailable
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	if id := platform.RequestID(ctx); id != "" {
		req.Header.Set("X-Request-Id", id)
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, domain.ErrUnavailable
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 65537))
	if err != nil {
		return nil, res.StatusCode, domain.ErrUnavailable
	}
	if res.StatusCode != 200 {
		return body, res.StatusCode, nil
	}
	media, _, _ := mime.ParseMediaType(res.Header.Get("Content-Type"))
	if len(body) > 65536 || media != "application/json" || !json.Valid(body) {
		return nil, res.StatusCode, domain.ErrContract
	}
	return body, res.StatusCode, nil
}

/**
 * Llama directamente a Consumo y adapta solo los datos necesarios para Solar.
 *
 * @method Estimate
 * @param {context.Context} ctx - Presupuesto global del caso de uso.
 * @param {json.RawMessage} raw - Solicitud por catálogo a validar por Consumo.
 * @returns {domain.Consumption, error} Energía o error de dependencia/entrada.
 */
func (c *Client) Estimate(ctx context.Context, raw json.RawMessage) (domain.Consumption, error) {
	body, status, err := c.post(ctx, "/api/v1/publico/consumo-catalogo", raw)
	if err != nil {
		return domain.Consumption{}, domain.ErrUnavailable
	}
	if status == 422 {
		return domain.Consumption{}, domain.ErrValidation
	}
	if status != 200 {
		return domain.Consumption{}, domain.ErrUnavailable
	}
	var energy domain.Consumption
	if json.Unmarshal(body, &energy) != nil || energy.Days < 1 || energy.Days > 366 || energy.Total == "" || energy.Daily == "" {
		return domain.Consumption{}, domain.ErrUnavailable
	}
	return energy, nil
}

/**
 * Consulta al vendedor con circuito de tres fallos y una prueba de recuperación.
 *
 * @method Quote
 * @param {context.Context} ctx - Presupuesto global cancelable.
 * @param {int64} watts - Potencia objetivo en W.
 * @returns {domain.Quote, error} Oferta, falta de stock o fallo controlado.
 */
func (c *Client) Quote(ctx context.Context, watts int64) (domain.Quote, error) {
	return c.quote(ctx, watts, "")
}

/**
 * Solicita una cotización para el SKU exacto seleccionado por el dominio.
 *
 * @method QuoteModel
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {int64} watts - Potencia objetivo de la instalación, expresada en W.
 * @param {string} sku - Identificador del modelo de panel que debe cotizarse sin sustituciones.
 * @returns {domain.Quote, error} Oferta del modelo exacto o error de stock, dependencia o contrato.
 */
func (c *Client) QuoteModel(ctx context.Context, watts int64, sku string) (domain.Quote, error) {
	return c.quote(ctx, watts, sku)
}

/**
 * Aplica el circuito compartido y valida la oferta comercial de la versión solicitada.
 *
 * @method quote
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {int64} watts - Potencia objetivo de la instalación, expresada en W.
 * @param {string} sku - Identificador del modelo de panel que debe cotizarse sin sustituciones.
 * @returns {domain.Quote, error} Oferta comercial validada o error controlado por stock, contrato o circuito.
 */
func (c *Client) quote(ctx context.Context, watts int64, sku string) (domain.Quote, error) {
	c.mu.Lock()
	halfOpen := !c.openUntil.IsZero()
	if halfOpen && (time.Now().Before(c.openUntil) || c.probe) {
		c.mu.Unlock()
		return domain.Quote{}, domain.ErrUnavailable
	}
	if halfOpen {
		c.probe = true
	}
	c.mu.Unlock()
	payload := map[string]any{"potenciaObjetivoW": watts}
	path := "/api/v1/cotizaciones"
	if sku != "" {
		payload["sku"] = sku
		path = "/api/v2/cotizaciones"
	}
	raw, _ := json.Marshal(payload)
	body, status, err := c.post(ctx, path, raw)
	var quote domain.Quote
	if err == nil {
		switch {
		case status == 409 || (sku != "" && status == 404):
			err = domain.ErrStock
		case status == 429 || status >= 500 || status == 0:
			err = domain.ErrUnavailable
		case status != 200:
			err = domain.ErrContract
		default:
			decoder := json.NewDecoder(bytes.NewReader(body))
			decoder.DisallowUnknownFields()
			if decoder.Decode(&quote) != nil {
				err = domain.ErrContract
			} else {
				err = quote.Validate(watts)
				if err == nil && sku != "" && quote.SKU != sku {
					err = domain.ErrContract
				}
			}
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.probe = false
	if err != nil && err != domain.ErrStock {
		c.failures++
		if halfOpen || c.failures >= 3 {
			c.openUntil = time.Now().Add(10 * time.Second)
		}
	} else {
		c.failures = 0
		c.openUntil = time.Time{}
	}
	return quote, err
}
