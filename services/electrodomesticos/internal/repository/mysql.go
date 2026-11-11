package repository

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"strings"
	"time"
)

//go:embed schema.sql
var schema string

/**
 * Representa una ficha de equipo versionada, por potencia o por ciclos, con su fuente.
 *
 * @method Item
 * @returns {Item} Representación tipada usada por este componente.
 */
type Item struct {
	ID       string      `json:"id"`
	Name     string      `json:"nombre"`
	Category string      `json:"categoria"`
	Mode     string      `json:"modo"`
	Power    json.Number `json:"potenciaW"`
	Cycle    json.Number `json:"energiaPorCicloKWh"`
	Factor   json.Number `json:"factorFuncionamiento"`
	Source   string      `json:"fuente"`
	Version  int64       `json:"version"`
}

/**
 * Encapsula el pool SQL del catálogo de electrodomésticos y sus consultas canónicas.
 *
 * @method Store
 * @returns {Store} Representación tipada usada por este componente.
 */
type Store struct{ DB *sql.DB }

/**
 * Expresa búsqueda, filtros, orden y paginación del catálogo de equipos.
 *
 * @method Query
 * @returns {Query} Representación tipada usada por este componente.
 */
type Query struct {
	Text, Category, Mode, Sort string
	Page, Size                 int
	MinPower, MaxPower         *float64
}

/**
 * Devuelve fichas de equipos, total y límites de la página solicitada.
 *
 * @method Page
 * @returns {Page} Representación tipada usada por este componente.
 */
type Page struct {
	Items []Item `json:"items"`
	Total int    `json:"total"`
	Page  int    `json:"pagina"`
	Size  int    `json:"tamano"`
}

/**
 * Abre la infraestructura propia y prepara sus índices o conexiones.
 *
 * @method Open
 * @param {string} user - Usuario de conexión SQL o actor autenticado según el tipo declarado.
 * @param {string} password - Contraseña recibida o de conexión; nunca se incluye en la respuesta ni en logs.
 * @param {string} address - Dirección del servidor SQL, incluido su puerto.
 * @param {string} database - Nombre de la base que pertenece al servicio.
 * @returns {*Store, error} Repositorio con conexiones e índices preparados o error de inicialización.
 */
func Open(user, password, address, database string) (*Store, error) {
	cfg := mysql.NewConfig()
	cfg.User = user
	cfg.Passwd = password
	cfg.Net = "tcp"
	cfg.Addr = address
	cfg.DBName = database
	cfg.ParseTime = true
	cfg.Timeout = 2 * time.Second
	cfg.ReadTimeout = 2 * time.Second
	cfg.WriteTimeout = 2 * time.Second
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	return &Store{db}, nil
}

/**
 * Crea o migra el esquema propio de manera repetible sin eliminar datos existentes.
 *
 * @method Migrate
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @returns {error} nil al completar la operación; error propagado al llamador si no se completa.
 */
