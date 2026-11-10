package domain

import (
	"encoding/json"
	"testing"
)

/**
 * Comprueba el comportamiento y los invariantes indicados por RecommendationPhysicalAndRankingInvariants.
 *
 * @method TestRecommendationPhysicalAndRankingInvariants
 * @param {*testing.T} t - Contexto de verificación de la prueba.
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func TestRecommendationPhysicalAndRankingInvariants(t *testing.T) {
	panels := []Panel{{ID: "a", Watts: 450, AreaMilli: 2500, UnitCents: 25000000, Currency: "ARS", Active: true}, {ID: "b", Watts: 550, AreaMilli: 2800, UnitCents: 30000000, Currency: "ARS", Active: true}}
	options := StudyOptions{Sun: "4", Yield: "0.8", Coverage: "1"}
	result, err := Recommend("3", options, panels)
	if err != nil || result.Status != "COMPLETADO" || result.Selected.Panel.ID != "b" || result.Selected.Quantity != 2 || result.Selected.Generation != json.Number("3.520000") {
		t.Fatalf("ranking: %+v %v", result, err)
	}
	area := DecimalNumber("1")
	options.Area = &area
	result, err = Recommend("3", options, panels)
	if err != nil || result.Status != "SIN_ALTERNATIVA" || result.Selected.Quantity != 0 || result.Selected.Feasible {
		t.Fatalf("superficie insuficiente: %+v %v", result, err)
	}
	options.Area = nil
	result, err = Recommend("0", options, panels)
	if err != nil || result.Status != "COMPLETADO" || result.Selected.Quantity != 0 {
		t.Fatalf("cero: %+v %v", result, err)
	}
	result, err = Recommend("3", options, nil)
	if err != nil || result.Status != "SIN_ALTERNATIVA" || result.Selected != nil {
		t.Fatal("sin candidatos")
	}
	options.Sun = "0"
	if _, err = Recommend("3", options, panels); err == nil {
		t.Fatal("aceptó recurso cero")
	}
}

/**
 * Comprueba el comportamiento y los invariantes indicados por RecommendationNonComparablePricesAndInactivePanels.
 *
 * @method TestRecommendationNonComparablePricesAndInactivePanels
 * @param {*testing.T} t - Contexto de verificación de la prueba.
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func TestRecommendationNonComparablePricesAndInactivePanels(t *testing.T) {
	panels := []Panel{{ID: "inactive", Watts: 2000, AreaMilli: 1, Active: false}, {ID: "expensive", Watts: 500, AreaMilli: 1000, UnitCents: 9900, Currency: "ARS", Active: true}, {ID: "unpriced", Watts: 500, AreaMilli: 2000, Active: true}}
	result, err := Recommend("1", StudyOptions{Sun: "4", Yield: "1", Coverage: "1"}, panels)
	if err != nil || len(result.Candidates) != 2 || result.Selected.Panel.ID != "expensive" || result.Criteria != "menor superficie, menor cantidad e id (precios no comparables)" {
		t.Fatalf("%+v %v", result, err)
	}
}

/**
 * Rechaza parámetros fuera de rango y dimensionamientos que exceden la cota aritmética.
 *
 * @method TestSolarInputBounds
 * @param {*testing.T} t - Contexto de verificación.
 * @returns {void} Falla si se admite energía, recurso o cobertura inválidos.
 */
