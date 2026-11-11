package application

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel"
)

/**
 * Identifica un campo rechazado y el motivo público de validación.
 *
 * @method Detail
 * @returns {Detail} Representación tipada usada por este componente.
 */
type Detail struct {
	Field  string `json:"campo"`
	Reason string `json:"motivo"`
}

/**
 * Transporta una infracción de negocio con campo y causa sin detalles internos.
 *
 * @method ValidationError
 * @returns {ValidationError} Representación tipada usada por este componente.
 */
type ValidationError struct{ Detail Detail }

/**
 * Devuelve la representación pública del error sin incluir credenciales.
 *
 * @method Error
 * @returns {string} Descripción pública y saneada del error.
 */
func (e *ValidationError) Error() string { return e.Detail.Field + ": " + e.Detail.Reason }

/**
 * Construye un error de validación con campo y causa públicos.
 *
 * @method invalid
 * @param {string} field - Ruta del campo que incumple la regla de validación.
 * @param {string} reason - Motivo público del rechazo del campo.
 * @returns {error} Error de validación con campo y motivo públicos.
 */
func invalid(field, reason string) error { return &ValidationError{Detail{field, reason}} }

/**
 * Relaciona el identificador del equipo con su energía estimada en kWh.
 *
 * @method Breakdown
 * @returns {Breakdown} Representación tipada usada por este componente.
 */
type Breakdown struct {
	ID     string      `json:"id"`
	Energy json.Number `json:"energiaPeriodoKWh"`
}

/**
 * Reúne consumo total y diario, desglose, supuestos, algoritmo y correlación del cálculo.
 *
 * @method Result
 * @returns {Result} Representación tipada usada por este componente.
 */
type Result struct {
	Version     string      `json:"versionAlgoritmo"`
	Days        int         `json:"diasPeriodo"`
	Breakdown   []Breakdown `json:"desglose"`
	Total       json.Number `json:"totalPeriodoKWh"`
	Average     json.Number `json:"promedioDiarioKWh"`
	Assumptions []string    `json:"supuestos"`
	TraceID     string      `json:"traceId"`
}

/**
 * Decodifica un objeto JSON sin aceptar campos duplicados ni valores fuera de forma.
 *
 * @method Object
 * @param {[]byte} raw - Cuerpo original recibido para validar.
 * @returns {map[string]any, error} Objeto JSON con números conservados; error ante forma, claves duplicadas o datos inválidos.
 */
func Object(raw []byte) (map[string]any, error) {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var obj map[string]any
	if d.Decode(&obj) != nil || obj == nil {
		return nil, invalid("$", "Se requiere un objeto JSON.")
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return nil, invalid("$", "Se requiere un único objeto JSON.")
	}
	return obj, nil
}

/**
 * Exige los campos permitidos y obligatorios de la variante de equipo.
 *
 * @method shape
 * @param {map[string]any} m - Objeto JSON ya decodificado cuyos campos se verifican.
 * @param {[]string} required - Nombres de campos que deben estar presentes.
 * @param {[]string} allowed - Conjunto de campos admitidos para esta variante.
 * @param {string} path - Ruta del campo JSON o de la operación HTTP indicada por este adaptador.
 * @returns {error} nil si están los campos requeridos y no hay campos ajenos; error de validación en otro caso.
 */
func shape(m map[string]any, required, allowed []string, path string) error {
	for _, key := range required {
		if value, ok := m[key]; !ok || value == nil {
			return invalid(path+key, "Campo obligatorio y no nulo.")
		}
	}
	for key, value := range m {
		found := false
		for _, a := range allowed {
			if key == a {
				found = true
				break
			}
		}
		if !found || value == nil {
			return invalid(path+key, "Campo desconocido o nulo.")
		}
	}
	return nil
}

