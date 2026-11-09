package private

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	amqp "github.com/rabbitmq/amqp091-go"
	"renewable.local/platform"
	"time"
)

var ErrConflict = errors.New("conflicto de versión o idempotencia")
var ErrNotFound = errors.New("recurso inexistente")
var ErrPlaceInUse = errors.New("el lugar tiene consumos guardados")

type Place struct {
	ID       string   `json:"id"`
	Name     string   `json:"nombre"`
	Location string   `json:"ubicacion"`
	Area     *float64 `json:"superficieM2"`
	Version  int64    `json:"version"`
	Archived bool     `json:"archivado"`
}
type Config struct {
	ID       string          `json:"id"`
	PlaceID  string          `json:"lugarId"`
	Name     string          `json:"nombre"`
	Version  int64           `json:"version"`
	Input    json.RawMessage `json:"consumo"`
	Snapshot json.RawMessage `json:"snapshot"`
	Archived bool            `json:"archivada"`
}
type Evaluation struct {
	ID           string          `json:"id"`
	ConfigID     string          `json:"configuracionId"`
	Version      int64           `json:"versionConfiguracion"`
	PlaceVersion int64           `json:"versionLugar"`
	Snapshot     json.RawMessage `json:"snapshot"`
	Studies      []string        `json:"estudioIds"`
	Publication  string          `json:"publicacion"`
}
type SolarOptions struct {
	Sun      float64  `json:"horasSolPico"`
	Yield    float64  `json:"factorRendimiento"`
	Coverage float64  `json:"coberturaObjetivo"`
	Area     *float64 `json:"superficieM2"`
}
type Event struct {
	ID           string          `json:"eventId"`
	Type         string          `json:"tipo"`
	StudyID      string          `json:"estudioId"`
	EvaluationID string          `json:"evaluacionId"`
	Owner        string          `json:"propietarioId"`
	Snapshot     json.RawMessage `json:"snapshot"`
	Solar        SolarOptions    `json:"solar"`
	Traceparent  string          `json:"traceparent"`
	RequestID    string          `json:"requestId"`
}
type Store struct{ DB *sql.DB }

/**
 * Crea o migra el esquema propio de manera repetible sin eliminar datos existentes.
 *
 * @method Migrate
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @returns {error} Resultado de la operación; error si no puede completarse.
 */
