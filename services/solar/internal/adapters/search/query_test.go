package search

import (
	"net/url"
	"renewable.local/solar/internal/domain"
	"testing"
)

/**
 * Rechaza filtros ambiguos, monedas incomparables y valores fuera de las cotas públicas.
 *
 * @method TestQueryBounds
 * @param {*testing.T} t - Contexto de las pruebas de frontera.
 * @returns {void} Falla ante un filtro inválido aceptado.
 */
func TestQueryBounds(t *testing.T) {
	for _, raw := range []string{"potenciaMin=600&potenciaMax=500", "potenciaMin=NaN", "potenciaMax=2001", "pagina=0", "tamano=51", "precioMax=1.001&moneda=ARS", "precioMax=2", "orden=precio", "moneda=EUR", "q=a&q=b", "desconocido=1", "superficieMax=0.0001", "orden=otro"} {
		values, err := url.ParseQuery(raw)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Parse(values); err == nil {
			t.Fatal("aceptó", raw)
		}
	}
	q, err := Parse(url.Values{"moneda": {"ARS"}, "precioMax": {"123.45"}, "superficieMax": {"1.8"}, "potenciaMin": {"500"}})
	if err != nil || *q.MaxCents != 12345 || *q.MaxAreaMilli != 1800 || *q.MinWatts != 500 {
		t.Fatalf("%+v %v", q, err)
	}
}

/**
 * Conserva actividad, moneda, cotas y paginación en el respaldo, sin tratar precio desconocido como gratis.
 *
 * @method TestFallbackFiltersAndUnknownPrice
 * @param {*testing.T} t - Contexto de comparación de resultados.
 * @returns {void} Falla si el respaldo mezcla fichas incompatibles o expone resultados inactivos.
 */
func TestFallbackFiltersAndUnknownPrice(t *testing.T) {
	panels := []domain.Panel{{ID: "a", Name: "Panel estándar", Watts: 520, AreaMilli: 1800, UnitCents: 12345, Currency: "ARS", Active: true}, {ID: "b", Name: "Panel estándar", Watts: 550, AreaMilli: 2500, UnitCents: 18000, Currency: "ARS", Active: true}, {ID: "c", Name: "Panel estándar", Watts: 520, AreaMilli: 1800, UnitCents: 100, Currency: "USD", Active: true}, {ID: "d", Name: "Panel estándar", Watts: 520, AreaMilli: 1800, Currency: "ARS", Active: true}, {ID: "e", Name: "Panel estándar", Watts: 520, AreaMilli: 1800, UnitCents: 100, Currency: "ARS", Active: false}}
	q, err := Parse(url.Values{"q": {"panel estandar"}, "moneda": {"ARS"}, "precioMax": {"150"}, "superficieMax": {"2"}, "potenciaMin": {"500"}})
	if err != nil {
		t.Fatal(err)
	}
	p := Fallback(q, panels)
	if p.Total != 1 || len(p.Items) != 1 || p.Items[0].ID != "a" {
		t.Fatalf("%+v", p)
	}
	q, err = Parse(url.Values{"moneda": {"ARS"}, "orden": {"precio"}, "tamano": {"1"}, "pagina": {"2"}})
	if err != nil {
		t.Fatal(err)
	}
	p = Fallback(q, panels)
	if p.Total != 2 || len(p.Items) != 1 || p.Items[0].ID != "b" {
		t.Fatalf("%+v", p)
	}
	q.Page = 3
	if len(Fallback(q, panels).Items) != 0 {
		t.Fatal("página fuera de rango")
	}
}
