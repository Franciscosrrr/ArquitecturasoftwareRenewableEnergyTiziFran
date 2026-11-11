package domain

import (
	"encoding/json"
	"math/big"
)

/**
 * Valida una oferta ARS con flete y descuento sin exigir un identificador inexistente.
 *
 * @method validateCommercial
 * @param {int64} target - Potencia solicitada, calculada con la ficha técnica local.
 * @returns {error} Contrato incompatible, stock insuficiente o validación satisfactoria.
 */
func (q Quote) validateCommercial(target int64) error {
	if target < 1 || q.ID != "" || q.SKU == "" || q.ProviderSKU == "" || q.ProviderID < 1 || q.ProviderName == "" || len(q.ProviderName) > 200 || q.RequestID == "" || len(q.RequestID) > 128 || q.PanelWatts < 1 || q.PanelWatts > 2000 || q.Quantity < 1 || q.Quantity > 1000 || q.Quantity != (target+q.PanelWatts-1)/q.PanelWatts || q.Stock < 0 || q.UnitCents < 1 || q.UnitCents > 1000000000 || q.TotalCents < 0 || q.TotalCents > 1000000000000000 || q.FreightCents < 0 || q.FreightCents > 1000000000000000 || q.Currency != "ARS" {
		return ErrContract
	}
	discount, ok := boundedRat(json.Number(q.Discount))
	if !ok || discount.Sign() < 0 || discount.Cmp(big.NewRat(100, 1)) > 0 {
		return ErrContract
	}
	total := new(big.Rat).SetInt64(q.UnitCents * q.Quantity)
	if q.DiscountApplied {
		total.Mul(total, new(big.Rat).Sub(big.NewRat(1, 1), new(big.Rat).Quo(discount, big.NewRat(100, 1))))
	}
	total.Add(total, big.NewRat(q.FreightCents, 1))
	cents, rem := new(big.Int), new(big.Int)
	cents.QuoRem(total.Num(), total.Denom(), rem)
	if new(big.Int).Mul(rem, big.NewInt(2)).Cmp(total.Denom()) >= 0 {
		cents.Add(cents, big.NewInt(1))
	}
	if !cents.IsInt64() || cents.Int64() != q.TotalCents {
		return ErrContract
	}
	if q.Stock < q.Quantity {
		return ErrStock
	}
	return nil
}
