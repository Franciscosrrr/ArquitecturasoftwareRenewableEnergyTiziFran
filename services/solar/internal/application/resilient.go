package application

import (
	"context"
	"encoding/json"
	"errors"
	"renewable.local/platform"
	"renewable.local/solar/internal/domain"
	"strconv"
)

/**
 * Proporciona el catálogo técnico propio al recorrido anónimo y a los estudios.
 */
type PanelCatalog interface {
	/**
	 * Recupera las fichas técnicas activas o el catálogo completo.
	 *
	 * @method Panels
	 * @param {context.Context} ctx - Presupuesto de lectura.
	 * @param {bool} all - Incluye fichas inactivas.
	 * @returns {[]domain.Panel, error} Fichas canónicas o error de almacenamiento.
	 */
	Panels(context.Context, bool) ([]domain.Panel, error)
}

/**
 * Prefiere el proveedor real y repite el estudio entero en simulación ante un fallo técnico.
 */
type ResilientSupplier struct {
	Primary  ModelSupplier
	Fallback ModelSupplier
	Catalog  PanelCatalog
	External bool
}

/**
 * Expone el puerto de modelo; la conmutación coherente se aplica al estudio completo.
 *
 * @method QuoteModel
 * @param {context.Context} ctx - Presupuesto de operación.
 * @param {int64} watts - Potencia objetivo.
 * @param {string} sku - Ficha técnica seleccionada.
 * @returns {domain.Quote, error} Oferta del adaptador principal.
 */
func (s *ResilientSupplier) QuoteModel(ctx context.Context, watts int64, sku string) (domain.Quote, error) {
	return s.Primary.QuoteModel(ctx, watts, sku)
}

/**
 * Evita simular ante falta de stock o datos inválidos y respeta cancelaciones del usuario.
 *
 * @method canSimulate
 * @param {context.Context} ctx - Presupuesto todavía disponible.
 * @param {error} err - Fallo del proveedor.
 * @returns {bool} Verdadero para indisponibilidad o contrato incompatible.
 */
func canSimulate(ctx context.Context, err error) bool {
	return ctx.Err() == nil && (errors.Is(err, domain.ErrUnavailable) || errors.Is(err, domain.ErrContract))
}

/**
 * Calcula y cotiza alternativas sin mezclar precios reales con precios simulados.
 *
 * @method PlanStudy
 * @param {context.Context} ctx - Presupuesto compartido.
 * @param {json.Number} daily - Demanda diaria estimada.
 * @param {domain.StudyOptions} options - Supuestos solares explícitos.
 * @param {[]domain.Panel} panels - Fichas técnicas canónicas.
 * @returns {domain.Recommendation, error} Estudio con procedencia persistible.
 */
func (s *ResilientSupplier) PlanStudy(ctx context.Context, daily json.Number, options domain.StudyOptions, panels []domain.Panel) (domain.Recommendation, error) {
	result, err := runStudy(ctx, daily, options, panels, s.Primary)
	if !s.External || !canSimulate(ctx, err) {
		return result, err
	}
	platform.Operations.WithLabelValues("proveedor_paneles", "respaldo_simulado").Inc()
	result, err = runStudy(ctx, daily, options, panels, s.Fallback)
	if err == nil {
		result.Note += " Proveedor real no disponible: paneles, precios y stock de respaldo simulados."
		if result.Quote != nil {
			result.Quote.Source = "simulado"
			result.Quote.FallbackReason = "PROVEEDOR_NO_DISPONIBLE"
		}
	}
	return result, err
}

/**
 * Cotiza el recorrido invitado con las mismas fichas, selección y respaldo que los estudios.
 *
 * @method Quote
 * @param {context.Context} ctx - Presupuesto del formulario anónimo.
 * @param {int64} watts - Potencia objetivo ya validada.
 * @returns {domain.Quote, error} Oferta elegida o ausencia de alternativa con stock.
 */
func (s *ResilientSupplier) Quote(ctx context.Context, watts int64) (domain.Quote, error) {
	if watts < 1 || watts > 10000000 {
		return domain.Quote{}, domain.ErrValidation
	}
	panels, err := s.Catalog.Panels(ctx, false)
	if err != nil {
		return domain.Quote{}, domain.ErrUnavailable
	}
	daily := json.Number(strconv.FormatInt(watts, 10) + "e-3")
	result, err := s.PlanStudy(ctx, daily, domain.StudyOptions{Sun: "1", Yield: "1", Coverage: "1"}, panels)
	if err != nil {
		return domain.Quote{}, err
	}
	if result.Quote == nil {
		return domain.Quote{}, domain.ErrStock
	}
	return *result.Quote, nil
}

/**
 * Enriquece el catálogo con ofertas actuales; un fallo conmuta toda la lista a simulación.
 *
 * @method PricePanels
 * @param {context.Context} ctx - Presupuesto de lectura comercial.
 * @param {[]domain.Panel} panels - Fichas que coinciden con los filtros técnicos.
 * @returns {[]domain.Panel, error} Precios y etiquetas coherentes para filtrar y mostrar.
 */
func (s *ResilientSupplier) PricePanels(ctx context.Context, panels []domain.Panel) ([]domain.Panel, error) {
	result, err := pricePanels(ctx, panels, s.Primary, s.External)
	if s.External && canSimulate(ctx, err) {
		result, err = pricePanels(ctx, panels, s.Fallback, false)
		for i := range result {
			if result[i].Offer != nil {
				result[i].Offer.FallbackReason = "PROVEEDOR_NO_DISPONIBLE"
			}
		}
		return result, err
	}
	return result, err
}

/**
 * Anota precio unitario, disponibilidad y procedencia sin alterar el catálogo persistido.
 *
 * @method pricePanels
 * @param {context.Context} ctx - Presupuesto cancelable.
 * @param {[]domain.Panel} panels - Fichas técnicas a enriquecer.
 * @param {ModelSupplier} supplier - Proveedor real o simulador.
 * @param {bool} external - Identifica una consulta al grupo proveedor.
 * @returns {[]domain.Panel, error} Vista comercial o fallo que activa el respaldo.
 */
func pricePanels(ctx context.Context, panels []domain.Panel, supplier ModelSupplier, external bool) ([]domain.Panel, error) {
	result := append([]domain.Panel{}, panels...)
	for i := range result {
		p := &result[i]
		q, err := supplier.QuoteModel(ctx, p.Watts, p.ID)
		p.UnitCents = 0
		p.Availability = "SIN_OFERTA"
		p.Offer = nil
		p.OfferSource = "simulado"
		if external {
			p.OfferSource = "externo"
		}
		if errors.Is(err, domain.ErrStock) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if q.Source == "" {
			q.Source = p.OfferSource
		}
		p.Offer = &q
		p.UnitCents = q.UnitCents
		p.Currency = q.Currency
		p.Availability = "DISPONIBLE"
	}
	return result, nil
}
