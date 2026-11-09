package private

import (
	"context"
	"database/sql"
	"errors"
)

/**
 * Cambia solo el estado de archivo, sin alterar datos ni versión del contenido.
 *
 * @method SetArchived
 * @param {context.Context} ctx - Presupuesto de la operación.
 * @param {string} owner - Propietario autenticado.
 * @param {string} id - Recurso propio solicitado.
 * @param {int64} version - Versión de contenido esperada.
 * @param {bool} place - Selecciona lugar o configuración, sin aceptar nombres SQL externos.
 * @param {bool} archived - Estado de organización deseado.
 * @returns {error} Ausencia, conflicto o error de persistencia; nil al confirmar.
 */
func (s *Store) SetArchived(ctx context.Context, owner, id string, version int64, place, archived bool) error {
	table, column := "configuraciones", "archivada"
	if place {
		table, column = "lugares", "archivado"
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var current int64
	err = tx.QueryRowContext(ctx, "SELECT version FROM "+table+" WHERE id=? AND owner_id=? AND eliminado_at IS NULL FOR UPDATE", id, owner).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if current != version {
		return ErrConflict
	}
	if _, err = tx.ExecContext(ctx, "UPDATE "+table+" SET "+column+"=? WHERE id=? AND owner_id=?", archived, id, owner); err != nil {
		return err
	}
	return tx.Commit()
}

/**
 * Quita un recurso propio de las listas conservando evaluaciones y snapshots históricos.
 *
 * @method Delete
 * @param {context.Context} ctx - Presupuesto de la operación.
 * @param {string} owner - Propietario autenticado.
 * @param {string} id - Recurso a eliminar lógicamente.
 * @param {int64} version - Versión esperada para evitar eliminar una edición ajena al intento.
 * @param {bool} place - Selecciona la tabla fija de lugares o configuraciones.
 * @returns {error} Conflicto si el lugar conserva consumos; nil tras confirmar la eliminación.
 */
func (s *Store) Delete(ctx context.Context, owner, id string, version int64, place bool) error {
	table := "configuraciones"
	if place {
		table = "lugares"
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var current int64
	err = tx.QueryRowContext(ctx, "SELECT version FROM "+table+" WHERE id=? AND owner_id=? AND eliminado_at IS NULL FOR UPDATE", id, owner).Scan(&current)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if current != version {
		return ErrConflict
	}
	if place {
		var count int
		if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM configuraciones WHERE lugar_id=? AND owner_id=? AND eliminado_at IS NULL", id, owner).Scan(&count); err != nil {
			return err
		}
		if count > 0 {
			return ErrPlaceInUse
		}
	}
	if _, err = tx.ExecContext(ctx, "UPDATE "+table+" SET eliminado_at=CURRENT_TIMESTAMP WHERE id=? AND owner_id=?", id, owner); err != nil {
		return err
	}
	return tx.Commit()
}
