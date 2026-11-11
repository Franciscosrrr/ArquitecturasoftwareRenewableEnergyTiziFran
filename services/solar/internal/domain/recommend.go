package domain

import (
	"encoding/json"
	"math/big"
	"sort"
)

/**
 * Describe una ficha técnica local versionada; la oferta opcional es una vista comercial no persistida.
 *
 * @method Panel
 * @returns {Panel} Representación tipada usada por este componente.
 */
type Panel struct {
	ID           string `json:"id" bson:"_id"`
	Name         string `json:"nombre"`
	Source       string `json:"fuente"`
	Watts        int64  `json:"potenciaW"`
	AreaMilli    int64  `json:"superficieEfectivaMilliM2"`
	UnitCents    int64  `json:"precioCentavos"`
	Currency     string `json:"moneda"`
	Active       bool   `json:"activo"`
	Version      int64  `json:"version"`
	Offer        *Quote `json:"oferta,omitempty" bson:"-"`
	OfferSource  string `json:"origenOferta,omitempty" bson:"-"`
	Availability string `json:"disponibilidad,omitempty" bson:"-"`
}

/**
 * Agrupa recurso solar manual, rendimiento, cobertura y superficie útil para dimensionar.
 *
 * @method StudyOptions
 * @returns {StudyOptions} Representación tipada usada por este componente.
 */
type StudyOptions struct {
	Sun      DecimalNumber  `json:"horasSolPico"`
	Yield    DecimalNumber  `json:"factorRendimiento"`
	Coverage DecimalNumber  `json:"coberturaObjetivo"`
	Area     *DecimalNumber `json:"superficieM2"`
}

/**
 * Conserva cantidad, generación diaria, cobertura, superficie y viabilidad de una alternativa.
 *
 * @method Candidate
 * @returns {Candidate} Representación tipada usada por este componente.
 */
type Candidate struct {
	Panel      Panel       `json:"panel"`
	Quantity   int64       `json:"cantidad"`
	Required   int64       `json:"cantidadRequerida"`
	Generation json.Number `json:"generacionDiariaKWh"`
	Coverage   json.Number `json:"coberturaAlcanzada"`
	Area       json.Number `json:"superficieM2"`
	TotalCents int64       `json:"totalCentavos"`
	Feasible   bool        `json:"cumpleObjetivo"`
	Watts      int64       `json:"potenciaObjetivoW"`
}

/**
 * Reúne alternativas, selección, oferta con procedencia y limitaciones del estudio.
 *
 * @method Recommendation
 * @returns {Recommendation} Representación tipada usada por este componente.
 */
type Recommendation struct {
	Status     string       `json:"estado"`
	Criteria   string       `json:"criterio"`
	Options    StudyOptions `json:"supuestos"`
	Candidates []Candidate  `json:"alternativas"`
	Selected   *Candidate   `json:"seleccion"`
	Quote      *Quote       `json:"cotizacion,omitempty"`
	Note       string       `json:"nota"`
}

/**
 * Redondea un racional no negativo a seis decimales con mitad hacia arriba.
 *
 * @method Round6
 * @param {*big.Rat} v - Valor recibido o racional a convertir según el tipo declarado.
 * @returns {json.Number} Número JSON redondeado a seis decimales, sin perder precisión en pasos intermedios.
 */
func Round6(v *big.Rat) json.Number {
	scaled := new(big.Rat).Mul(v, big.NewRat(1000000, 1))
	q, rem := new(big.Int), new(big.Int)
	q.QuoRem(scaled.Num(), scaled.Denom(), rem)
	if new(big.Int).Mul(rem, big.NewInt(2)).Cmp(scaled.Denom()) >= 0 {
		q.Add(q, big.NewInt(1))
	}
	return json.Number(new(big.Rat).SetFrac(q, big.NewInt(1000000)).FloatString(6))
}

/**
 * Obtiene el entero superior y detecta desbordamiento de int64.
 *
 * @method ceil
 * @param {*big.Rat} v - Valor recibido o racional a convertir según el tipo declarado.
 * @returns {int64} Entero superior del racional; -1 cuando el resultado no cabe en int64.
 */
func ceil(v *big.Rat) int64 {
	q, rem := new(big.Int), new(big.Int)
	q.QuoRem(v.Num(), v.Denom(), rem)
	if rem.Sign() > 0 {
		q.Add(q, big.NewInt(1))
	}
	if !q.IsInt64() {
		return -1
	}
	return q.Int64()
}

/**
 * Compara generación, superficie y costo sin efectuar operaciones de infraestructura.
 *
 * @method Recommend
 * @param {json.Number} daily - Consumo diario estimado, expresado en kWh.
 * @param {StudyOptions} o - HSP, rendimiento, cobertura y superficie del dimensionamiento.
 * @param {[]Panel} panels - Fichas locales activas con potencia, área, fuente y versión.
 * @returns {Recommendation, error} Alternativas técnicas y selección de referencia, o error por supuestos fuera de rango.
 */
