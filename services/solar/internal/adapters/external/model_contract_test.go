package external

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"renewable.local/solar/internal/adapters/simulator"
	"renewable.local/solar/internal/domain"
	"testing"
	"time"
)

/**
 * Comprueba el comportamiento y los invariantes indicados por ModelContractAndStock.
 *
 * @method TestModelContractAndStock
 * @param {*testing.T} t - Contexto de verificación de la prueba.
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func TestModelContractAndStock(t *testing.T) {
	server := httptest.NewServer(simulator.NewHandler(testToken, testToken))
	defer server.Close()
	client, err := NewClient(server.URL, testToken, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	quote, err := client.QuoteModel(context.Background(), 1100, "PANEL-SIM-550")
	if err != nil || quote.SKU != "PANEL-SIM-550" || quote.Quantity != 2 || quote.TotalCents != 60000000 {
		t.Fatal(quote, err)
	}
	if _, err = client.QuoteModel(context.Background(), 100000, "PANEL-SIM-550"); !errors.Is(err, domain.ErrStock) {
		t.Fatal("no detectó stock", err)
	}
	raw, err := os.ReadFile("../../../../../docs/contracts/paneles-sim-v2/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if json.Unmarshal(raw, &doc) != nil || doc["info"].(map[string]any)["version"] != "0.2.0-simulated" {
		t.Fatal("contrato v2")
	}
}

/**
 * Comprueba el comportamiento y los invariantes indicados por RequestedModelCannotBeSubstituted.
 *
 * @method TestRequestedModelCannotBeSubstituted
 * @param {*testing.T} t - Contexto de verificación de la prueba.
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func TestRequestedModelCannotBeSubstituted(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(validQuote))
	}))
	defer server.Close()
	client, _ := NewClient(server.URL, testToken, time.Second)
	if _, err := client.QuoteModel(context.Background(), 494, "PANEL-SIM-450"); !errors.Is(err, domain.ErrContract) {
		t.Fatal("SKU cambiado sin rechazo", err)
	}
}
