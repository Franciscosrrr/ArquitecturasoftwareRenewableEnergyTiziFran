package transport

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

/**
 * Congela el OpenAPI 1.0.1 entregado a Clínica y verifica la implementación real.
 *
 * @method TestClinicaV101Compatibility
 * @param {*testing.T} t - Contexto de prueba con ejemplos publicados.
 * @returns {void} Falla ante cambios del contrato, energía o permisos.
 */
func TestClinicaV101Compatibility(t *testing.T) {
	base := "../../../../docs/contracts/v1/"
	doc, err := os.ReadFile(base + "openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(strings.ReplaceAll(string(doc), "\r\n", "\n")))
	if hex.EncodeToString(sum[:]) != "3876121082fa76cd7ca6b25defbc7edc45ea0a4a89b3ee851fcf681e446ffda3" {
		t.Fatal("OpenAPI 1.0.1 cambió: acordar nueva versión antes de modificar la referencia")
	}
	request, err := os.ReadFile(base + "examples/solicitud-valida.json")
	if err != nil {
		t.Fatal(err)
	}
	key, denied := strings.Repeat("a", 32), strings.Repeat("b", 32)
	for _, tc := range []struct {
		key    string
		status int
		code   string
	}{
		{key, 200, ""}, {"", 401, "NO_AUTENTICADO"}, {denied, 403, "SIN_PERMISO"},
	} {
		handler := NewHandler(testHealthToken, Config{APIKey: key, DeniedKey: denied})
		req := httptest.NewRequest("POST", "/api/v1/estimaciones-consumo", strings.NewReader(string(request)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", tc.key)
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != tc.status {
			t.Fatalf("%d %s", res.Code, res.Body.String())
		}
		var body map[string]any
		if json.Unmarshal(res.Body.Bytes(), &body) != nil {
			t.Fatal("JSON")
		}
		if tc.status == 200 {
			if body["totalPeriodoKWh"] != 47.4 || body["promedioDiarioKWh"] != 1.58 || body["versionAlgoritmo"] != "consumo-v1" || len(body) != 7 {
				t.Fatalf("cambio incompatible: %v", body)
			}
			if _, exists := body["precio"]; exists {
				t.Fatal("Clínica calcula su tarifa")
			}
		} else if body["codigo"] != tc.code || len(body) != 4 {
			t.Fatalf("error incompatible: %v", body)
		}
	}
}

/**
 * Verifica los perfiles publicados por Clínica sin agregar tarifas a la API.
 *
 * @method TestClinicaEquipmentProfiles
 * @param {*testing.T} t - Contexto de compatibilidad del consumidor real.
 * @returns {void} Falla si cambia el cálculo o el redondeo de sus solicitudes.
 */
func TestClinicaEquipmentProfiles(t *testing.T) {
	key := strings.Repeat("c", 32)
	for _, tc := range []struct{ hours, extra, want string }{
		{"0.5", "", "0.371000"},
		{"0.5", `,{"id":"electrocardiografo","modo":"CICLOS","cantidad":1,"energiaPorCicloKWh":0.01,"ciclosPeriodo":1}`, "0.381000"},
		{"0.5", `,{"id":"balanza-pediatrica","modo":"POTENCIA","cantidad":1,"potenciaW":5,"horasPorDia":0.5,"diasUso":1,"factorFuncionamiento":1},{"id":"esterilizador","modo":"CICLOS","cantidad":1,"energiaPorCicloKWh":0.05,"ciclosPeriodo":1}`, "0.423500"},
		{"0.333333", "", "0.247333"},
		{"0.166667", "", "0.123667"},
	} {
		body := fmt.Sprintf(`{"diasPeriodo":1,"equipos":[{"id":"iluminacion-led","modo":"POTENCIA","cantidad":4,"potenciaW":18,"horasPorDia":%s,"diasUso":1,"factorFuncionamiento":1},{"id":"aire-acondicionado","modo":"POTENCIA","cantidad":1,"potenciaW":1100,"horasPorDia":%s,"diasUso":1,"factorFuncionamiento":0.5},{"id":"pc-consultorio","modo":"POTENCIA","cantidad":1,"potenciaW":120,"horasPorDia":%s,"diasUso":1,"factorFuncionamiento":1}%s]}`, tc.hours, tc.hours, tc.hours, tc.extra)
		req := httptest.NewRequest("POST", "/api/v1/estimaciones-consumo", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", key)
		res := httptest.NewRecorder()
		NewHandler(testHealthToken, Config{APIKey: key}).ServeHTTP(res, req)
		var result struct {
			Total json.Number `json:"totalPeriodoKWh"`
		}
		if res.Code != 200 || json.Unmarshal(res.Body.Bytes(), &result) != nil || string(result.Total) != tc.want {
			t.Fatalf("clínica: %d %s", res.Code, res.Body.String())
		}
	}
}
