package search

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"io"
	"net/http"
	"renewable.local/platform"
	"renewable.local/solar/internal/adapters/persistence"
	"renewable.local/solar/internal/domain"
	"strconv"
	"strings"
	"time"
)

/**
 * Sincroniza y consulta el índice solar derivado, con respaldo en MongoDB.
 *
 * @method Runtime
 * @returns {Runtime} Representación tipada usada por este componente.
 */
type Runtime struct {
	Store    *persistence.Store
	HTTP     *http.Client
	URL, Key string
}

/**
 * Conecta Solar al motor existente sin introducir contenedores ni acoplar repositorios SQL.
 *
 * @method New
 * @param {*persistence.Store} store - Repositorio canónico de paneles y pendientes.
 * @returns {*Runtime} Adaptador con presupuesto HTTP de dos segundos.
 */
func New(store *persistence.Store) *Runtime {
	return &Runtime{Store: store, HTTP: platform.Client(2 * time.Second), URL: platform.Env("SEARCH_URL", "http://meilisearch:7700"), Key: platform.Secret("SEARCH_API_KEY")}
}

/**
 * Ejecuta una operación autenticada y acotada contra Meilisearch.
 *
 * @method request
 * @param {context.Context} ctx - Presupuesto de la operación.
 * @param {string} method - Método HTTP.
 * @param {string} path - Ruta interna del motor.
 * @param {any} value - JSON opcional de entrada.
 * @returns {[]byte, error} JSON recibido o indisponibilidad.
 */
func (r *Runtime) request(ctx context.Context, method, path string, value any) ([]byte, error) {
	var raw []byte
	var err error
	if value != nil {
		raw, err = json.Marshal(value)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, r.URL+path, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.Key)
	res, err := r.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 300 {
		return nil, errors.New("buscador de paneles no disponible")
	}
	return body, nil
}

/**
 * Espera la confirmación real de una tarea antes de limpiar el pendiente Mongo.
 *
 * @method task
 * @param {context.Context} ctx - Presupuesto de sincronización.
 * @param {string} method - Método de la tarea.
 * @param {string} path - Ruta de indexación.
 * @param {any} value - Datos enviados al motor.
 * @returns {error} nil solo cuando la tarea quedó succeeded.
 */
func (r *Runtime) task(ctx context.Context, method, path string, value any) error {
	body, err := r.request(ctx, method, path, value)
	if err != nil {
		return err
	}
	var task struct {
		ID int `json:"taskUid"`
	}
	if json.Unmarshal(body, &task) != nil {
		return errors.New("tarea inválida")
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(40 * time.Millisecond):
		}
		body, err = r.request(ctx, "GET", "/tasks/"+strconv.Itoa(task.ID), nil)
		if err != nil {
			return err
		}
		var state struct {
			Status string `json:"status"`
		}
		if json.Unmarshal(body, &state) != nil {
			return errors.New("estado inválido")
		}
		if state.Status == "succeeded" {
			return nil
		}
		if state.Status == "failed" || state.Status == "canceled" {
			return errors.New("indexación fallida")
		}
	}
}

/**
 * Prepara el índice independiente de paneles con filtros y órdenes acordados.
 *
 * @method setup
 * @param {context.Context} ctx - Presupuesto de inicialización.
 * @returns {error} Error de configuración recuperable.
 */
func (r *Runtime) setup(ctx context.Context) error {
	if _, err := r.request(ctx, "GET", "/indexes/paneles", nil); err != nil {
		if err = r.task(ctx, "POST", "/indexes", map[string]string{"uid": "paneles", "primaryKey": "id"}); err != nil {
			return err
		}
	}
	return r.task(ctx, "PATCH", "/indexes/paneles/settings", map[string]any{"searchableAttributes": []string{"nombre", "fuente"}, "filterableAttributes": []string{"activo", "potenciaW", "moneda", "precioCentavos", "superficieEfectivaMilliM2"}, "sortableAttributes": []string{"nombre", "id", "potenciaW", "precioCentavos"}, "pagination": map[string]int{"maxTotalHits": 1000000}})
}

