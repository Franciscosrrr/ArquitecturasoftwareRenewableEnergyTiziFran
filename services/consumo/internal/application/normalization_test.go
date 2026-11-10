package application

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

/**
 * Comprueba el comportamiento y los invariantes indicados por NumericRepresentationsPreserveContract.
 *
 * @method TestNumericRepresentationsPreserveContract
 * @param {*testing.T} t - Contexto de verificación de la prueba.
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func TestNumericRepresentationsPreserveContract(t *testing.T) {
	for _, tc := range []struct {
		power, hours, days string
		want               json.Number
	}{
		{"1e2", "5.0000000", "30", "15.000000"},
		{"100", "0e999999999999999999999", "30", "0.000000"},
		{"1" + strings.Repeat("0", 1000) + "e-1000", "5", "30", "0.150000"},
	} {
		raw := `{"diasPeriodo":30,"equipos":[{"id":"a","modo":"POTENCIA","cantidad":1,"potenciaW":` + tc.power + `,"horasPorDia":` + tc.hours + `,"diasUso":` + tc.days + `,"factorFuncionamiento":1}]}`
		result, err := Calculate(context.Background(), []byte(raw), "t")
		if err != nil || result.Total != tc.want {
			t.Fatalf("representation changed energy: %s %v", result.Total, err)
		}
	}
}