func TestSolarInputBounds(t *testing.T) {
	panel := []Panel{{ID: "p", Watts: 500, AreaMilli: 2000, Active: true}}
	base := StudyOptions{Sun: "4", Yield: "0.8", Coverage: "1"}
	for _, daily := range []json.Number{"-1", "1000000000001", "0.0000001", "1e999"} {
		if _, err := Recommend(daily, base, panel); err == nil {
			t.Fatal("demanda inválida", daily)
		}
	}
	for _, o := range []StudyOptions{{Sun: "25", Yield: "0.8", Coverage: "1"}, {Sun: "4", Yield: "0", Coverage: "1"}, {Sun: "4", Yield: "1.1", Coverage: "1"}, {Sun: "4", Yield: ".8", Coverage: "0"}, {Sun: "4", Yield: ".8", Coverage: "1.1"}} {
		if _, err := Recommend("1", o, panel); err == nil {
			t.Fatal("supuesto inválido", o)
		}
	}
	for _, value := range []DecimalNumber{"-1", "1000000001"} {
		o := base
		o.Area = &value
		if _, err := Recommend("1", o, panel); err == nil {
			t.Fatal("superficie inválida")
		}
	}
	extreme := base
	extreme.Sun = "0.000001"
	extreme.Yield = "0.000001"
	if _, err := Recommend("1000000000000", extreme, panel); err == nil {
		t.Fatal("no acotó cantidad")
	}
}

/**
 * Selecciona máxima cobertura alcanzable cuando la superficie impide cumplir el objetivo.
 *
 * @method TestInfeasibleSurfaceAndInvalidCatalogEntry
 * @param {*testing.T} t - Contexto de verificación.
 * @returns {void} Falla si una ficha inválida impide comparar o se declara cumplimiento falso.
 */
func TestInfeasibleSurfaceAndInvalidCatalogEntry(t *testing.T) {
	area := DecimalNumber("3")
	r, err := Recommend("3", StudyOptions{Sun: "4", Yield: ".8", Coverage: "1", Area: &area}, []Panel{{ID: "bad", Watts: 0, AreaMilli: 2000, Active: true}, {ID: "a", Watts: 450, AreaMilli: 2500, UnitCents: 25000000, Currency: "ARS", Active: true}, {ID: "b", Watts: 550, AreaMilli: 2800, UnitCents: 30000000, Currency: "ARS", Active: true}})
	if err != nil || r.Status != "SIN_ALTERNATIVA" || len(r.Candidates) != 2 || r.Selected.Panel.ID != "b" || r.Selected.Coverage != "0.586667" || r.Selected.Feasible || r.Criteria != "mayor cobertura alcanzable; menor costo de paneles, menor superficie e id" {
		t.Fatalf("%+v %v", r, err)
	}
}

/**
 * Conserva el orden por costo de las opciones viables y resuelve empates por identificador.
 *
 * @method TestFeasiblePriceComparabilityAndDeterministicTie
 * @param {*testing.T} t - Contexto de verificación de las fronteras de selección.
 * @returns {void} Falla si una opción inviable altera el criterio o se aplica un desempate incorrecto.
 */
func TestFeasiblePriceComparabilityAndDeterministicTie(t *testing.T) {
	area := DecimalNumber("10")
	options := StudyOptions{Sun: "4", Yield: "1", Coverage: "1", Area: &area}
	panels := []Panel{
		{ID: "cheap", Watts: 500, AreaMilli: 2000, UnitCents: 100, Currency: "ARS", Active: true},
		{ID: "small", Watts: 500, AreaMilli: 1000, UnitCents: 200, Currency: "ARS", Active: true},
		{ID: "cannot-fit", Watts: 500, AreaMilli: 20000, Active: true},
	}
	r, err := Recommend("1", options, panels)
	if err != nil || r.Selected == nil || r.Selected.Panel.ID != "cheap" || r.Criteria != "menor costo de paneles, menor superficie e id" {
		t.Fatalf("un precio desconocido inviable alteró la comparación: %+v %v", r, err)
	}
	panels = []Panel{
		{ID: "b", Watts: 1000, AreaMilli: 2000, UnitCents: 200, Currency: "ARS", Active: true},
		{ID: "a", Watts: 500, AreaMilli: 1000, UnitCents: 100, Currency: "ARS", Active: true},
	}
	r, err = Recommend("3", options, panels)
	if err != nil || r.Selected == nil || r.Selected.Panel.ID != "a" || r.Selected.Quantity != 2 {
		t.Fatalf("el empate por costo y superficie debe resolverse por id: %+v %v", r, err)
	}
}
