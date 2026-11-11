package domain

import (
	"encoding/json"
	"errors"
	"testing"
)

/**
 * Comprueba límites y redondeo hacia arriba sin errores de coma flotante.
 *
 * @method TestTargetWatts
 * @param {*testing.T} t - Contexto de prueba.
 * @returns {void} Falla si el dimensionamiento o la validación cambian.
 */
func TestTargetWatts(t *testing.T) {
	for _, tc := range []struct {
		daily, sun, yield string
		want              int64
		bad               bool
	}{
		{"1.58", "4", "0.8", 494, false},
		{"1.44", "4", "0.8", 450, false},
		{"0.000001", "8", "1", 1, false},
		{"0", "4", "0.8", 0, true},
		{"1", "0.49", "0.8", 0, true},
		{"1", "4", "1.01", 0, true},
		{"NaN", "4", "0.8", 0, true},
		{"100000", "0.5", "0.1", 0, true},
	} {
		got, err := TargetWatts(json.Number(tc.daily), json.Number(tc.sun), json.Number(tc.yield))
		if got != tc.want || (err != nil) != tc.bad {
			t.Fatalf("%+v: got %d, %v", tc, got, err)
		}
	}
}

/**
 * Rechaza cotizaciones cuyo importe, moneda, cobertura o stock no son válidos.
 *
 * @method TestQuoteInvariants
 * @param {*testing.T} t - Contexto de prueba.
 * @returns {void} Falla si se acepta una oferta comercial inconsistente.
 */
func TestQuoteInvariants(t *testing.T) {
	valid := Quote{ID: "q1", SKU: "panel", PanelWatts: 450, Quantity: 2, Stock: 10, UnitCents: 25000000, TotalCents: 50000000, Currency: "ARS"}
	if err := valid.Validate(494); err != nil {
		t.Fatal(err)
	}
	bad := valid
	bad.Stock = 1
	if !errors.Is(bad.Validate(494), ErrStock) {
		t.Fatal("stock")
	}
	bad = valid
	bad.TotalCents++
	if !errors.Is(bad.Validate(494), ErrContract) {
		t.Fatal("importe")
	}
	bad = valid
	bad.Currency = "USD"
	if !errors.Is(bad.Validate(494), ErrContract) {
		t.Fatal("moneda")
	}
	bad = valid
	bad.Quantity = 1
	bad.TotalCents = bad.UnitCents
	if !errors.Is(bad.Validate(494), ErrContract) {
		t.Fatal("cobertura")
	}
}