/**
 * Lee un número JSON con representación acotada antes de efectuar aritmética.
 *
 * @method number
 * @param {map[string]any} m - Objeto JSON ya decodificado cuyos campos se verifican.
 * @param {string} key - Clave de idempotencia o identidad técnica.
 * @param {string} path - Ruta del campo JSON o de la operación HTTP indicada por este adaptador.
 * @param {int64} lo - Límite inferior permitido para el número.
 * @param {int64} hi - Límite superior permitido para el número.
 * @param {bool} positive - Exige un valor estrictamente positivo cuando es verdadero.
 * @param {bool} integer - Exige que el número no tenga fracción cuando es verdadero.
 * @returns {*big.Rat, error} Racional exacto dentro de los límites o error de validación con la ruta del campo.
 */
func number(m map[string]any, key, path string, lo, hi int64, positive, integer bool) (*big.Rat, error) {
	value, ok := m[key].(json.Number)
	if !ok {
		return nil, invalid(path+key, "Se requiere un número.")
	}
	n, ok := decimal(string(value))
	if !ok || n.Cmp(big.NewRat(lo, 1)) < 0 || n.Cmp(big.NewRat(hi, 1)) > 0 || (positive && n.Sign() == 0) {
		return nil, invalid(path+key, "Número fuera del rango permitido.")
	}
	if (integer && !n.IsInt()) || !new(big.Rat).Mul(n, big.NewRat(1000000, 1)).IsInt() {
		return nil, invalid(path+key, "Revisar tipo entero y precisión máxima de seis decimales.")
	}
	return n, nil
}

// Normalize zeroes/exponents before allocating big integers. Every accepted
// nonzero value has at most six fractional digits and fits the business range.
/**
 * Convierte un número validado a racional exacto con seis decimales efectivos.
 *
 * @method decimal
 * @param {string} raw - Cuerpo original recibido para validar.
 * @returns {*big.Rat, bool} Racional exacto y bandera de validez de su representación decimal.
 */
func decimal(raw string) (*big.Rat, bool) {
	mantissa, exponentText, hasExponent := strings.Cut(strings.ToLower(raw), "e")
	negative := strings.HasPrefix(mantissa, "-")
	mantissa = strings.TrimPrefix(mantissa, "-")
	fraction := 0
	if dot := strings.IndexByte(mantissa, '.'); dot >= 0 {
		fraction = len(mantissa) - dot - 1
	}
	digits := strings.TrimLeft(strings.ReplaceAll(mantissa, ".", ""), "0")
	if digits == "" {
		return new(big.Rat), true
	}
	exponent := int64(0)
	if hasExponent {
		var err error
		exponent, err = strconv.ParseInt(exponentText, 10, 64)
		if err != nil || exponent > int64(len(raw)+20) || exponent < -int64(len(raw)+20) {
			return nil, false
		}
	}
	trimmed := strings.TrimRight(digits, "0")
	scale := exponent - int64(fraction) + int64(len(digits)-len(trimmed))
	digits = trimmed
	if scale < -6 || scale > 16 || len(digits) > 16 {
		return nil, false
	}
	numerator, _ := new(big.Int).SetString(digits, 10)
	if negative {
		numerator.Neg(numerator)
	}
	power := new(big.Int).Exp(big.NewInt(10), big.NewInt(max(scale, -scale)), nil)
	if scale >= 0 {
		return new(big.Rat).SetInt(new(big.Int).Mul(numerator, power)), true
	}
	return new(big.Rat).SetFrac(numerator, power), true
}

var idPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

/**
 * Redondea una salida no negativa a seis decimales con mitad hacia arriba.
 *
 * @method rounded
 * @param {*big.Rat} n - Racional exacto que se redondeará para la respuesta JSON.
 * @returns {json.Number} Número JSON con seis decimales y redondeo mitad hacia arriba.
 */
func rounded(n *big.Rat) json.Number {
	scaled := new(big.Rat).Mul(n, big.NewRat(1000000, 1))
	q, rem := new(big.Int), new(big.Int)
	q.QuoRem(scaled.Num(), scaled.Denom(), rem)
	if new(big.Int).Mul(rem, big.NewInt(2)).Cmp(scaled.Denom()) >= 0 {
		q.Add(q, big.NewInt(1))
	}
	whole, frac := new(big.Int), new(big.Int)
	whole.QuoRem(q, big.NewInt(1000000), frac)
	return json.Number(fmt.Sprintf("%s.%06d", whole.String(), frac.Int64()))
}