func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.DB.ExecContext(ctx, schema); err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	seeds := []struct {
		id, name, category, mode string
		power, cycle, factor     string
	}{
		{"iluminacion", "Lámpara LED de referencia", "Iluminación", "POTENCIA", "10", "0", "1"},
		{"heladera", "Heladera de referencia", "Refrigeración", "POTENCIA", "150", "0", "0.3"},
		{"lavarropas", "Lavarropas de referencia", "Lavado", "CICLOS", "0", "0.75", "1"},
		{"televisor", "Televisor de referencia", "Entretenimiento", "POTENCIA", "100", "0", "1"},
		{"computadora", "Computadora de referencia", "Informática", "POTENCIA", "200", "0", "1"},
		{"microondas", "Microondas de referencia", "Cocina", "POTENCIA", "1200", "0", "1"},
	}
	for _, seed := range seeds {
		_, err = tx.ExecContext(ctx, `INSERT IGNORE INTO electrodomesticos (id,nombre,categoria,modo,potencia_w,energia_ciclo,factor,fuente) VALUES (?,?,?,?,?,?,?,?)`, seed.id, seed.name, seed.category, seed.mode, seed.power, seed.cycle, seed.factor, "Referencia didáctica del proyecto; reemplazar por la ficha técnica real.")
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

const columns = "id,nombre,categoria,modo,potencia_w,energia_ciclo,factor,fuente,version"

/**
 * Convierte filas SQL a fichas tipadas conservando números decimales y versiones.
 *
 * @method scanRows
 * @param {*sql.Rows} rows - Filas SQL abiertas por el repositorio que se convertirán a fichas.
 * @returns {[]Item, error} Fichas decodificadas o error de lectura/conversión de las filas.
 */
func scanRows(rows *sql.Rows) ([]Item, error) {
	defer rows.Close()
	items := []Item{}
	for rows.Next() {
		var i Item
		var power, cycle, factor string
		if err := rows.Scan(&i.ID, &i.Name, &i.Category, &i.Mode, &power, &cycle, &factor, &i.Source, &i.Version); err != nil {
			return nil, err
		}
		i.Power = json.Number(power)
		i.Cycle = json.Number(cycle)
		i.Factor = json.Number(factor)
		items = append(items, i)
	}
	return items, rows.Err()
}

/**
 * Lista resultados con propiedad y límites definidos por el repositorio del servicio.
 *
 * @method List
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {Query} q - Filtros, orden y límites de la consulta.
 * @returns {Page, error} Página de fichas canónicas y error de lectura si no puede obtenerse.
 */
func (s *Store) List(ctx context.Context, q Query) (Page, error) {
	ctx, span := otel.Tracer("electrodomesticos").Start(ctx, "mysql.catalogo.listar")
	defer span.End()
	where := "activo=TRUE"
	args := []any{}
	if q.Text != "" {
		where += " AND nombre LIKE ?"
		escaped := strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(q.Text)
		where += " ESCAPE '!'"
		args = append(args, "%"+escaped+"%")
	}
	if q.Category != "" {
		where += " AND categoria=?"
		args = append(args, q.Category)
	}
	if q.Mode != "" {
		where += " AND modo=?"
		args = append(args, q.Mode)
	}
	if q.MinPower != nil {
		where += " AND modo='POTENCIA' AND potencia_w>=?"
		args = append(args, *q.MinPower)
	}
	if q.MaxPower != nil {
		where += " AND modo='POTENCIA' AND potencia_w<=?"
		args = append(args, *q.MaxPower)
	}
	order := map[string]string{"nombre": "nombre,id", "potencia": "potencia_w,id", "-potencia": "potencia_w DESC,id"}[q.Sort]
	if order == "" {
		order = "nombre,id"
	}
	p := Page{Items: []Item{}, Page: q.Page, Size: q.Size}
	// A repeatable-read transaction keeps count and page from different snapshots apart.
	tx, err := s.DB.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return p, err
	}
	defer tx.Rollback()
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM electrodomesticos WHERE "+where, args...).Scan(&p.Total); err != nil {
		span.SetStatus(codes.Error, "consulta fallida")
		return p, err
	}
	args = append(args, q.Size, (q.Page-1)*q.Size)
	rows, err := tx.QueryContext(ctx, "SELECT "+columns+" FROM electrodomesticos WHERE "+where+" ORDER BY "+order+" LIMIT ? OFFSET ?", args...)
	if err != nil {
		return p, err
	}
	p.Items, err = scanRows(rows)
	if err != nil {
		return p, err
	}
	return p, tx.Commit()
}

/**
 * Obtiene fichas canónicas activas para calcular con sus versiones exactas.
 *
 * @method Resolve
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {[]string} ids - Identificadores únicos de las fichas a recuperar en lote.
 * @returns {[]Item, error} Fichas activas solicitadas o error de validación, ausencia o dependencia.
 */
func (s *Store) Resolve(ctx context.Context, ids []string) ([]Item, error) {
	ctx, span := otel.Tracer("electrodomesticos").Start(ctx, "mysql.catalogo.resolver")
	defer span.End()
	if len(ids) < 1 || len(ids) > 100 {
		return nil, fmt.Errorf("cantidad inválida")
	}
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	rows, err := s.DB.QueryContext(ctx, "SELECT "+columns+" FROM electrodomesticos WHERE activo=TRUE AND id IN ("+strings.TrimSuffix(strings.Repeat("?,", len(ids)), ",")+")", args...)
	if err != nil {
		span.SetStatus(codes.Error, "consulta fallida")
		return nil, err
	}
	return scanRows(rows)
}
