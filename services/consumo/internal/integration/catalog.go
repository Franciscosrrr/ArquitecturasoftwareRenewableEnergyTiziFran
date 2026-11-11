package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"renewable.local/consumo/internal/application"
	"renewable.local/platform"
	"sync"
	"time"
)

var ErrUnavailable = errors.New("catálogo no disponible")

/**
 * Resuelve lotes del catálogo por HTTP con credencial interna, timeout y circuito.
 *
 * @method Catalog
 * @returns {Catalog} Representación tipada usada por este componente.
 */
type Catalog struct {
	URL, Token string
	Client     *http.Client
	mu         sync.Mutex
	failures   int
	openUntil  time.Time
	probe      bool
}

/**
 * Construye el resolver HTTP autenticado con timeout y circuito de catálogo.
 *
 * @method NewCatalog
 * @param {string} url - URL interna del servicio de catálogo.
 * @param {string} token - Credencial técnica del servicio.
 * @returns {*Catalog} Cliente de catálogo con timeout, circuito y credencial interna.
 */
func NewCatalog(url, token string) *Catalog {
	return &Catalog{URL: url, Token: token, Client: platform.Client(2 * time.Second)}
}

// Circuit breaker: three failed calls open for ten seconds; one half-open probe.
/**
 * Obtiene fichas canónicas activas para calcular con sus versiones exactas.
 *
 * @method Resolve
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {[]string} ids - Identificadores únicos de las fichas a recuperar en lote.
 * @returns {[]application.Ficha, error} Fichas activas solicitadas o error de validación, ausencia o dependencia.
 */
func (c *Catalog) Resolve(ctx context.Context, ids []string) ([]application.Ficha, error) {
	c.mu.Lock()
	halfOpen := !c.openUntil.IsZero()
	if halfOpen && (time.Now().Before(c.openUntil) || c.probe) {
		c.mu.Unlock()
		return nil, ErrUnavailable
	}
	if halfOpen {
		c.probe = true
	}
	c.mu.Unlock()
	raw, _ := json.Marshal(map[string]any{"ids": ids})
	req, err := http.NewRequestWithContext(ctx, "POST", c.URL+"/internal/v1/electrodomesticos/resolver", bytes.NewReader(raw))
	if err != nil {
		return nil, ErrUnavailable
	}
	req.Header.Set("Content-Type", "application/json")
	if id := platform.RequestID(ctx); id != "" {
		req.Header.Set("X-Request-Id", id)
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	res, err := c.Client.Do(req)
	var fichas []application.Ficha
	if err == nil {
		defer res.Body.Close()
		if res.StatusCode != 200 {
			err = ErrUnavailable
		} else {
			dec := json.NewDecoder(io.LimitReader(res.Body, 262144))
			dec.UseNumber()
			err = dec.Decode(&fichas)
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.probe = false
	if err != nil {
		c.failures++
		if halfOpen || c.failures >= 3 {
			c.openUntil = time.Now().Add(10 * time.Second)
		}
		return nil, ErrUnavailable
	}
	c.failures = 0
	c.openUntil = time.Time{}
	return fichas, nil
}
