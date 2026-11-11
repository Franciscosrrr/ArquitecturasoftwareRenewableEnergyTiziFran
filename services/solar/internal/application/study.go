package application

import (
	"context"
	"encoding/json"
	"errors"
	"renewable.local/solar/internal/domain"
)

/**
 * Cotiza un modelo técnico exacto mediante un adaptador comercial.
 */
type ModelSupplier interface {
	/**
	 * Consulta precio y stock para la cantidad derivada de una ficha propia.
	 *
	 * @method QuoteModel
	 * @param {context.Context} ctx - Presupuesto de la consulta.
	 * @param {int64} watts - Potencia objetivo.
	 * @param {string} sku - Identidad de la ficha local.
	 * @returns {domain.Quote, error} Oferta validada o error controlado.
	 */
	QuoteModel(context.Context, int64, string) (domain.Quote, error)
}

/**
 * Selecciona una alternativa factible cuya oferta comercial confirma modelo y stock.
 *
 * @method RunStudy
 * @param {context.Context} ctx - Presupuesto de ejecución del trabajo.
 * @param {json.Number} daily - Demanda diaria congelada.
 * @param {domain.StudyOptions} options - Recurso manual, rendimiento, cobertura y superficie.
 * @param {[]domain.Panel} panels - Fichas activas y versionadas.
 * @param {ModelSupplier} supplier - Puerto comercial del vendedor.
 * @returns {domain.Recommendation, error} Comparación y oferta confirmada o falla recuperable.
 */
func RunStudy(ctx context.Context, daily json.Number, options domain.StudyOptions, panels []domain.Panel, supplier ModelSupplier) (domain.Recommendation, error) {
	if planner, ok := supplier.(interface {
		PlanStudy(context.Context, json.Number, domain.StudyOptions, []domain.Panel) (domain.Recommendation, error)
	}); ok {
		return planner.PlanStudy(ctx, daily, options, panels)
	}
	return runStudy(ctx, daily, options, panels, supplier)
}

/**
 * Cotiza cada alternativa viable y compara el costo comercial completo de las ofertas.
 *
 * @method runStudy
 * @param {context.Context} ctx - Presupuesto cancelable del estudio.
 * @param {json.Number} daily - Demanda diaria congelada.
 * @param {domain.StudyOptions} options - Supuestos de dimensionamiento.
 * @param {[]domain.Panel} panels - Fichas técnicas propias.
 * @param {ModelSupplier} supplier - Fuente comercial única para todo el estudio.
 * @returns {domain.Recommendation, error} Selección por costo cotizado o ausencia de stock.
 */
func runStudy(ctx context.Context, daily json.Number, options domain.StudyOptions, panels []domain.Panel, supplier ModelSupplier) (domain.Recommendation, error) {
	result, err := domain.Recommend(daily, options, panels)
	if err != nil {
		return result, err
	}
	if result.Selected == nil || result.Selected.Quantity == 0 || result.Status != "COMPLETADO" {
		return result, nil
	}
	result.Status = "SIN_ALTERNATIVA"
	result.Selected = nil
	for _, candidate := range result.Candidates {
		if !candidate.Feasible || candidate.Quantity == 0 {
			continue
		}
		quote, err := supplier.QuoteModel(ctx, candidate.Watts, candidate.Panel.ID)
		if errors.Is(err, domain.ErrStock) {
			continue
		}
		if err != nil {
			return result, err
		}
		if quote.PanelWatts != candidate.Panel.Watts || quote.Quantity != candidate.Quantity || quote.Currency != candidate.Panel.Currency || (quote.Source != "externo" && candidate.Panel.UnitCents > 0 && quote.UnitCents != candidate.Panel.UnitCents) {
			return result, domain.ErrContract
		}
		if quote.Source == "" {
			quote.Source = "simulado"
		}
		if result.Quote == nil || quote.TotalCents < result.Quote.TotalCents {
			chosen := candidate
			chosen.TotalCents = quote.TotalCents
			result.Selected = &chosen
			result.Quote = &quote
			result.Status = "COMPLETADO"
		}
	}
	if result.Quote != nil && result.Quote.Source == "externo" {
		result.Criteria = "menor costo total cotizado con stock (incluye flete y descuento aplicado); empate por orden técnico"
		result.Note += " Precio y stock consultados al Comparador de Proveedores; son una foto del momento, sin reserva de precio ni inventario. Las superficies y potencias provienen de las fichas locales de referencia, no del proveedor comercial."
	}
	if result.Status == "SIN_ALTERNATIVA" {
		result.Note += " No hay oferta del vendedor con stock suficiente para cumplir el objetivo."
	}
	return result, nil
}
