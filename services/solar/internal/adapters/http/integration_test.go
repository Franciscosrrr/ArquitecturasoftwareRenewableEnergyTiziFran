package transport

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"renewable.local/solar/internal/adapters/external"
	"renewable.local/solar/internal/adapters/simulator"
	"renewable.local/solar/internal/application"
	"strings"
	"testing"
	"time"
)

/**
 * Ejercita el caso completo contra dos servidores HTTP y la caída del vendedor.
 *
 * @method TestSolarHTTPFlowAndFailure
 * @param {*testing.T} t - Contexto de integración local HTTP.
 * @returns {void} Falla si una cotización inválida se presenta como exitosa.
 */
func TestSolarHTTPFlowAndFailure(t *testing.T) {
	consumo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/publico/consumo-catalogo" {
			t.Error("Solar debe llamar directamente a Consumo")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"totalPeriodoKWh":47.4,"promedioDiarioKWh":1.58,"diasPeriodo":30}`))
	}))
	defer consumo.Close()
	supplier := httptest.NewServer(simulator.NewHandler(testHealthToken, testHealthToken))
	defer supplier.Close()
	estimator, _ := external.NewClient(consumo.URL, "", time.Second)
	provider, _ := external.NewClient(supplier.URL, testHealthToken, time.Second)
	handler := NewHandler(testHealthToken, application.Service{Estimator: estimator, Supplier: provider, Mode: "simulado"})
	input := `{"consumo":{"diasPeriodo":30,"equipos":[]},"horasSolPico":4,"factorRendimiento":0.8}`
	for _, tc := range []struct {
		body   string
		status int
	}{
		{input, 200}, {strings.Replace(input, `:4,`, `:"4",`, 1), 422},
		{strings.Replace(input, `:4,`, `:1e999999999,`, 1), 422},
		{strings.Replace(input, `:0.8}`, `:0}`, 1), 422}, {input + `{}`, 400},
	} {
		req := httptest.NewRequest("POST", "/api/v1/publico/solar/cotizaciones", strings.NewReader(tc.body))
		req.Header.Set("Content-Type", "application/json")
		res := httptest.NewRecorder()
		handler.ServeHTTP(res, req)
		if res.Code != tc.status {
			t.Fatalf("%s: %d %s", tc.body, res.Code, res.Body.String())
		}
		if tc.status == 200 {
			var result application.Result
			if json.Unmarshal(res.Body.Bytes(), &result) != nil || result.Watts != 494 || result.Quote.Quantity != 2 || result.Mode != "simulado" {
				t.Fatal(res.Body.String())
			}
		}
	}
	supplier.Close()
	req := httptest.NewRequest("POST", "/api/v1/publico/solar/cotizaciones", strings.NewReader(input))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != 503 || !strings.Contains(res.Body.String(), "Tu estimación de consumo sigue disponible") {
		t.Fatalf("caída: %d %s", res.Code, res.Body.String())
	}
}
