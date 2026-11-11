package application

import (
	"context"
	"renewable.local/solar/internal/domain"
	"testing"
)

/**
 * Controla stock y respuestas incompatibles en las pruebas de selección solar.
 *
 * @method supplierProbe
 * @returns {supplierProbe} Representación tipada usada por este componente.
 */
type supplierProbe struct {
	calls int
	bad   bool
}

/**
 * Solicita una cotización para el SKU exacto seleccionado por el dominio.
 *
 * @method QuoteModel
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {int64} watts - Potencia objetivo de la instalación, expresada en W.
 * @param {string} sku - Identificador del modelo de panel que debe cotizarse sin sustituciones.
 * @returns {domain.Quote, error} Oferta del modelo exacto o error de stock, dependencia o contrato.
 */
func (p *supplierProbe) QuoteModel(ctx context.Context, watts int64, sku string) (domain.Quote, error) {
	p.calls++
	if sku == "a" {
		return domain.Quote{}, domain.ErrStock
	}
	quote := domain.Quote{PanelWatts: 550, Quantity: 2, Currency: "ARS", UnitCents: 30000000}
	if p.bad {
		quote.Quantity = 1
	}
	return quote, nil
}

/**
 * Comprueba el comportamiento y los invariantes indicados por CommercialStockChangesSelectedAlternative.
 *
 * @method TestCommercialStockChangesSelectedAlternative
 * @param {*testing.T} t - Contexto de verificación de la prueba.
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func TestCommercialStockChangesSelectedAlternative(t *testing.T) {
	p := &supplierProbe{}
	panels := []domain.Panel{{ID: "a", Watts: 450, AreaMilli: 2500, UnitCents: 25000000, Currency: "ARS", Active: true}, {ID: "b", Watts: 550, AreaMilli: 2800, UnitCents: 30000000, Currency: "ARS", Active: true}}
	r, err := RunStudy(context.Background(), "2", domain.StudyOptions{Sun: "4", Yield: "0.8", Coverage: "1"}, panels, p)
	if err != nil || r.Selected.Panel.ID != "b" || p.calls != 2 || r.Quote == nil {
		t.Fatalf("%+v %v calls%d", r, err, p.calls)
	}
	p.bad = true
	if _, err = RunStudy(context.Background(), "2", domain.StudyOptions{Sun: "4", Yield: "0.8", Coverage: "1"}, panels, p); err != domain.ErrContract {
		t.Fatal("aceptó oferta incompatible", err)
	}
}

/**
 * Comprueba el comportamiento y los invariantes indicados por ZeroConsumptionSkipsSeller.
 *
 * @method TestZeroConsumptionSkipsSeller
 * @param {*testing.T} t - Contexto de verificación de la prueba.
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func TestZeroConsumptionSkipsSeller(t *testing.T) {
	p := &supplierProbe{}
	_, err := RunStudy(context.Background(), "0", domain.StudyOptions{Sun: "4", Yield: "0.8", Coverage: "1"}, []domain.Panel{{ID: "b", Watts: 550, AreaMilli: 2800, Active: true}}, p)
	if err != nil || p.calls != 0 {
		t.Fatal(err, p.calls)
	}
}

/**
 * Permite confirmar una oferta cuando el catálogo indica precio desconocido.
 *
 * @method TestUnknownReferencePriceUsesCommercialOffer
 * @param {*testing.T} t - Contexto de verificación.
 * @returns {void} Falla si un precio desconocido impide cotizar una alternativa válida.
 */
func TestUnknownReferencePriceUsesCommercialOffer(t *testing.T) {
	p := &supplierProbe{}
	r, err := RunStudy(context.Background(), "2", domain.StudyOptions{Sun: "4", Yield: "0.8", Coverage: "1"}, []domain.Panel{{ID: "b", Watts: 550, AreaMilli: 2800, Currency: "ARS", Active: true}}, p)
	if err != nil || r.Quote == nil || r.Status != "COMPLETADO" {
		t.Fatal(r, err)
	}
}