func (s *Store) Migrate(ctx context.Context) error {
	for _, q := range []string{
		`CREATE TABLE IF NOT EXISTS lugares(id CHAR(36) PRIMARY KEY,owner_id CHAR(36) NOT NULL,nombre VARCHAR(100) NOT NULL,ubicacion VARCHAR(200) NOT NULL,superficie DOUBLE NULL,version BIGINT NOT NULL DEFAULT 1,archivado BOOLEAN NOT NULL DEFAULT FALSE,INDEX(owner_id))`,
		`CREATE TABLE IF NOT EXISTS configuraciones(id CHAR(36) PRIMARY KEY,owner_id CHAR(36) NOT NULL,lugar_id CHAR(36) NOT NULL,nombre VARCHAR(100) NOT NULL,version BIGINT NOT NULL,entrada JSON NOT NULL,snapshot JSON NOT NULL,archivada BOOLEAN NOT NULL DEFAULT FALSE,INDEX(owner_id),FOREIGN KEY(lugar_id) REFERENCES lugares(id))`,
		`CREATE TABLE IF NOT EXISTS evaluaciones(id CHAR(36) PRIMARY KEY,owner_id CHAR(36) NOT NULL,config_id CHAR(36) NOT NULL,config_version BIGINT NOT NULL,place_version BIGINT NOT NULL,snapshot JSON NOT NULL,created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,UNIQUE(config_id,config_version,place_version),INDEX(owner_id))`,
		`CREATE TABLE IF NOT EXISTS idempotencia(owner_id CHAR(36) NOT NULL,clave VARCHAR(128) NOT NULL,hash CHAR(64) NOT NULL,evaluacion_id CHAR(36) NOT NULL,PRIMARY KEY(owner_id,clave))`,
		`CREATE TABLE IF NOT EXISTS outbox(id CHAR(36) PRIMARY KEY,owner_id CHAR(36) NOT NULL,evaluacion_id CHAR(36) NOT NULL,payload JSON NOT NULL,created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,sent_at TIMESTAMP NULL,lease_until BIGINT NOT NULL DEFAULT 0,attempts INT NOT NULL DEFAULT 0,INDEX(sent_at,lease_until))`,
	} {
		if _, err := s.DB.ExecContext(ctx, q); err != nil {
			return err
		}
	}
	for _, table := range []string{"lugares", "configuraciones"} {
		var exists int
		if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name=? AND column_name='eliminado_at'", table).Scan(&exists); err != nil {
			return err
		}
		if exists == 0 {
			if _, err := s.DB.ExecContext(ctx, "ALTER TABLE "+table+" ADD COLUMN eliminado_at TIMESTAMP NULL"); err != nil {
				return err
			}
		}
	}
	var hasPlaceVersion int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name='evaluaciones' AND column_name='place_version'").Scan(&hasPlaceVersion); err != nil {
		return err
	}
	if hasPlaceVersion == 0 {
		if _, err := s.DB.ExecContext(ctx, "ALTER TABLE evaluaciones ADD COLUMN place_version BIGINT NOT NULL DEFAULT 1"); err != nil {
			return err
		}
		if _, err := s.DB.ExecContext(ctx, "UPDATE evaluaciones SET place_version=COALESCE(JSON_EXTRACT(snapshot,'$.lugar.version'),1)"); err != nil {
			return err
		}
	}
	var indexedColumns int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema=DATABASE() AND table_name='evaluaciones' AND index_name='config_id'").Scan(&indexedColumns); err != nil {
		return err
	}
	if indexedColumns == 2 {
		if _, err := s.DB.ExecContext(ctx, "ALTER TABLE evaluaciones DROP INDEX config_id, ADD UNIQUE KEY config_id(config_id,config_version,place_version)"); err != nil {
			return err
		}
	}
	return nil
}

/**
 * Lista lugares pertenecientes exclusivamente al propietario autenticado.
 *
 * @method Places
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {string} owner - Identificador del propietario autenticado.
 * @returns {[]Place, error} Resultado de la operación; error si no puede completarse.
 */
func (s *Store) Places(ctx context.Context, owner string) ([]Place, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT id,nombre,ubicacion,superficie,version,archivado FROM lugares WHERE owner_id=? AND eliminado_at IS NULL ORDER BY nombre,id", owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Place{}
	for rows.Next() {
		var p Place
		if err = rows.Scan(&p.ID, &p.Name, &p.Location, &p.Area, &p.Version, &p.Archived); err != nil {
			return nil, err
		}
		items = append(items, p)
	}
	return items, rows.Err()
}

/**
 * Recupera un lugar propio o informa ausencia sin revelar datos ajenos.
 *
 * @method Place
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {string} owner - Identificador del propietario autenticado.
 * @param {string} id - Identificador del recurso solicitado.
 * @returns {Place, error} Resultado de la operación; error si no puede completarse.
 */
func (s *Store) Place(ctx context.Context, owner, id string) (Place, error) {
	var p Place
	err := s.DB.QueryRowContext(ctx, "SELECT id,nombre,ubicacion,superficie,version,archivado FROM lugares WHERE owner_id=? AND id=? AND eliminado_at IS NULL", owner, id).Scan(&p.ID, &p.Name, &p.Location, &p.Area, &p.Version, &p.Archived)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	return p, err
}