/**
 * Calcula energía por equipo y total con racionales, validación cruzada y redondeo final.
 *
 * @method Calculate
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {[]byte} raw - Cuerpo original recibido para validar.
 * @param {string} traceID - Identificador de la traza asociada al resultado o al estudio.
 * @returns {Result, error} Consumo, desglose y supuestos; error de validación si la entrada viola las reglas.
 */
func Calculate(ctx context.Context, raw []byte, traceID string) (Result, error) {
	_, span := otel.Tracer("consumo").Start(ctx, "consumo.calcular")
	defer span.End()
	data, err := Object(raw)
	if err != nil {
		return Result{}, err
	}
	keys := []string{"diasPeriodo", "equipos"}
	if err = shape(data, keys, keys, ""); err != nil {
		return Result{}, err
	}
	days, err := number(data, "diasPeriodo", "", 1, 366, false, true)
	if err != nil {
		return Result{}, err
	}
	items, ok := data["equipos"].([]any)
	if !ok || len(items) < 1 || len(items) > 100 {
		return Result{}, invalid("equipos", "Se requieren entre 1 y 100 equipos.")
	}
	result := Result{Version: "consumo-v1", Days: int(days.Num().Int64()), Breakdown: []Breakdown{}, Assumptions: []string{"Estimación basada en equipos y hábitos declarados; no es una medición real.", "El factor de funcionamiento se aplica una sola vez."}, TraceID: traceID}
	total := new(big.Rat)
	seen := map[string]bool{}
	for i, item := range items {
		path := fmt.Sprintf("equipos[%d].", i)
		m, ok := item.(map[string]any)
		if !ok {
			return Result{}, invalid(path, "Se requiere un objeto.")
		}
		id, ok := m["id"].(string)
		if !ok || !idPattern.MatchString(id) || seen[id] {
			return Result{}, invalid(path+"id", "Identificador inválido o repetido.")
		}
		seen[id] = true
		mode, _ := m["modo"].(string)
		keys := []string{"id", "modo", "cantidad"}
		switch mode {
		case "POTENCIA":
			keys = append(keys, "potenciaW", "horasPorDia", "diasUso", "factorFuncionamiento")
		case "CICLOS":
			keys = append(keys, "energiaPorCicloKWh", "ciclosPeriodo")
		default:
			return Result{}, invalid(path+"modo", "Utilice POTENCIA o CICLOS.")
		}
		if err = shape(m, keys, keys, path); err != nil {
			return Result{}, err
		}
		quantity, err := number(m, "cantidad", path, 1, 1000, false, true)
		if err != nil {
			return Result{}, err
		}
		energy := new(big.Rat).Set(quantity)
		fields := []struct {
			key               string
			lo, hi            int64
			positive, integer bool
		}{}
		if mode == "POTENCIA" {
			fields = append(fields, struct {
				key               string
				lo, hi            int64
				positive, integer bool
			}{"potenciaW", 0, 1000000, true, false}, struct {
				key               string
				lo, hi            int64
				positive, integer bool
			}{"horasPorDia", 0, 24, false, false}, struct {
				key               string
				lo, hi            int64
				positive, integer bool
			}{"diasUso", 0, int64(result.Days), false, true}, struct {
				key               string
				lo, hi            int64
				positive, integer bool
			}{"factorFuncionamiento", 0, 1, true, false})
		} else {
			fields = append(fields, struct {
				key               string
				lo, hi            int64
				positive, integer bool
			}{"energiaPorCicloKWh", 0, 1000, true, false}, struct {
				key               string
				lo, hi            int64
				positive, integer bool
			}{"ciclosPeriodo", 0, 10000, false, true})
		}
		for _, f := range fields {
			n, err := number(m, f.key, path, f.lo, f.hi, f.positive, f.integer)
			if err != nil {
				return Result{}, err
			}
			energy.Mul(energy, n)
		}
		if mode == "POTENCIA" {
			energy.Quo(energy, big.NewRat(1000, 1))
		}
		total.Add(total, energy)
		result.Breakdown = append(result.Breakdown, Breakdown{id, rounded(energy)})
	}
	result.Total = rounded(total)
	result.Average = rounded(new(big.Rat).Quo(total, days))
	return result, nil
}

