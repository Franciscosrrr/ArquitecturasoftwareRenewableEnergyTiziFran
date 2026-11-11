package search

import (
	"errors"
	"math"
	"net/url"
	"renewable.local/solar/internal/domain"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

/**
 * Expresa texto y límites de potencia, área y precio para buscar paneles.
 *
 * @method Query
 * @returns {Query} Representación tipada usada por este componente.
 */
type Query struct {
	Text, Currency, Order                      string
	Page, Size                                 int
	MinWatts, MaxWatts, MaxAreaMilli, MaxCents *int64
}

/**
 * Devuelve las fichas solares visibles y el total de coincidencias de la consulta.
 *
 * @method Page
 * @returns {Page} Representación tipada usada por este componente.
 */
type Page struct {
	Items []domain.Panel `json:"items"`
	Total int            `json:"total"`
	Page  int            `json:"pagina"`
	Size  int            `json:"tamano"`
}

var ErrQuery = errors.New("Revisá los filtros: rangos, moneda y página deben ser válidos.")

/**
 * Valida texto, rangos y orden antes de construir filtros del motor.
 *
 * @method Parse
 * @param {url.Values} values - Parámetros de búsqueda públicos.
 * @returns {Query, error} Consulta acotada o error de validación.
 */
func Parse(values url.Values) (Query, error) {
	q := Query{Text: strings.TrimSpace(values.Get("q")), Currency: values.Get("moneda"), Order: values.Get("orden"), Page: 1, Size: 6}
	allowed := map[string]bool{"q": true, "moneda": true, "orden": true, "pagina": true, "tamano": true, "potenciaMin": true, "potenciaMax": true, "superficieMax": true, "precioMax": true}
	for key, v := range values {
		if !allowed[key] || len(v) != 1 {
			return q, ErrQuery
		}
	}
	if utf8.RuneCountInString(q.Text) > 100 || (q.Currency != "" && q.Currency != "ARS" && q.Currency != "USD") {
		return q, ErrQuery
	}
	if q.Order == "" {
		q.Order = "nombre"
	}
	if q.Order != "nombre" && q.Order != "potencia" && q.Order != "-potencia" && q.Order != "precio" {
		return q, ErrQuery
	}
	for _, field := range []struct {
		key    string
		target *int
		max    int
	}{{"pagina", &q.Page, 100000}, {"tamano", &q.Size, 50}} {
		if raw := values.Get(field.key); raw != "" {
			v, e := strconv.Atoi(raw)
			if e != nil || v < 1 || v > field.max {
				return q, ErrQuery
			}
			*field.target = v
		}
	}
	for _, field := range []struct {
		key        string
		target     **int64
		scale, max float64
	}{{"potenciaMin", &q.MinWatts, 1, 2000}, {"potenciaMax", &q.MaxWatts, 1, 2000}, {"superficieMax", &q.MaxAreaMilli, 1000, 100}, {"precioMax", &q.MaxCents, 100, 10000000}} {
		if raw := values.Get(field.key); raw != "" {
			v, e := strconv.ParseFloat(raw, 64)
			scaled := v * field.scale
			if e != nil || math.IsNaN(v) || math.IsInf(v, 0) || v < 0 || v > field.max || math.Abs(scaled-math.Round(scaled)) > .000001 {
				return q, ErrQuery
			}
			n := int64(math.Round(scaled))
			*field.target = &n
		}
	}
	if q.MinWatts != nil && q.MaxWatts != nil && *q.MinWatts > *q.MaxWatts {
		return q, ErrQuery
	}
	if (q.MaxCents != nil || q.Order == "precio") && q.Currency == "" {
		return q, ErrQuery
	}
	return q, nil
}

/**
 * Normaliza texto de la búsqueda de respaldo sin interpretar expresiones regulares.
 *
 * @method normalize
 * @param {string} value - Nombre, fuente o texto consultado.
 * @returns {string} Texto minúsculo sin acentos vocálicos comunes.
 */
func normalize(value string) string {
	return strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u").Replace(strings.ToLower(value))
}

/**
 * Filtra y pagina fichas canónicas cuando el índice derivado no está disponible.
 *
 * @method Fallback
 * @param {Query} q - Consulta validada.
 * @param {[]domain.Panel} panels - Fichas recuperadas desde Mongo.
 * @returns {Page} Resultados activos con orden determinista.
 */
func Fallback(q Query, panels []domain.Panel) Page {
	items := []domain.Panel{}
	for _, p := range panels {
		if q.Order == "precio" && p.UnitCents == 0 {
			continue
		}
		if !p.Active || (q.Currency != "" && p.Currency != q.Currency) || (q.MinWatts != nil && p.Watts < *q.MinWatts) || (q.MaxWatts != nil && p.Watts > *q.MaxWatts) || (q.MaxAreaMilli != nil && p.AreaMilli > *q.MaxAreaMilli) || (q.MaxCents != nil && (p.UnitCents == 0 || p.UnitCents > *q.MaxCents)) {
			continue
		}
		matches := true
		for _, word := range strings.Fields(normalize(q.Text)) {
			if !strings.Contains(normalize(p.Name+" "+p.Source), word) {
				matches = false
				break
			}
		}
		if matches {
			items = append(items, p)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		a, b := items[i], items[j]
		switch q.Order {
		case "potencia":
			if a.Watts != b.Watts {
				return a.Watts < b.Watts
			}
		case "-potencia":
			if a.Watts != b.Watts {
				return a.Watts > b.Watts
			}
		case "precio":
			if a.UnitCents != b.UnitCents {
				return a.UnitCents < b.UnitCents
			}
		}
		if normalize(a.Name) != normalize(b.Name) {
			return normalize(a.Name) < normalize(b.Name)
		}
		return a.ID < b.ID
	})
	result := Page{Items: []domain.Panel{}, Total: len(items), Page: q.Page, Size: q.Size}
	start := (q.Page - 1) * q.Size
	if start < len(items) {
		result.Items = items[start:min(start+q.Size, len(items))]
	}
	return result
}