/**
 * Crea o actualiza un lugar usando comparación de versión y propietario.
 *
 * @method SavePlace
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {string} owner - Identificador del propietario autenticado.
 * @param {Place} p - Lugar o panel a persistir con su versión.
 * @returns {Place, error} Resultado de la operación; error si no puede completarse.
 */
func (s *Store) SavePlace(ctx context.Context, owner string, p Place) (Place, error) {
	if p.ID == "" {
		p.ID = platform.RandomID()
		p.Version = 1
		_, err := s.DB.ExecContext(ctx, "INSERT INTO lugares(id,owner_id,nombre,ubicacion,superficie) VALUES(?,?,?,?,?)", p.ID, owner, p.Name, p.Location, p.Area)
		return p, err
	}
	res, err := s.DB.ExecContext(ctx, "UPDATE lugares SET nombre=?,ubicacion=?,superficie=?,archivado=?,version=version+1 WHERE id=? AND owner_id=? AND version=? AND eliminado_at IS NULL", p.Name, p.Location, p.Area, p.Archived, p.ID, owner, p.Version)
	if err != nil {
		return p, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return p, ErrConflict
	}
	p.Version++
	return p, nil
}

/**
 * Recupera una configuración propia con su entrada y snapshot guardados.
 *
 * @method Config
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {string} owner - Identificador del propietario autenticado.
 * @param {string} id - Identificador del recurso solicitado.
 * @returns {Config, error} Resultado de la operación; error si no puede completarse.
 */
func (s *Store) Config(ctx context.Context, owner, id string) (Config, error) {
	var c Config
	err := s.DB.QueryRowContext(ctx, "SELECT id,lugar_id,nombre,version,entrada,snapshot,archivada FROM configuraciones WHERE owner_id=? AND id=? AND eliminado_at IS NULL", owner, id).Scan(&c.ID, &c.PlaceID, &c.Name, &c.Version, &c.Input, &c.Snapshot, &c.Archived)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	return c, err
}

/**
 * Lista configuraciones propias sin mezclar datos de otros usuarios.
 *
 * @method Configs
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {string} owner - Identificador del propietario autenticado.
 * @returns {[]Config, error} Resultado de la operación; error si no puede completarse.
 */
