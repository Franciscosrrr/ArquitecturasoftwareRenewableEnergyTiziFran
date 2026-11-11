package features

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/redis/go-redis/v9"
	"io"
	"net/http"
	"renewable.local/electrodomesticos/internal/repository"
	"renewable.local/platform"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

/**
 * Añade el estado de actividad a una ficha canónica destinada al índice derivado.
 *
 * @method Document
 * @returns {Document} Representación tipada usada por este componente.
 */
type Document struct {
	repository.Item
	Active bool `json:"activo"`
}

/**
 * Coordina índice Meilisearch, caché Valkey y outbox del catálogo SQL.
 *
 * @method Runtime
 * @returns {Runtime} Representación tipada usada por este componente.
 */
type Runtime struct {
	indexMu  sync.Mutex
	Store    *repository.Store
	Redis    *redis.Client
	HTTP     *http.Client
	URL, Key string
	Indexed  atomic.Bool
}

var ErrConflict = errors.New("conflicto de catálogo")

/**
 * Configura caché y buscador con presupuestos explícitos y sin retries ocultos.
 *
 * @method New
 * @param {*repository.Store} store - Repositorio canónico del componente, que conserva la propiedad de sus datos.
 * @returns {*Runtime} Instancia configurada del componente con sus dependencias y estado inicial.
 */
func New(store *repository.Store) *Runtime {
	return &Runtime{Store: store, Redis: redis.NewClient(&redis.Options{Addr: platform.Env("VALKEY_ADDR", "valkey:6379"), Password: platform.Secret("VALKEY_PASSWORD"), DialTimeout: 200 * time.Millisecond, ReadTimeout: 200 * time.Millisecond, WriteTimeout: 200 * time.Millisecond, PoolTimeout: 200 * time.Millisecond, ContextTimeoutEnabled: true, MaxRetries: -1, DialerRetries: 1, PoolSize: 10}), HTTP: platform.Client(2 * time.Second), URL: platform.Env("SEARCH_URL", "http://meilisearch:7700"), Key: platform.Secret("SEARCH_API_KEY")}
}

/**
 * Prepara la tabla outbox del catálogo sin modificar fichas existentes.
 *
 * @method Init
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @returns {error} nil al completar la operación; error propagado al llamador si no se completa.
 */
