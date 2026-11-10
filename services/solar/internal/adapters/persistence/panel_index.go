package persistence

import (
	"context"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"renewable.local/solar/internal/domain"
	"time"
)

/**
 * Reclama una ficha pendiente con lease compartido entre las dos instancias Solar.
 *
 * @method ClaimPanelIndex
 * @param {context.Context} ctx - Presupuesto del sincronizador.
 * @param {string} claim - Identificador del intento de indexación.
 * @returns {domain.Panel, error} Ficha canónica; ErrNoDocuments cuando no hay trabajo.
 */
func (s *Store) ClaimPanelIndex(ctx context.Context, claim string) (domain.Panel, error) {
	var p domain.Panel
	err := s.DB.Collection("paneles").FindOneAndUpdate(ctx, bson.M{"busquedaPendiente": true, "$or": []bson.M{{"busquedaLeaseHasta": bson.M{"$exists": false}}, {"busquedaLeaseHasta": bson.M{"$lt": time.Now().UTC()}}}}, bson.M{"$set": bson.M{"busquedaIntento": claim, "busquedaLeaseHasta": time.Now().Add(30 * time.Second).UTC()}}, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&p)
	return p, err
}

/**
 * Confirma únicamente la versión indexada y libera el lease sin perder ediciones nuevas.
 *
 * @method FinishPanelIndex
 * @param {context.Context} ctx - Presupuesto de persistencia.
 * @param {domain.Panel} panel - Versión enviada al motor.
 * @param {string} claim - Intento que posee el lease.
 * @param {bool} success - La tarea de Meilisearch confirmó éxito.
 * @returns {error} Error de persistencia; las fichas no confirmadas continúan pendientes.
 */
func (s *Store) FinishPanelIndex(ctx context.Context, panel domain.Panel, claim string, success bool) error {
	collection := s.DB.Collection("paneles")
	if success {
		_, err := collection.UpdateOne(ctx, bson.M{"_id": panel.ID, "version": panel.Version, "busquedaIntento": claim}, bson.M{"$set": bson.M{"busquedaPendiente": false}})
		if err != nil {
			return err
		}
	}
	_, err := collection.UpdateOne(ctx, bson.M{"_id": panel.ID, "busquedaIntento": claim}, bson.M{"$unset": bson.M{"busquedaIntento": "", "busquedaLeaseHasta": ""}})
	return err
}

/**
 * Solicita reconstruir el índice derivado conservando intactas las fichas canónicas.
 *
 * @method ReindexPanels
 * @param {context.Context} ctx - Presupuesto de la operación administrativa.
 * @returns {error} Error Mongo; los workers procesan las fichas marcadas en segundo plano.
 */
func (s *Store) ReindexPanels(ctx context.Context) error {
	_, err := s.DB.Collection("paneles").UpdateMany(ctx, bson.M{}, bson.M{"$set": bson.M{"busquedaPendiente": true}})
	return err
}
