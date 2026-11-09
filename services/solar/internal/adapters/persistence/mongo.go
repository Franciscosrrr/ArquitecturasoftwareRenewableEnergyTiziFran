package persistence

import (
	"context"
	"encoding/json"
	"errors"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
	"renewable.local/platform"
	"renewable.local/solar/internal/domain"
	"time"
)

var ErrConflict = errors.New("conflicto de versión")
var ErrBusy = errors.New("estudio en procesamiento")

type Study struct {
	ID           string                 `json:"id" bson:"_id"`
	EventID      string                 `json:"eventId"`
	EvaluationID string                 `json:"evaluacionId"`
	Owner        string                 `json:"-" bson:"owner"`
	Hash         string                 `json:"-" bson:"hash"`
	Status       string                 `json:"estado"`
	Snapshot     json.RawMessage        `json:"snapshot"`
	Result       *domain.Recommendation `json:"resultado"`
	Lease        time.Time              `json:"-" bson:"lease"`
	Updated      time.Time              `json:"actualizado"`
	TraceID      string                 `json:"traceId"`
	Error        string                 `json:"error,omitempty"`
}
type Store struct {
	Client *mongo.Client
	DB     *mongo.Database
}

/**
 * Abre la infraestructura propia y prepara sus índices o conexiones.
 *
 * @method Open
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @returns {*Store, error} Resultado de la operación; error si no puede completarse.
 */
func Open(ctx context.Context) (*Store, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(platform.Env("MONGO_URI", "")).SetTimeout(3 * time.Second).SetWriteConcern(writeconcern.Journaled()).SetBSONOptions(&options.BSONOptions{UseJSONStructTags: true}))
	if err != nil {
		return nil, err
	}
	store := &Store{Client: client, DB: client.Database("solar_db")}
	if err = client.Ping(ctx, nil); err != nil {
		return nil, err
	}
	_, err = store.DB.Collection("estudios").Indexes().CreateMany(ctx, []mongo.IndexModel{{Keys: bson.D{{Key: "eventId", Value: 1}}, Options: options.Index().SetUnique(true)}, {Keys: bson.D{{Key: "owner", Value: 1}, {Key: "evaluacionId", Value: 1}}}})
	if err != nil {
		return nil, err
	}
	for _, p := range []domain.Panel{{ID: "PANEL-SIM-450", Name: "Panel de referencia 450 W", Source: "Referencia didáctica del simulador; verificar ficha técnica real", Watts: 450, AreaMilli: 2500, UnitCents: 25000000, Currency: "ARS", Active: true, Version: 1}, {ID: "PANEL-SIM-550", Name: "Panel de referencia 550 W", Source: "Referencia didáctica del simulador; verificar ficha técnica real", Watts: 550, AreaMilli: 2800, UnitCents: 30000000, Currency: "ARS", Active: true, Version: 1}} {
		if _, err = store.DB.Collection("paneles").UpdateOne(ctx, bson.M{"_id": p.ID}, bson.M{"$setOnInsert": p}, options.UpdateOne().SetUpsert(true)); err != nil {
			return nil, err
		}
	}
	_, err = store.DB.Collection("paneles").UpdateMany(ctx, bson.M{"busquedaPendiente": bson.M{"$exists": false}}, bson.M{"$set": bson.M{"busquedaPendiente": true}})
	return store, err
}

/**
 * Comprueba si la dependencia necesaria puede atender la operación.
 *
 * @method Ready
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @returns {error} Resultado de la operación; error si no puede completarse.
 */
func (s *Store) Ready(ctx context.Context) error { return s.Client.Ping(ctx, nil) }

/**
 * Recupera fichas solares versionadas con filtro de actividad opcional.
 *
 * @method Panels
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {bool} all - Valor de all requerido por la operación.
 * @returns {[]domain.Panel, error} Resultado de la operación; error si no puede completarse.
 */
func (s *Store) Panels(ctx context.Context, all bool) ([]domain.Panel, error) {
	filter := bson.M{}
	if !all {
		filter["activo"] = true
	}
	cursor, err := s.DB.Collection("paneles").Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "_id", Value: 1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	panels := []domain.Panel{}
	err = cursor.All(ctx, &panels)
	return panels, err
}

/**
 * Crea o actualiza una ficha Mongo con comparación atómica de versión.
 *
 * @method SavePanel
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {domain.Panel} p - Lugar o panel a persistir con su versión.
 * @param {bool} create - Valor de create requerido por la operación.
 * @returns {domain.Panel, error} Resultado de la operación; error si no puede completarse.
 */
