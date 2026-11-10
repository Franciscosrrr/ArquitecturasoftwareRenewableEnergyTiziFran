package ports

import (
	"context"
	"encoding/json"
	"renewable.local/solar/internal/domain"
)

/**
 * Define la estimación interna sin acoplar Solar a tipos de otro microservicio.
 */
type Estimator interface {
	/**
	 * Solicita a Consumo un cálculo con fichas del catálogo.
	 *
	 * @method Estimate
	 * @param {context.Context} ctx - Contexto con deadline y correlación.
	 * @param {json.RawMessage} raw - Solicitud del contrato de catálogo.
	 * @returns {domain.Consumption, error} Energía calculada o error.
	 */
	Estimate(ctx context.Context, raw json.RawMessage) (domain.Consumption, error)
}

/**
 * Define la capacidad comercial consumida desde Solar.
 */
type Supplier interface {
	/**
	 * Cotiza paneles suficientes para la potencia objetivo.
	 *
	 * @method Quote
	 * @param {context.Context} ctx - Contexto de la operación.
	 * @param {int64} watts - Potencia objetivo en W.
	 * @returns {domain.Quote, error} Cotización validada o error.
	 */
	Quote(ctx context.Context, watts int64) (domain.Quote, error)
}
