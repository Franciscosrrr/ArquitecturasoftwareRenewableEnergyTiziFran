package application

import (
	"context"
	"encoding/json"
	"renewable.local/solar/internal/domain"
	"renewable.local/solar/internal/ports"
)

/**
 * Reúne el consumo por catálogo y los supuestos solares explícitos.
 */
type Request struct {
	Consumption json.RawMessage      `json:"consumo"`
	Sun         domain.DecimalNumber `json:"horasSolPico"`
	Yield       domain.DecimalNumber `json:"factorRendimiento"`
}

/**
 * Expone energía y una cotización sin modificar la capacidad M2M de Clínica.
 */
type Result struct {
	Consumption domain.Consumption `json:"consumo"`
	Watts       int64              `json:"potenciaObjetivoW"`
	Quote       domain.Quote       `json:"cotizacion"`
	Mode        string             `json:"modoProveedor"`
	TraceID     string             `json:"traceId"`
}

/**
 * Coordina los puertos del caso de uso sin depender de HTTP ni persistencia.
 */
type Service struct {
	Estimator ports.Estimator
	Supplier  ports.Supplier
	Mode      string
}

/**
 * Obtiene energía confiable y una oferta con stock suficiente del vendedor.
 *
 * @method Execute
 * @param {context.Context} ctx - Deadline compartido de la operación.
 * @param {Request} input - Consumo y parámetros de dimensionamiento.
 * @param {string} traceID - Identificador de traza para la respuesta.
 * @returns {Result, error} Cotización completa o error sin éxito ficticio.
 */
func (s Service) Execute(ctx context.Context, input Request, traceID string) (Result, error) {
	if len(input.Consumption) == 0 || !json.Valid(input.Consumption) {
		return Result{}, domain.ErrValidation
	}
	if _, err := domain.TargetWatts("1", json.Number(input.Sun), json.Number(input.Yield)); err != nil {
		return Result{}, err
	}
	consumption, err := s.Estimator.Estimate(ctx, input.Consumption)
	if err != nil {
		return Result{}, err
	}
	watts, err := domain.TargetWatts(consumption.Daily, json.Number(input.Sun), json.Number(input.Yield))
	if err != nil {
		return Result{}, err
	}
	quote, err := s.Supplier.Quote(ctx, watts)
	if err != nil {
		return Result{}, err
	}
	if err = quote.Validate(watts); err != nil {
		return Result{}, err
	}
	return Result{consumption, watts, quote, s.Mode, traceID}, nil
}