/**
 * Sincroniza pendientes durables con leases por ficha y reintentos sin perder nuevas versiones.
 *
 * @method Run
 * @param {context.Context} ctx - Vida del proceso Solar.
 * @returns {void} Procesa en segundo plano hasta la cancelación.
 */
func (r *Runtime) Run(ctx context.Context) {
	platform.RetryLoop(ctx, func(ctx context.Context) error {
		setup, cancel := context.WithTimeout(ctx, 15*time.Second)
		err := r.setup(setup)
		cancel()
		if err != nil {
			return err
		}
		if err = r.Store.ReindexPanels(ctx); err != nil {
			return err
		}
		lastReconcile := time.Now()
		for {
			if ctx.Err() != nil {
				return nil
			}
			if time.Since(lastReconcile) >= time.Minute {
				repair, done := context.WithTimeout(ctx, 15*time.Second)
				err = r.setup(repair)
				if err == nil {
					err = r.Store.ReindexPanels(repair)
				}
				done()
				if err != nil {
					return err
				}
				lastReconcile = time.Now()
			}
			claim := platform.RandomID()
			work, cancel := context.WithTimeout(ctx, 10*time.Second)
			p, err := r.Store.ClaimPanelIndex(work, claim)
			if errors.Is(err, mongo.ErrNoDocuments) {
				cancel()
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(500 * time.Millisecond):
				}
				continue
			}
			if err != nil {
				cancel()
				return err
			}
			err = r.task(work, "POST", "/indexes/paneles/documents", []domain.Panel{p})
			cancel()
			finish, done := context.WithTimeout(ctx, 2*time.Second)
			saved := r.Store.FinishPanelIndex(finish, p, claim, err == nil)
			done()
			if saved != nil {
				return saved
			}
			if err != nil {
				return err
			}
			platform.Operations.WithLabelValues("paneles", "indexado").Inc()
		}
	})
}

/**
 * Busca paneles en Meilisearch y conserva lectura desde Mongo ante indisponibilidad.
 *
 * @method List
 * @param {context.Context} ctx - Presupuesto HTTP del usuario.
 * @param {Query} q - Texto, filtros, orden y página validados.
 * @returns {Page, string, error} Página, origen de lectura y error si falla también Mongo.
 */
func (r *Runtime) List(ctx context.Context, q Query) (Page, string, error) {
	filters := []string{"activo = true"}
	if q.Currency != "" {
		filters = append(filters, "moneda = "+q.Currency)
	}
	for _, f := range []struct {
		value     *int64
		field, op string
	}{{q.MinWatts, "potenciaW", ">="}, {q.MaxWatts, "potenciaW", "<="}, {q.MaxAreaMilli, "superficieEfectivaMilliM2", "<="}, {q.MaxCents, "precioCentavos", "<="}} {
		if f.value != nil {
			filters = append(filters, fmt.Sprintf("%s %s %d", f.field, f.op, *f.value))
		}
	}
	if q.MaxCents != nil || q.Order == "precio" {
		filters = append(filters, "precioCentavos > 0")
	}
	order := []string{"nombre:asc", "id:asc"}
	switch q.Order {
	case "potencia":
		order = []string{"potenciaW:asc", "id:asc"}
	case "-potencia":
		order = []string{"potenciaW:desc", "id:asc"}
	case "precio":
		order = []string{"precioCentavos:asc", "id:asc"}
	}
	body, err := r.request(ctx, "POST", "/indexes/paneles/search", map[string]any{"q": q.Text, "matchingStrategy": "all", "filter": strings.Join(filters, " AND "), "sort": order, "page": q.Page, "hitsPerPage": q.Size})
	if err == nil {
		var result struct {
			Items []domain.Panel `json:"hits"`
			Total int            `json:"totalHits"`
		}
		err = json.Unmarshal(body, &result)
		if err == nil {
			if result.Items == nil {
				result.Items = []domain.Panel{}
			}
			return Page{result.Items, result.Total, q.Page, q.Size}, "meilisearch", nil
		}
	}
	platform.Operations.WithLabelValues("paneles", "degradado_mongo").Inc()
	panels, err := r.Store.Panels(ctx, false)
	if err != nil {
		return Page{}, "mongo-degradado", err
	}
	return Fallback(q, panels), "mongo-degradado", nil
}