func (r *Runtime) Init(ctx context.Context) error {
	_, err := r.Store.DB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS catalogo_outbox(id CHAR(36) PRIMARY KEY,entity_id VARCHAR(64) NOT NULL,version BIGINT NOT NULL,created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,sent_at TIMESTAMP NULL,INDEX(sent_at))`)
	return err
}

/**
 * Lee el catálogo canónico incluyendo fichas inactivas para sincronizar el índice.
 *
 * @method All
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @returns {[]Document, error} Todas las fichas canónicas necesarias para reconstruir el índice, o error SQL.
 */
func (r *Runtime) All(ctx context.Context) ([]Document, error) {
	rows, err := r.Store.DB.QueryContext(ctx, "SELECT id,nombre,categoria,modo,potencia_w,energia_ciclo,factor,fuente,version,activo FROM electrodomesticos ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	docs := []Document{}
	for rows.Next() {
		var d Document
		var p, e, f string
		if err = rows.Scan(&d.ID, &d.Name, &d.Category, &d.Mode, &p, &e, &f, &d.Source, &d.Version, &d.Active); err != nil {
			return nil, err
		}
		d.Power = json.Number(p)
		d.Cycle = json.Number(e)
		d.Factor = json.Number(f)
		docs = append(docs, d)
	}
	return docs, rows.Err()
}

/**
 * Ejecuta una operación acotada y autenticada contra el motor de búsqueda.
 *
 * @method request
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {string} method - Método HTTP de la operación contra el motor.
 * @param {string} path - Ruta del campo JSON o de la operación HTTP indicada por este adaptador.
 * @param {any} value - Cuerpo JSON opcional que se envía al motor de búsqueda.
 * @returns {[]byte, error} Cuerpo HTTP acotado o error de transporte, lectura o estado no satisfactorio.
 */
func (r *Runtime) request(ctx context.Context, method, path string, value any) ([]byte, error) {
	var raw []byte
	if value != nil {
		raw, _ = json.Marshal(value)
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
		return nil, errors.New("motor de búsqueda no disponible")
	}
	return body, nil
}

/**
 * Espera la confirmación de una tarea asíncrona de Meilisearch.
 *
 * @method task
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {string} method - Método HTTP de la operación contra el motor.
 * @param {string} path - Ruta del campo JSON o de la operación HTTP indicada por este adaptador.
 * @param {any} value - Cuerpo JSON opcional que se envía al motor de búsqueda.
 * @returns {error} nil cuando Meilisearch confirma éxito; error o cancelación si la tarea no se completa.
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
 * Configura índices, filtros, campos de búsqueda y criterios de orden.
 *
 * @method setup
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {string} index - Nombre del índice derivado que se configura.
 * @returns {error} nil al completar la operación; error propagado al llamador si no se completa.
 */
func (r *Runtime) setup(ctx context.Context, index string) error {
	if _, err := r.request(ctx, "GET", "/indexes/"+index, nil); err != nil {
		if err = r.task(ctx, "POST", "/indexes", map[string]string{"uid": index, "primaryKey": "id"}); err != nil {
			return err
		}
	}
	return r.task(ctx, "PATCH", "/indexes/"+index+"/settings", map[string]any{"searchableAttributes": []string{"nombre", "categoria"}, "filterableAttributes": []string{"activo", "categoria", "modo", "potenciaW"}, "sortableAttributes": []string{"nombre", "potenciaW", "id"}, "pagination": map[string]int{"maxTotalHits": 1000000}})
}

/**
 * Construye un índice temporal y lo intercambia cuando todos los documentos están listos.
 *
 * @method Rebuild
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @returns {error} nil al completar la operación; error propagado al llamador si no se completa.
 */
func (r *Runtime) Rebuild(ctx context.Context) error {
	r.indexMu.Lock()
	defer r.indexMu.Unlock()
	if err := r.setup(ctx, "catalogo"); err != nil {
		return err
	}
	temporary := "catalogo_build_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if err := r.setup(ctx, temporary); err != nil {
		return err
	}
	docs, err := r.All(ctx)
	if err != nil {
		return err
	}
	if err = r.task(ctx, "POST", "/indexes/"+temporary+"/documents", docs); err != nil {
		return err
	}
	if err = r.task(ctx, "POST", "/swap-indexes", []map[string]any{{"indexes": []string{"catalogo", temporary}}}); err != nil {
		return err
	}
	_ = r.task(ctx, "DELETE", "/indexes/"+temporary, nil)
	r.Indexed.Store(true)
	_ = r.Redis.Incr(ctx, "catalogo:generacion").Err()
	platform.Operations.WithLabelValues("buscador", "reconstruido").Inc()
	return nil
}

/**
 * Consulta Meilisearch con filtros, orden y paginación explícitos.
 *
 * @method Search
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {repository.Query} q - Filtros, orden y límites de la consulta.
 * @returns {repository.Page, error} Página obtenida del índice o error de la búsqueda.
 */
func (r *Runtime) Search(ctx context.Context, q repository.Query) (repository.Page, error) {
	filter := []string{"activo = true"}
	if q.Category != "" {
		b, _ := json.Marshal(q.Category)
		filter = append(filter, "categoria = "+string(b))
	}
	if q.Mode != "" {
		b, _ := json.Marshal(q.Mode)
		filter = append(filter, "modo = "+string(b))
	}
	if q.MinPower != nil {
		filter = append(filter, fmt.Sprintf("modo = POTENCIA AND potenciaW >= %g", *q.MinPower))
	}
	if q.MaxPower != nil {
		filter = append(filter, fmt.Sprintf("modo = POTENCIA AND potenciaW <= %g", *q.MaxPower))
	}
	sort := []string{"nombre:asc", "id:asc"}
	if q.Sort == "potencia" {
		sort = []string{"potenciaW:asc", "id:asc"}
	}
	if q.Sort == "-potencia" {
		sort = []string{"potenciaW:desc", "id:asc"}
	}
	body, err := r.request(ctx, "POST", "/indexes/catalogo/search", map[string]any{"q": q.Text, "filter": strings.Join(filter, " AND "), "sort": sort, "page": q.Page, "hitsPerPage": q.Size})
	if err != nil {
		return repository.Page{}, err
	}
	var result struct {
		Items []repository.Item `json:"hits"`
		Total int               `json:"totalHits"`
	}
	if err = json.Unmarshal(body, &result); err != nil {
		return repository.Page{}, err
	}
	if result.Items == nil {
		result.Items = []repository.Item{}
	}
	return repository.Page{Items: result.Items, Total: result.Total, Page: q.Page, Size: q.Size}, nil
}

/**
 * Lista resultados con propiedad y límites definidos por el repositorio del servicio.
 *
 * @method List
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {repository.Query} q - Filtros, orden y límites de la consulta.
 * @param {bool} bypass - Omite la caché para comparar o verificar la lectura canónica.
 * @returns {repository.Page, string, error} Página, origen efectivo de la lectura y error si fallan índice y respaldo.
 */
func (r *Runtime) List(ctx context.Context, q repository.Query, bypass bool) (repository.Page, string, error) {
	cacheCtx, cancelCache := context.WithTimeout(ctx, 300*time.Millisecond)
	defer cancelCache()
	generation := "0"
	var err error
	cacheOK := false
	if !bypass {
		generation, err = r.Redis.Get(cacheCtx, "catalogo:generacion").Result()
		cacheOK = err == nil || errors.Is(err, redis.Nil)
	}
	if generation == "" {
		generation = "0"
	}
	raw, _ := json.Marshal(q)
	key := "catalogo:" + generation + ":" + platform.Digest(string(raw))
	if cacheOK && !bypass {
		cached, err := r.Redis.Get(cacheCtx, key).Bytes()
		if err == nil {
			var page repository.Page
			if json.Unmarshal(cached, &page) == nil {
				platform.Operations.WithLabelValues("cache", "hit").Inc()
				return page, "cache", nil
			}
		}
		platform.Operations.WithLabelValues("cache", "miss").Inc()
	} else if !cacheOK && !bypass {
		platform.Operations.WithLabelValues("cache", "error").Inc()
	}
	var page repository.Page
	source := "meilisearch"
	if r.Indexed.Load() {
		page, err = r.Search(ctx, q)
	} else {
		err = errors.New("índice inicializando")
	}
	if err != nil {
		platform.Operations.WithLabelValues("buscador", "degradado_sql").Inc()
		page, err = r.Store.List(ctx, q)
		source = "sql-degradado"
	}
	if err != nil {
		return page, source, err
	}
	if cacheOK && !bypass {
		data, _ := json.Marshal(page)
		if err = r.Redis.Set(cacheCtx, key, data, 60*time.Second).Err(); err != nil {
			platform.Operations.WithLabelValues("cache", "error").Inc()
		}
	}
	return page, source, nil
}

/**
 * Guarda la ficha y su evento de outbox en el mismo commit SQL.
 *
 * @method Save
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {repository.Item} item - Ficha canónica de equipo que debe guardarse junto con su versión.
 * @param {bool} active - Estado de actividad que debe conservar la ficha.
 * @param {bool} create - Crea una ficha nueva cuando es verdadero; de otro modo actualiza su versión.
 * @returns {repository.Item, error} Ficha creada o actualizada junto con su outbox, o error de versión/persistencia.
 */
func (r *Runtime) Save(ctx context.Context, item repository.Item, active bool, create bool) (repository.Item, error) {
	tx, err := r.Store.DB.BeginTx(ctx, nil)
	if err != nil {
		return item, err
	}
	defer tx.Rollback()
	if create {
		item.Version = 1
		_, err = tx.ExecContext(ctx, "INSERT INTO electrodomesticos(id,nombre,categoria,modo,potencia_w,energia_ciclo,factor,fuente,version,activo) VALUES(?,?,?,?,?,?,?,?,?,?)", item.ID, item.Name, item.Category, item.Mode, string(item.Power), string(item.Cycle), string(item.Factor), item.Source, item.Version, active)
	} else {
		res, e := tx.ExecContext(ctx, "UPDATE electrodomesticos SET nombre=?,categoria=?,modo=?,potencia_w=?,energia_ciclo=?,factor=?,fuente=?,activo=?,version=version+1 WHERE id=? AND version=?", item.Name, item.Category, item.Mode, string(item.Power), string(item.Cycle), string(item.Factor), item.Source, active, item.ID, item.Version)
		err = e
		if err == nil {
			n, _ := res.RowsAffected()
			if n == 0 {
				return item, ErrConflict
			}
		}
		item.Version++
	}
	if err != nil {
		return item, ErrConflict
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO catalogo_outbox(id,entity_id,version) VALUES(?,?,?)", platform.RandomID(), item.ID, item.Version); err != nil {
		return item, err
	}
	return item, tx.Commit()
}

/**
 * Publica la outbox durable y marca envíos solo después del publisher confirm.
 *
 * @method Relay
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @returns {error} nil al completar la operación; error propagado al llamador si no se completa.
 */
func (r *Runtime) Relay(ctx context.Context) error {
	b, err := platform.OpenBus()
	if err != nil {
		return err
	}
	defer b.Close()
	closed := b.Ch.NotifyClose(make(chan *amqp.Error, 1))
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-closed:
			return errors.New("conexión del publicador cerrada")
		case <-tick.C:
			rows, err := r.Store.DB.QueryContext(ctx, "SELECT id,entity_id,version FROM catalogo_outbox WHERE sent_at IS NULL ORDER BY created_at,id LIMIT 20")
			if err != nil {
				return err
			}
			type item struct {
				ID, Entity string
				Version    int64
			}
			items := []item{}
			for rows.Next() {
				var i item
				if err = rows.Scan(&i.ID, &i.Entity, &i.Version); err != nil {
					rows.Close()
					return err
				}
				items = append(items, i)
			}
			rows.Close()
			platform.State.WithLabelValues("outbox_catalogo", "pendientes").Set(float64(len(items)))
			for _, i := range items {
				raw, _ := json.Marshal(map[string]any{"eventId": i.ID, "tipo": "ElectrodomesticoActualizado.v1", "id": i.Entity, "version": i.Version})
				publish, cancel := context.WithTimeout(ctx, 3*time.Second)
				err = b.Publish(publish, "renewable.events", "catalogo.actualizado.v1", raw, nil, "")
				cancel()
				if err != nil {
					return err
				}
				if _, err = r.Store.DB.ExecContext(ctx, "UPDATE catalogo_outbox SET sent_at=CURRENT_TIMESTAMP WHERE id=?", i.ID); err != nil {
					return err
				}
			}
		}
	}
}

/**
 * Indexa la versión canónica actual e invalida la generación de caché tras confirmarla.
 *
 * @method IndexEvent
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {[]byte} raw - Cuerpo original recibido para validar.
 * @returns {error} nil al completar la operación; error propagado al llamador si no se completa.
 */
func (r *Runtime) IndexEvent(ctx context.Context, raw []byte) error {
	r.indexMu.Lock()
	defer r.indexMu.Unlock()
	var event struct {
		ID   string `json:"id"`
		Type string `json:"tipo"`
	}
	if json.Unmarshal(raw, &event) != nil || event.ID == "" || event.Type != "ElectrodomesticoActualizado.v1" {
		return &platform.PoisonError{Reason: "evento de catálogo inválido"}
	}
	docs, err := r.All(ctx)
	if err != nil {
		return err
	}
	for _, d := range docs {
		if d.ID == event.ID {
			if err = r.task(ctx, "POST", "/indexes/catalogo/documents", []Document{d}); err != nil {
				return err
			}
			if err = r.Redis.Incr(ctx, "catalogo:generacion").Err(); err != nil {
				return err
			}
			platform.Operations.WithLabelValues("buscador", "evento_indexado").Inc()
			return nil
		}
	}
	return &platform.PoisonError{Reason: "ficha inexistente"}
}

/**
 * Inicia sincronización y publicación recuperables en segundo plano.
 *
 * @method Start
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func (r *Runtime) Start(ctx context.Context) {
	go platform.RetryLoop(ctx, func(ctx context.Context) error {
		if !r.Indexed.Load() {
			rebuild, cancel := context.WithTimeout(ctx, 30*time.Second)
			err := r.Rebuild(rebuild)
			cancel()
			if err != nil {
				return err
			}
		}
		return platform.Consume(ctx, "catalogo.actualizado.v1", r.IndexEvent)
	})
	go platform.RetryLoop(ctx, r.Relay)
}