func (s *Store) SavePanel(ctx context.Context, p domain.Panel, create bool) (domain.Panel, error) {
	document := bson.M{"_id": p.ID, "nombre": p.Name, "fuente": p.Source, "potenciaW": p.Watts, "superficieEfectivaMilliM2": p.AreaMilli, "precioCentavos": p.UnitCents, "moneda": p.Currency, "activo": p.Active, "busquedaPendiente": true}
	if create {
		p.Version = 1
		document["version"] = p.Version
		_, err := s.DB.Collection("paneles").InsertOne(ctx, document)
		if mongo.IsDuplicateKeyError(err) {
			return p, ErrConflict
		}
		return p, err
	}
	expected := p.Version
	p.Version++
	delete(document, "_id")
	document["version"] = p.Version
	result, err := s.DB.Collection("paneles").UpdateOne(ctx, bson.M{"_id": p.ID, "version": expected}, bson.M{"$set": document})
	if err != nil {
		return p, err
	}
	if result.MatchedCount == 0 {
		return p, ErrConflict
	}
	return p, nil
}

/**
 * Recupera un estudio propio mediante filtro de propietario en Mongo.
 *
 * @method Get
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {string} owner - Identificador del propietario autenticado.
 * @param {string} id - Identificador del recurso solicitado.
 * @returns {Study, error} Resultado de la operación; error si no puede completarse.
 */
func (s *Store) Get(ctx context.Context, owner, id string) (Study, error) {
	var study Study
	err := s.DB.Collection("estudios").FindOne(ctx, bson.M{"_id": id, "owner": owner}).Decode(&study)
	return study, err
}

/**
 * Lista resultados con propiedad y límites definidos por el repositorio del servicio.
 *
 * @method List
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {string} owner - Identificador del propietario autenticado.
 * @param {string} evaluation - Valor de evaluation requerido por la operación.
 * @returns {[]Study, error} Resultado de la operación; error si no puede completarse.
 */
func (s *Store) List(ctx context.Context, owner, evaluation string) ([]Study, error) {
	filter := bson.M{"owner": owner}
	if evaluation != "" {
		filter["evaluacionId"] = evaluation
	}
	cursor, err := s.DB.Collection("estudios").Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "actualizado", Value: -1}}).SetLimit(100))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	items := []Study{}
	err = cursor.All(ctx, &items)
	return items, err
}

/**
 * Deduplica el evento y obtiene un lease atómico para procesar su estudio.
 *
 * @method Claim
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {Study} study - Valor de study requerido por la operación.
 * @returns {bool, error} Resultado de la operación; error si no puede completarse.
 */
func (s *Store) Claim(ctx context.Context, study Study) (bool, error) {
	study.Status = "PENDIENTE"
	study.Updated = time.Now().UTC()
	_, err := s.DB.Collection("estudios").UpdateOne(ctx, bson.M{"_id": study.ID}, bson.M{"$setOnInsert": study}, options.UpdateOne().SetUpsert(true))
	if err != nil {
		return false, err
	}
	var prior Study
	if err = s.DB.Collection("estudios").FindOne(ctx, bson.M{"_id": study.ID}).Decode(&prior); err != nil {
		return false, err
	}
	if prior.Hash != study.Hash || prior.EventID != study.EventID {
		return false, &platform.PoisonError{Reason: "eventId con otro contenido"}
	}
	if prior.Status == "COMPLETADO" || prior.Status == "SIN_ALTERNATIVA" {
		return false, nil
	}
	result, err := s.DB.Collection("estudios").UpdateOne(ctx, bson.M{"_id": study.ID, "$or": []bson.M{{"estado": bson.M{"$ne": "EN_PROCESAMIENTO"}}, {"lease": bson.M{"$lt": time.Now().UTC()}}}}, bson.M{"$set": bson.M{"estado": "EN_PROCESAMIENTO", "lease": time.Now().Add(30 * time.Second).UTC(), "actualizado": time.Now().UTC()}})
	if err != nil {
		return false, err
	}
	if result.ModifiedCount == 0 {
		return false, ErrBusy
	}
	return true, nil
}

/**
 * Persiste resultado y estado final antes de confirmar el mensaje AMQP.
 *
 * @method Complete
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {string} id - Identificador del recurso solicitado.
 * @param {domain.Recommendation} result - Valor de result requerido por la operación.
 * @param {string} traceID - Valor de traceID requerido por la operación.
 * @returns {error} Resultado de la operación; error si no puede completarse.
 */
func (s *Store) Complete(ctx context.Context, id string, result domain.Recommendation, traceID string) error {
	_, err := s.DB.Collection("estudios").UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"estado": result.Status, "resultado": result, "traceId": traceID, "actualizado": time.Now().UTC(), "error": ""}, "$unset": bson.M{"lease": ""}})
	return err
}

/**
 * Conserva el estado de error recuperable y libera el lease del estudio.
 *
 * @method Failed
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {string} id - Identificador del recurso solicitado.
 * @returns {error} Resultado de la operación; error si no puede completarse.
 */
func (s *Store) Failed(ctx context.Context, id string) error {
	_, err := s.DB.Collection("estudios").UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"estado": "ERROR", "error": "Dependencia no disponible; se reintentará o requerirá reproceso.", "actualizado": time.Now().UTC()}, "$unset": bson.M{"lease": ""}})
	return err
}