// Ficha is Consumo's own representation of the public catalog contract.
type Ficha struct {
	Version int64       `json:"version"`
	ID      string      `json:"id"`
	Name    string      `json:"nombre"`
	Mode    string      `json:"modo"`
	Power   json.Number `json:"potenciaW"`
	Cycle   json.Number `json:"energiaPorCicloKWh"`
	Factor  json.Number `json:"factorFuncionamiento"`
}

/**
 * Desacopla el cálculo de consumo de la obtención de fichas canónicas del catálogo.
 *
 * @method Resolver
 * @returns {Resolver} Representación tipada usada por este componente.
 */
type Resolver interface {
	/**
	 * Obtiene fichas canónicas activas para calcular el consumo sin depender del transporte.
	 *
	 * @method Resolve
	 * @param {context.Context} ctx - Presupuesto de lectura del catálogo.
	 * @param {[]string} ids - Identificadores de fichas activas solicitadas.
	 * @returns {[]Ficha, error} Fichas activas solicitadas o error de validación, ausencia o dependencia.
	 */
	Resolve(context.Context, []string) ([]Ficha, error)
}

/**
 * Resuelve un lote canónico y calcula consumo sin confiar potencias enviadas por el navegador.
 *
 * @method CalculateCatalog
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {[]byte} raw - Cuerpo original recibido para validar.
 * @param {string} traceID - Identificador de la traza asociada al resultado o al estudio.
 * @param {Resolver} resolver - Puerto que recupera las fichas canónicas activas del catálogo.
 * @returns {Result, error} Consumo calculado con fichas canónicas; error de entrada o de la dependencia de catálogo.
 */
func CalculateCatalog(ctx context.Context, raw []byte, traceID string, resolver Resolver) (Result, error) {
	ctx, span := otel.Tracer("consumo").Start(ctx, "consumo.resolver-catalogo")
	defer span.End()
	data, err := Object(raw)
	if err != nil {
		return Result{}, err
	}
	keys := []string{"diasPeriodo", "equipos"}
	if err = shape(data, keys, keys, ""); err != nil {
		return Result{}, err
	}
	if _, err = number(data, "diasPeriodo", "", 1, 366, false, true); err != nil {
		return Result{}, err
	}
	items, ok := data["equipos"].([]any)
	if !ok || len(items) < 1 || len(items) > 100 {
		return Result{}, invalid("equipos", "Se requieren entre 1 y 100 equipos.")
	}
	ids := []string{}
	seen := map[string]bool{}
	for i, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			return Result{}, invalid("equipos", "Se requiere un objeto.")
		}
		id, _ := m["catalogoId"].(string)
		if !idPattern.MatchString(id) || seen[id] {
			return Result{}, invalid(fmt.Sprintf("equipos[%d].catalogoId", i), "Identificador inválido o repetido.")
		}
		seen[id] = true
		ids = append(ids, id)
	}
	fichas, err := resolver.Resolve(ctx, ids)
	if err != nil {
		return Result{}, err
	}
	byID := map[string]Ficha{}
	for _, f := range fichas {
		byID[f.ID] = f
	}
	for i, item := range items {
		m := item.(map[string]any)
		path := fmt.Sprintf("equipos[%d].", i)
		f, found := byID[ids[i]]
		if !found {
			return Result{}, invalid(path+"catalogoId", "Ficha inexistente o inactiva.")
		}
		required := []string{"catalogoId", "cantidad"}
		if f.Mode == "POTENCIA" {
			required = append(required, "horasPorDia", "diasUso", "factorFuncionamiento")
		} else {
			required = append(required, "ciclosPeriodo")
		}
		if err = shape(m, required, required, path); err != nil {
			return Result{}, err
		}
		delete(m, "catalogoId")
		m["id"] = f.ID
		m["modo"] = f.Mode
		if f.Mode == "POTENCIA" {
			m["potenciaW"] = f.Power
		} else {
			m["energiaPorCicloKWh"] = f.Cycle
		}
	}
	normalized, _ := json.Marshal(data)
	return Calculate(ctx, normalized, traceID)
}
