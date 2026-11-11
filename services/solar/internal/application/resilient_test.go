package application

import (
	"context"
	"renewable.local/solar/internal/domain"
	"testing"
)

/**
 * Controla fallos y precios independientes para probar la selección y el respaldo.
 */
type commercialProbe struct {
	calls      int
	failure    error
	prices     map[string]int64
	external   bool
	failSecond bool
}

/**
 * Responde una oferta comercial controlada para un modelo exacto.
 *
 * @method QuoteModel
 * @param {context.Context} ctx - Presupuesto del estudio.
 * @param {int64} watts - Potencia solicitada.
 * @param {string} sku - Modelo técnico elegido.
 * @returns {domain.Quote, error} Oferta de prueba o fallo programado.
 */
func (s *commercialProbe) QuoteModel(ctx context.Context, watts int64, sku string) (domain.Quote, error) {
	s.calls++
	if s.failure != nil || s.failSecond && s.calls%2 == 0 {
		if s.failure != nil {
			return domain.Quote{}, s.failure
		}
		return domain.Quote{}, domain.ErrUnavailable
	}
	q := domain.Quote{ID: "sim-test", SKU: sku, PanelWatts: 500, Quantity: watts / 500, Stock: 100, UnitCents: 100, TotalCents: s.prices[sku], Currency: "ARS"}
	if s.external {
		q.Source = "externo"
		q.ID = ""
		q.UnitCents = 900
	}
	return q, nil
}

/**
 * Comprueba la elección por importe real completo aunque difiera del catálogo de referencia.
 *
 * @method TestRealOfferDeterminesSelection
 * @param {*testing.T} t - Prueba de la consecuencia comercial de la integración.
 * @returns {void} Falla si se impone un precio didáctico sobre el precio real.
 */
func TestRealOfferDeterminesSelection(t *testing.T) {
	panels := []domain.Panel{{ID: "a", Watts: 500, AreaMilli: 1000, UnitCents: 100, Currency: "ARS", Active: true}, {ID: "b", Watts: 500, AreaMilli: 1100, UnitCents: 100, Currency: "ARS", Active: true}}
	real := &commercialProbe{external: true, prices: map[string]int64{"a": 9999, "b": 5000}}
	result, err := RunStudy(context.Background(), "2", domain.StudyOptions{Sun: "4", Yield: "1", Coverage: "1"}, panels, real)
	if err != nil || result.Selected == nil || result.Selected.Panel.ID != "b" || result.Selected.TotalCents != 5000 || real.calls != 2 {
		t.Fatalf("%+v %v", result, err)
	}
}

/**
 * Evita mezclar una primera oferta real exitosa con un segundo modelo simulado.
 *
 * @method TestWholeStudyFallbackAndRecovery
 * @param {*testing.T} t - Prueba de conmutación consistente y vuelta al proveedor real.
 * @returns {void} Falla si se oculta el respaldo o se mezclan procedencias.
 */
func TestWholeStudyFallbackAndRecovery(t *testing.T) {
	panels := []domain.Panel{{ID: "a", Watts: 500, AreaMilli: 1000, UnitCents: 100, Currency: "ARS", Active: true}, {ID: "b", Watts: 500, AreaMilli: 1100, UnitCents: 100, Currency: "ARS", Active: true}}
	primary := &commercialProbe{external: true, failSecond: true, prices: map[string]int64{"a": 1, "b": 2}}
	fallback := &commercialProbe{prices: map[string]int64{"a": 200, "b": 300}}
	s := &ResilientSupplier{Primary: primary, Fallback: fallback, External: true}
	result, err := RunStudy(context.Background(), "2", domain.StudyOptions{Sun: "4", Yield: "1", Coverage: "1"}, panels, s)
	if err != nil || result.Quote == nil || result.Quote.Source != "simulado" || result.Quote.FallbackReason == "" || result.Quote.TotalCents != 200 || fallback.calls != 2 {
		t.Fatalf("%+v %v", result, err)
	}
	primary.failSecond = false
	result, err = RunStudy(context.Background(), "2", domain.StudyOptions{Sun: "4", Yield: "1", Coverage: "1"}, panels, s)
	if err != nil || result.Quote.Source != "externo" || fallback.calls != 2 {
		t.Fatal("no volvió al proveedor", result, err)
	}
}

/**
 * Conserva una respuesta válida sin stock sin fabricar disponibilidad simulada.
 *
 * @method TestNoStockDoesNotActivateSimulation
 * @param {*testing.T} t - Prueba de distinción entre indisponibilidad técnica y negocio.
 * @returns {void} Falla si el respaldo contradice la falta de oferta informada por el proveedor.
 */
func TestNoStockDoesNotActivateSimulation(t *testing.T) {
	primary := &commercialProbe{failure: domain.ErrStock}
	fallback := &commercialProbe{}
	s := &ResilientSupplier{Primary: primary, Fallback: fallback, External: true}
	panels := []domain.Panel{{ID: "a", Watts: 500, AreaMilli: 1000, Currency: "ARS", Active: true}}
	result, err := RunStudy(context.Background(), "2", domain.StudyOptions{Sun: "4", Yield: "1", Coverage: "1"}, panels, s)
	if err != nil || result.Status != "SIN_ALTERNATIVA" || result.Selected != nil || result.Quote != nil || fallback.calls != 0 {
		t.Fatal(result, err)
	}
}

/**
 * Etiqueta todas las tarjetas simuladas cuando falla una consulta del catálogo comercial.
 *
 * @method TestCatalogFallbackAndNoStock
 * @param {*testing.T} t - Prueba del origen mostrado en la búsqueda.
 * @returns {void} Falla si quedan precios reales mezclados o stock inventado.
 */
func TestCatalogFallbackAndNoStock(t *testing.T) {
	panels := []domain.Panel{{ID: "a", Watts: 500, UnitCents: 100}, {ID: "b", Watts: 500, UnitCents: 100}}
	primary := &commercialProbe{external: true, failSecond: true, prices: map[string]int64{"a": 1, "b": 2}}
	fallback := &commercialProbe{prices: map[string]int64{"a": 100, "b": 100}}
	s := &ResilientSupplier{Primary: primary, Fallback: fallback, External: true}
	priced, err := s.PricePanels(context.Background(), panels)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range priced {
		if p.OfferSource != "simulado" || p.Offer == nil || p.UnitCents != 100 {
			t.Fatal(p)
		}
	}
	primary.failSecond = false
	primary.failure = domain.ErrStock
	priced, err = s.PricePanels(context.Background(), panels)
	if err != nil || fallback.calls != 2 {
		t.Fatal(err)
	}
	for _, p := range priced {
		if p.Availability != "SIN_OFERTA" || p.OfferSource != "externo" || p.UnitCents != 0 {
			t.Fatal(p)
		}
	}
}