func Recommend(daily json.Number, o StudyOptions, panels []Panel) (Recommendation, error) {
	result := Recommendation{Status: "SIN_ALTERNATIVA", Criteria: "menor costo de paneles, menor superficie e id", Options: o, Candidates: []Candidate{}, Note: "Recurso solar manual explícito. Estimación didáctica; no garantiza autonomía horaria ni reemplaza un estudio técnico."}
	demand, ok := boundedRat(daily)
	sun, okS := boundedRat(json.Number(o.Sun))
	yield, okY := boundedRat(json.Number(o.Yield))
	coverage, okC := boundedRat(json.Number(o.Coverage))
	if !ok || !okS || !okY || !okC || demand.Sign() < 0 || demand.Cmp(big.NewRat(1000000000000, 1)) > 0 || sun.Sign() <= 0 || sun.Cmp(big.NewRat(24, 1)) > 0 || yield.Sign() <= 0 || yield.Cmp(big.NewRat(1, 1)) > 0 || coverage.Sign() <= 0 || coverage.Cmp(big.NewRat(1, 1)) > 0 {
		return result, ErrValidation
	}
	var area *big.Rat
	if o.Area != nil {
		var valid bool
		area, valid = boundedRat(json.Number(*o.Area))
		if !valid || area.Sign() < 0 || area.Cmp(big.NewRat(1000000000, 1)) > 0 {
			return result, ErrValidation
		}
	}
	target := new(big.Rat).Mul(demand, coverage)
	for _, panel := range panels {
		if !panel.Active {
			continue
		}
		if panel.Watts < 1 || panel.Watts > 2000 || panel.AreaMilli < 1 || panel.AreaMilli > 100000 || panel.UnitCents < 0 || panel.UnitCents > 1000000000 {
			continue
		}
		generation := new(big.Rat).Mul(big.NewRat(panel.Watts, 1000), new(big.Rat).Mul(sun, yield))
		required := ceil(new(big.Rat).Quo(target, generation))
		if required < 0 || required > 1000000000 {
			return result, ErrValidation
		}
		quantity := required
		if area != nil {
			fits := new(big.Rat).Quo(new(big.Rat).Mul(area, big.NewRat(1000, 1)), big.NewRat(panel.AreaMilli, 1))
			quantity = min(quantity, new(big.Int).Quo(fits.Num(), fits.Denom()).Int64())
		}
		produced := new(big.Rat).Mul(generation, big.NewRat(quantity, 1))
		actualCoverage := new(big.Rat)
		if demand.Sign() > 0 {
			actualCoverage.Quo(produced, demand)
		}
		watts := ceil(new(big.Rat).Mul(big.NewRat(quantity*panel.Watts, 1), big.NewRat(1, 1)))
		result.Candidates = append(result.Candidates, Candidate{panel, quantity, required, Round6(produced), Round6(actualCoverage), Round6(big.NewRat(quantity*panel.AreaMilli, 1000)), quantity * panel.UnitCents, produced.Cmp(target) >= 0, watts})
	}
	hasFeasible := false
	for _, candidate := range result.Candidates {
		hasFeasible = hasFeasible || candidate.Feasible
	}
	comparable := true
	currency := ""
	for _, candidate := range result.Candidates {
		if hasFeasible && !candidate.Feasible {
			continue
		}
		panel := candidate.Panel
		if panel.UnitCents == 0 || panel.Currency == "" {
			comparable = false
		}
		if currency != "" && currency != panel.Currency {
			comparable = false
		}
		currency = panel.Currency
	}
	if !comparable {
		result.Criteria = "menor superficie, menor cantidad e id (precios no comparables)"
	}
	if len(result.Candidates) > 0 && !hasFeasible {
		result.Criteria = "mayor cobertura alcanzable; " + result.Criteria
	}
	sort.SliceStable(result.Candidates, func(i, j int) bool {
		a, b := result.Candidates[i], result.Candidates[j]
		if a.Feasible != b.Feasible {
			return a.Feasible
		}
		if !a.Feasible {
			ac, _ := boundedRat(a.Coverage)
			bc, _ := boundedRat(b.Coverage)
			if ac.Cmp(bc) != 0 {
				return ac.Cmp(bc) > 0
			}
		}
		if comparable && a.TotalCents != b.TotalCents {
			return a.TotalCents < b.TotalCents
		}
		av := a.Quantity * a.Panel.AreaMilli
		bv := b.Quantity * b.Panel.AreaMilli
		if av != bv {
			return av < bv
		}
		if !comparable && a.Quantity != b.Quantity {
			return a.Quantity < b.Quantity
		}
		return a.Panel.ID < b.Panel.ID
	})
	if len(result.Candidates) > 0 {
		selected := result.Candidates[0]
		result.Selected = &selected
		if selected.Feasible {
			result.Status = "COMPLETADO"
		}
	}
	if demand.Sign() == 0 {
		result.Status = "COMPLETADO"
		result.Note = "Consumo cero: no se requieren paneles ni cotización."
	}
	return result, nil
}
