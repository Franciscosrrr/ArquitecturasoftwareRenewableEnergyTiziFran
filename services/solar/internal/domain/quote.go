package domain

import (
	"encoding/json"
	"errors"
	"math/big"
	"strconv"
	"strings"
)

/**
 * Limita exponentes y tamaño antes de construir un racional de entrada.
 *
 * @method boundedRat
 * @param {json.Number} value - Número JSON de tamaño acotado.
 * @returns {*big.Rat, bool} Racional y validez del número.
 */
func boundedRat(value json.Number) (*big.Rat, bool) {
	raw := string(value)
	if len(raw) > 32 {
		return nil, false
	}
	if _, exponent, found := strings.Cut(strings.ToLower(raw), "e"); found {
		n, err := strconv.Atoi(exponent)
		if err != nil || n < -12 || n > 12 {
			return nil, false
		}
	}
	r, ok := new(big.Rat).SetString(raw)
	if !ok || !new(big.Rat).Mul(r, big.NewRat(1000000, 1)).IsInt() {
		return nil, false
	}
	return r, true
}

var (
	ErrValidation  = errors.New("datos solares inválidos")
	ErrUnavailable = errors.New("dependencia no disponible")
	ErrContract    = errors.New("respuesta incompatible del proveedor")
	ErrStock       = errors.New("stock insuficiente")
)

/**
 * Conserva un número JSON sin aceptar cadenas ni valores nulos.
 */
type DecimalNumber string

/**
 * Serializa supuestos como números JSON y conserva su precisión decimal.
 *
 * @method MarshalJSON
 * @returns {[]byte, error} Token numérico válido o error de representación.
 */
func (n DecimalNumber) MarshalJSON() ([]byte, error) {
	if _, ok := boundedRat(json.Number(n)); !ok {
		return nil, ErrValidation
	}
	return []byte(n), nil
}

/**
 * Valida la representación numérica antes de adaptar una solicitud solar.
 *
 * @method UnmarshalJSON
 * @param {[]byte} raw - Token recibido en JSON.
 * @returns {error} Error si el valor no es un número JSON.
 */
func (n *DecimalNumber) UnmarshalJSON(raw []byte) error {
	if len(raw) == 0 || raw[0] == '"' || string(raw) == "null" {
		return ErrValidation
	}
	var value json.Number
	if json.Unmarshal(raw, &value) != nil {
		return ErrValidation
	}
	if _, ok := boundedRat(value); !ok {
		return ErrValidation
	}
	*n = DecimalNumber(value)
	return nil
}

/**
 * Representa la energía calculada por Consumo, sin duplicar sus reglas.
 */
type Consumption struct {
	Total json.Number `json:"totalPeriodoKWh"`
	Daily json.Number `json:"promedioDiarioKWh"`
	Days  int         `json:"diasPeriodo"`
}

/**
 * Representa una cotización comercial; no reserva ni compra paneles.
 */
type Quote struct {
	ID         string `json:"cotizacionId"`
	SKU        string `json:"sku"`
	PanelWatts int64  `json:"potenciaPanelW"`
	Quantity   int64  `json:"cantidad"`
	Stock      int64  `json:"stockDisponible"`
	UnitCents  int64  `json:"precioUnitarioCentavos"`
	TotalCents int64  `json:"totalCentavos"`
	Currency   string `json:"moneda"`
}

/**
 * Calcula la potencia objetivo con racionales y redondeo hacia arriba.
 *
 * @method TargetWatts
 * @param {json.Number} daily - Energía diaria estimada en kWh.
 * @param {json.Number} sun - Horas solares pico entre 0.5 y 8.
 * @param {json.Number} yield - Rendimiento entre 0.1 y 1.
 * @returns {int64, error} Potencia en W o error de validación.
 */
func TargetWatts(daily, sun, yield json.Number) (int64, error) {
	d, okD := boundedRat(daily)
	s, okS := boundedRat(sun)
	y, okY := boundedRat(yield)
	if !okD || !okS || !okY || d.Sign() <= 0 || d.Cmp(big.NewRat(100000, 1)) > 0 || s.Cmp(big.NewRat(1, 2)) < 0 || s.Cmp(big.NewRat(8, 1)) > 0 || y.Cmp(big.NewRat(1, 10)) < 0 || y.Cmp(big.NewRat(1, 1)) > 0 {
		return 0, ErrValidation
	}
	w := new(big.Rat).Quo(new(big.Rat).Mul(d, big.NewRat(1000, 1)), new(big.Rat).Mul(s, y))
	n, remainder := new(big.Int), new(big.Int)
	n.QuoRem(w.Num(), w.Denom(), remainder)
	if remainder.Sign() > 0 {
		n.Add(n, big.NewInt(1))
	}
	if !n.IsInt64() || n.Int64() > 10000000 {
		return 0, ErrValidation
	}
	return n.Int64(), nil
}

/**
 * Verifica cobertura, stock y consistencia monetaria del contrato externo.
 *
 * @method Validate
 * @param {int64} target - Potencia objetivo solicitada al vendedor.
 * @returns {error} Incompatibilidad, falta de stock o nil.
 */
func (q Quote) Validate(target int64) error {
	if q.ID == "" || len(q.ID) > 64 || q.SKU == "" || len(q.SKU) > 64 || q.PanelWatts < 1 || q.PanelWatts > 2000 || q.Quantity < 1 || q.Quantity > 1000 || q.Stock < 0 || q.Stock > 1000000 || q.UnitCents < 1 || q.UnitCents > 1000000000 || q.TotalCents != q.UnitCents*q.Quantity || q.Currency != "ARS" || q.Quantity != (target+q.PanelWatts-1)/q.PanelWatts {
		return ErrContract
	}
	if q.Stock < q.Quantity {
		return ErrStock
	}
	return nil
}
