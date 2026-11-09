package application

import (
	"context"
	"encoding/json"
	"errors"
	"renewable.local/solar/internal/domain"
)

type ModelSupplier interface {
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
	result, err := domain.Recommend(daily, options, panels)
	if err != nil {
		return result, err
	}
	if result.Selected == nil || result.Selected.Quantity == 0 || result.Status != "COMPLETADO" {
		return result, nil
	}
	result.Status = "SIN_ALTERNATIVA"
	for _, candidate := range result.Candidates {
		if !candidate.Feasible {
			continue
		}
		quote, err := supplier.QuoteModel(ctx, candidate.Watts, candidate.Panel.ID)
		if errors.Is(err, domain.ErrStock) {
			continue
		}
		if err != nil {
			return result, err
		}
		if quote.PanelWatts != candidate.Panel.Watts || quote.Quantity != candidate.Quantity || quote.Currency != candidate.Panel.Currency || (candidate.Panel.UnitCents > 0 && quote.UnitCents != candidate.Panel.UnitCents) {
			return result, domain.ErrContract
		}
		chosen := candidate
		result.Selected = &chosen
		result.Quote = &quote
		result.Status = "COMPLETADO"
		break
	}
	if result.Status == "SIN_ALTERNATIVA" {
		result.Note += " No hay oferta del vendedor con stock suficiente para cumplir el objetivo."
	}
	return result, nil
}
