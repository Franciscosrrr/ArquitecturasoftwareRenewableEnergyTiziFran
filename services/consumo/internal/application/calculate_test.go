package application

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

/**
 * Carga la solicitud de referencia del contrato para las pruebas del algoritmo.
 *
 * @method example
 * @param {*testing.T} t - Contexto de verificación de la prueba.
 * @returns {[]byte} Cuerpo JSON del ejemplo publicado usado como entrada de la prueba.
 */
func example(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile("../../../../docs/contracts/v1/examples/solicitud-valida.json")
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

/**
 * Comprueba el comportamiento y los invariantes indicados por MixedConsumption.
 *
 * @method TestMixedConsumption
 * @param {*testing.T} t - Contexto de verificación de la prueba.
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func TestMixedConsumption(t *testing.T) {
	r, err := Calculate(context.Background(), example(t), "trace-test")
	if err != nil {
		t.Fatal(err)
	}
	if r.Total != "47.400000" || r.Average != "1.580000" || len(r.Breakdown) != 3 {
		t.Fatalf("unexpected %+v", r)
	}
	for i, want := range []json.Number{"6.000000", "32.400000", "9.000000"} {
		if r.Breakdown[i].Energy != want {
			t.Fatal(r)
		}
	}
}

/**
 * Comprueba el comportamiento y los invariantes indicados por Validation.
 *
 * @method TestValidation
 * @param {*testing.T} t - Contexto de verificación de la prueba.
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func TestValidation(t *testing.T) {
	cases := map[string]string{
		"not-object": `[]`, "null": `null`, "malformed": `{`, "extra-json": `{} {}`, "missing-days": `{"equipos":[]}`, "empty": `{"diasPeriodo":30,"equipos":[]}`, "null-days": `{"diasPeriodo":null,"equipos":[]}`, "bool": `{"diasPeriodo":true,"equipos":[]}`, "extra": `{"diasPeriodo":30,"equipos":[],"otro":1}`,
	}
	base := `{"diasPeriodo":30,"equipos":[{"id":"a","modo":"POTENCIA","cantidad":1,"potenciaW":100,"horasPorDia":5,"diasUso":30,"factorFuncionamiento":1}]}`
	for name, replacement := range map[string][2]string{"bad-id": {`"a"`, `"a b"`}, "mode": {`"POTENCIA"`, `"OTRO"`}, "decimal-quantity": {`"cantidad":1`, `"cantidad":1.5`}, "zero-quantity": {`"cantidad":1`, `"cantidad":0`}, "high-quantity": {`"cantidad":1`, `"cantidad":1001`}, "zero-power": {`"potenciaW":100`, `"potenciaW":0`}, "hours": {`"horasPorDia":5`, `"horasPorDia":25`}, "days": {`"diasUso":30`, `"diasUso":31`}, "factor": {`"factorFuncionamiento":1`, `"factorFuncionamiento":1.1`}, "precision": {`"potenciaW":100`, `"potenciaW":0.1234567`}, "exponent": {`"potenciaW":100`, `"potenciaW":1e999999999`}, "null-power": {`"potenciaW":100`, `"potenciaW":null`}, "opposite-mode-field": {`"potenciaW":100`, `"potenciaW":100,"ciclosPeriodo":1`}, "null-item": {`{"id":"a","modo":"POTENCIA","cantidad":1,"potenciaW":100,"horasPorDia":5,"diasUso":30,"factorFuncionamiento":1}`, `null`}} {
		cases[name] = strings.Replace(base, replacement[0], replacement[1], 1)
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := Calculate(context.Background(), []byte(raw), "t"); err == nil {
				t.Fatal("invalid input accepted")
			}
		})
	}
	duplicate := strings.Replace(base, `}]}`, `},{"id":"a","modo":"CICLOS","cantidad":1,"energiaPorCicloKWh":1,"ciclosPeriodo":1}]}`, 1)
	if _, err := Calculate(context.Background(), []byte(duplicate), "t"); err == nil {
		t.Fatal("duplicate accepted")
	}
}

/**
 * Comprueba el comportamiento y los invariantes indicados por RoundingWithoutIntermediateLoss.
 *
 * @method TestRoundingWithoutIntermediateLoss
 * @param {*testing.T} t - Contexto de verificación de la prueba.
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func TestRoundingWithoutIntermediateLoss(t *testing.T) {
	raw := `{"diasPeriodo":1,"equipos":[{"id":"a","modo":"POTENCIA","cantidad":1,"potenciaW":0.001,"horasPorDia":0.5,"diasUso":1,"factorFuncionamiento":1},{"id":"b","modo":"POTENCIA","cantidad":1,"potenciaW":0.001,"horasPorDia":0.5,"diasUso":1,"factorFuncionamiento":1}]}`
	r, err := Calculate(context.Background(), []byte(raw), "t")
	if err != nil {
		t.Fatal(err)
	}
	if r.Total != "0.000001" || r.Breakdown[0].Energy != "0.000001" || r.Breakdown[1].Energy != "0.000001" {
		t.Fatal(r)
	}
}

/**
 * Comprueba el comportamiento y los invariantes indicados por CyclesArePerUnitAndZeroIsValid.
 *
 * @method TestCyclesArePerUnitAndZeroIsValid
 * @param {*testing.T} t - Contexto de verificación de la prueba.
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func TestCyclesArePerUnitAndZeroIsValid(t *testing.T) {
	for _, cycles := range []string{"0", "12"} {
		raw := `{"diasPeriodo":30,"equipos":[{"id":"a","modo":"CICLOS","cantidad":2,"energiaPorCicloKWh":0.75,"ciclosPeriodo":` + cycles + `}]}`
		r, err := Calculate(context.Background(), []byte(raw), "t")
		if err != nil {
			t.Fatal(err)
		}
		want := json.Number("18.000000")
		if cycles == "0" {
			want = "0.000000"
		}
		if r.Total != want {
			t.Fatal(r)
		}
	}
}

/**
 * Suministra fichas controladas para verificar el cálculo sin infraestructura.
 *
 * @method fakeCatalog
 * @returns {fakeCatalog} Representación tipada usada por este componente.
 */
type fakeCatalog struct {
	items []Ficha
	err   error
}

/**
 * Obtiene fichas canónicas activas para calcular con sus versiones exactas.
 *
 * @method Resolve
 * @returns {[]Ficha, error} Fichas activas solicitadas o error de validación, ausencia o dependencia.
 */
func (f fakeCatalog) Resolve(context.Context, []string) ([]Ficha, error) { return f.items, f.err }

/**
 * Comprueba el comportamiento y los invariantes indicados por CatalogUsesServerPower.
 *
 * @method TestCatalogUsesServerPower
 * @param {*testing.T} t - Contexto de verificación de la prueba.
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func TestCatalogUsesServerPower(t *testing.T) {
	raw := `{"diasPeriodo":30,"equipos":[{"catalogoId":"led","cantidad":4,"horasPorDia":5,"diasUso":30,"factorFuncionamiento":1}]}`
	catalog := fakeCatalog{items: []Ficha{{ID: "led", Mode: "POTENCIA", Power: "10"}}}
	r, err := CalculateCatalog(context.Background(), []byte(raw), "t", catalog)
	if err != nil || r.Total != "6.000000" {
		t.Fatalf("%+v %v", r, err)
	}
	if _, err = CalculateCatalog(context.Background(), []byte(raw), "t", fakeCatalog{}); err == nil {
		t.Fatal("missing item accepted")
	}
	tampered := strings.Replace(raw, `"cantidad":4`, `"cantidad":4,"potenciaW":1`, 1)
	if _, err = CalculateCatalog(context.Background(), []byte(tampered), "t", catalog); err == nil {
		t.Fatal("client power accepted")
	}
}