func (s *Store) Configs(ctx context.Context, owner string) ([]Config, error) {
	rows, err := s.DB.QueryContext(ctx, "SELECT id,lugar_id,nombre,version,entrada,snapshot,archivada FROM configuraciones WHERE owner_id=? AND eliminado_at IS NULL ORDER BY nombre,id", owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Config{}
	for rows.Next() {
		var c Config
		if err = rows.Scan(&c.ID, &c.PlaceID, &c.Name, &c.Version, &c.Input, &c.Snapshot, &c.Archived); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

/**
 * Persiste un escenario versionado después de validar y bloquear el lugar propio.
 *
 * @method SaveConfig
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {string} owner - Identificador del propietario autenticado.
 * @param {Config} c - Solicitud HTTP en curso.
 * @returns {Config, error} Resultado de la operación; error si no puede completarse.
 */
func (s *Store) SaveConfig(ctx context.Context, owner string, c Config) (Config, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return c, err
	}
	defer tx.Rollback()
	var archived bool
	if err = tx.QueryRowContext(ctx, "SELECT archivado FROM lugares WHERE id=? AND owner_id=? AND eliminado_at IS NULL FOR UPDATE", c.PlaceID, owner).Scan(&archived); err != nil || archived {
		return c, ErrNotFound
	}
	if c.ID == "" {
		c.ID = platform.RandomID()
		c.Version = 1
		_, err = tx.ExecContext(ctx, "INSERT INTO configuraciones(id,owner_id,lugar_id,nombre,version,entrada,snapshot) VALUES(?,?,?,?,?,?,?)", c.ID, owner, c.PlaceID, c.Name, c.Version, []byte(c.Input), []byte(c.Snapshot))
	} else {
		var res sql.Result
		res, err = tx.ExecContext(ctx, "UPDATE configuraciones SET lugar_id=?,nombre=?,entrada=?,snapshot=?,archivada=?,version=version+1 WHERE id=? AND owner_id=? AND version=? AND eliminado_at IS NULL", c.PlaceID, c.Name, []byte(c.Input), []byte(c.Snapshot), c.Archived, c.ID, owner, c.Version)
		if err == nil {
			n, _ := res.RowsAffected()
			if n == 0 {
				err = ErrConflict
			}
		}
		c.Version++
	}
	if err != nil {
		return c, err
	}
	return c, tx.Commit()
}

/**
 * Recupera el snapshot inmutable y el estado de publicación de sus estudios.
 *
 * @method Evaluation
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {string} owner - Identificador del propietario autenticado.
 * @param {string} id - Identificador del recurso solicitado.
 * @returns {Evaluation, error} Resultado de la operación; error si no puede completarse.
 */
func (s *Store) Evaluation(ctx context.Context, owner, id string) (Evaluation, error) {
	var e Evaluation
	err := s.DB.QueryRowContext(ctx, "SELECT id,config_id,config_version,place_version,snapshot FROM evaluaciones WHERE owner_id=? AND id=?", owner, id).Scan(&e.ID, &e.ConfigID, &e.Version, &e.PlaceVersion, &e.Snapshot)
	if errors.Is(err, sql.ErrNoRows) {
		return e, ErrNotFound
	}
	if err != nil {
		return e, err
	}
	e.Studies = []string{}
	e.Publication = "ENVIADA"
	rows, err := s.DB.QueryContext(ctx, "SELECT payload,sent_at FROM outbox WHERE owner_id=? AND evaluacion_id=? ORDER BY created_at,id", owner, id)
	if err != nil {
		return e, err
	}
	defer rows.Close()
	for rows.Next() {
		var raw []byte
		var sent sql.NullTime
		if err = rows.Scan(&raw, &sent); err != nil {
			return e, err
		}
		var event Event
		if json.Unmarshal(raw, &event) != nil {
			return e, ErrConflict
		}
		e.Studies = append(e.Studies, event.StudyID)
		if !sent.Valid {
			e.Publication = "PENDIENTE"
		}
	}
	return e, rows.Err()
}

/**
 * Compone el historial propio de evaluaciones conservando versiones anteriores.
 *
 * @method History
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {string} owner - Identificador del propietario autenticado.
 * @param {string} config - Valor de config requerido por la operación.
 * @returns {[]Evaluation, error} Resultado de la operación; error si no puede completarse.
 */
func (s *Store) History(ctx context.Context, owner, config string) ([]Evaluation, error) {
	query := "SELECT id FROM evaluaciones WHERE owner_id=?"
	args := []any{owner}
	if config != "" {
		query += " AND config_id=?"
		args = append(args, config)
	}
	rows, err := s.DB.QueryContext(ctx, query+" ORDER BY created_at DESC,id", args...)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	result := []Evaluation{}
	for _, id := range ids {
		e, err := s.Evaluation(ctx, owner, id)
		if err != nil {
			return nil, err
		}
		result = append(result, e)
	}
	return result, nil
}

/**
 * Busca una clave de idempotencia y exige que su hash coincida.
 *
 * @method Existing
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {string} owner - Identificador del propietario autenticado.
 * @param {string} key - Clave de idempotencia o identidad técnica.
 * @param {string} hash - Huella de la entrada canónica.
 * @returns {string, error} Resultado de la operación; error si no puede completarse.
 */
func (s *Store) Existing(ctx context.Context, owner, key, hash string) (string, error) {
	var previous, id string
	err := s.DB.QueryRowContext(ctx, "SELECT hash,evaluacion_id FROM idempotencia WHERE owner_id=? AND clave=?", owner, key).Scan(&previous, &id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if previous != hash {
		return "", ErrConflict
	}
	return id, nil
}

/**
 * Confirma evaluación, idempotencia y outbox en una transacción de versiones coherentes.
 *
 * @method Confirm
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {string} owner - Identificador del propietario autenticado.
 * @param {string} key - Clave de idempotencia o identidad técnica.
 * @param {string} hash - Huella de la entrada canónica.
 * @param {Config} c - Solicitud HTTP en curso.
 * @param {int64} placeVersion - Valor de placeVersion requerido por la operación.
 * @param {Event} event - Evento de negocio que debe procesarse.
 * @returns {string, error} Resultado de la operación; error si no puede completarse.
 */
func (s *Store) Confirm(ctx context.Context, owner, key, hash string, c Config, placeVersion int64, event Event) (string, error) {
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var current int64
	var snapshot []byte
	var archived bool
	err = tx.QueryRowContext(ctx, "SELECT version,snapshot,archivada FROM configuraciones WHERE owner_id=? AND id=? AND eliminado_at IS NULL FOR UPDATE", owner, c.ID).Scan(&current, &snapshot, &archived)
	if err != nil {
		return "", ErrNotFound
	}
	if current != c.Version || archived {
		return "", ErrConflict
	}
	var place Place
	if err = tx.QueryRowContext(ctx, "SELECT id,nombre,ubicacion,superficie,version,archivado FROM lugares WHERE id=? AND owner_id=? AND eliminado_at IS NULL FOR UPDATE", c.PlaceID, owner).Scan(&place.ID, &place.Name, &place.Location, &place.Area, &place.Version, &place.Archived); err != nil || place.Archived {
		return "", ErrNotFound
	}
	if place.Version != placeVersion {
		return "", ErrConflict
	}
	var frozen map[string]json.RawMessage
	if json.Unmarshal(snapshot, &frozen) != nil {
		return "", ErrConflict
	}
	frozen["lugar"], _ = json.Marshal(place)
	snapshot, _ = json.Marshal(frozen)
	event.Snapshot = snapshot
	var previous, id string
	err = tx.QueryRowContext(ctx, "SELECT hash,evaluacion_id FROM idempotencia WHERE owner_id=? AND clave=? FOR UPDATE", owner, key).Scan(&previous, &id)
	if err == nil {
		if previous != hash {
			return "", ErrConflict
		}
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	err = tx.QueryRowContext(ctx, "SELECT id FROM evaluaciones WHERE config_id=? AND config_version=? AND place_version=?", c.ID, c.Version, placeVersion).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		id = platform.RandomID()
		if _, err = tx.ExecContext(ctx, "INSERT INTO evaluaciones(id,owner_id,config_id,config_version,place_version,snapshot) VALUES(?,?,?,?,?,?)", id, owner, c.ID, c.Version, placeVersion, snapshot); err != nil {
			return "", err
		}
		event.EvaluationID = id
		raw, _ := json.Marshal(event)
		if _, err = tx.ExecContext(ctx, "INSERT INTO outbox(id,owner_id,evaluacion_id,payload) VALUES(?,?,?,?)", event.ID, owner, id, raw); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	} else {
		var priorRaw []byte
		if err = tx.QueryRowContext(ctx, "SELECT payload FROM outbox WHERE evaluacion_id=? ORDER BY created_at,id LIMIT 1", id).Scan(&priorRaw); err != nil {
			return "", err
		}
		var prior Event
		if json.Unmarshal(priorRaw, &prior) != nil {
			return "", ErrConflict
		}
		a, _ := json.Marshal(prior.Solar)
		b, _ := json.Marshal(event.Solar)
		if string(a) != string(b) {
			return "", ErrConflict
		}
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO idempotencia(owner_id,clave,hash,evaluacion_id) VALUES(?,?,?,?)", owner, key, hash, id); err != nil {
		return "", err
	}
	return id, tx.Commit()
}

/**
 * Crea una variante solar idempotente sin modificar la evaluación anterior.
 *
 * @method AddStudy
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {string} owner - Identificador del propietario autenticado.
 * @param {string} id - Identificador del recurso solicitado.
 * @param {string} key - Clave de idempotencia o identidad técnica.
 * @param {Event} event - Evento de negocio que debe procesarse.
 * @returns {string, error} Resultado de la operación; error si no puede completarse.
 */
func (s *Store) AddStudy(ctx context.Context, owner, id, key string, event Event) (string, error) {
	e, err := s.Evaluation(ctx, owner, id)
	if err != nil {
		return "", err
	}
	event.EvaluationID = id
	event.Snapshot = e.Snapshot
	raw, _ := json.Marshal(event)
	event.ID = platform.Digest(owner + ":" + id + ":" + key)[:36]
	event.StudyID = event.ID
	raw, _ = json.Marshal(event)
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	var old []byte
	err = tx.QueryRowContext(ctx, "SELECT payload FROM outbox WHERE id=? FOR UPDATE", event.ID).Scan(&old)
	if err == nil {
		var prior Event
		_ = json.Unmarshal(old, &prior)
		a, _ := json.Marshal(prior.Solar)
		b, _ := json.Marshal(event.Solar)
		if string(a) != string(b) {
			return "", ErrConflict
		}
		return prior.StudyID, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	if _, err = tx.ExecContext(ctx, "INSERT INTO outbox(id,owner_id,evaluacion_id,payload) VALUES(?,?,?,?)", event.ID, owner, id, raw); err != nil {
		return "", err
	}
	return event.StudyID, tx.Commit()
}

/**
 * Publica la outbox durable y marca envíos solo después del publisher confirm.
 *
 * @method Relay
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @returns {error} Resultado de la operación; error si no puede completarse.
 */
func (s *Store) Relay(ctx context.Context) error {
	b, err := platform.OpenBus()
	if err != nil {
		return err
	}
	defer b.Close()
	closed := b.Ch.NotifyClose(make(chan *amqp.Error, 1))
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-closed:
			return errors.New("conexión del publicador cerrada")
		case <-ticker.C:
			var pending int
			_ = s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM outbox WHERE sent_at IS NULL").Scan(&pending)
			platform.State.WithLabelValues("outbox_consumo", "pendientes").Set(float64(pending))
			rows, err := s.DB.QueryContext(ctx, "SELECT id,payload FROM outbox WHERE sent_at IS NULL AND lease_until<? ORDER BY created_at,id LIMIT 20", time.Now().Unix())
			if err != nil {
				return err
			}
			type item struct {
				ID  string
				Raw []byte
			}
			batch := []item{}
			for rows.Next() {
				var i item
				if err = rows.Scan(&i.ID, &i.Raw); err != nil {
					rows.Close()
					return err
				}
				batch = append(batch, i)
			}
			rows.Close()
			for _, i := range batch {
				res, err := s.DB.ExecContext(ctx, "UPDATE outbox SET lease_until=?,attempts=attempts+1 WHERE id=? AND sent_at IS NULL AND lease_until<?", time.Now().Add(10*time.Second).Unix(), i.ID, time.Now().Unix())
				if err != nil {
					return err
				}
				n, _ := res.RowsAffected()
				if n == 0 {
					continue
				}
				publish, cancel := context.WithTimeout(ctx, 3*time.Second)
				err = b.Publish(publish, "renewable.events", "consumo.evaluado.v1", i.Raw, nil, "")
				cancel()
				if err != nil {
					return err
				}
				if _, err = s.DB.ExecContext(ctx, "UPDATE outbox SET sent_at=CURRENT_TIMESTAMP,lease_until=0 WHERE id=?", i.ID); err != nil {
					return err
				}
				platform.Operations.WithLabelValues("outbox_consumo", "publicado").Inc()
			}
		}
	}
}
