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
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const testToken = "secreto-ficticio-de-pruebas-1234567890"
const validQuote = `{"cotizacionId":"q1","sku":"panel","potenciaPanelW":450,"cantidad":2,"stockDisponible":10,"precioUnitarioCentavos":25000000,"totalCentavos":50000000,"moneda":"ARS"}`

/**
 * Verifica la llamada comercial y la versión publicada del simulador.
 *
 * @method TestSupplierContract
 * @param {*testing.T} t - Contexto de prueba.
 * @returns {void} Falla ante incompatibilidades en operación, credencial o datos.
 */
func TestSupplierContract(t *testing.T) {
	server := httptest.NewServer(simulator.NewHandler(testToken, testToken))
	defer server.Close()
	client, err := NewClient(server.URL, testToken, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	quote, err := client.Quote(context.Background(), 494)
	if err != nil || quote.Quantity != 2 || quote.TotalCents != 50000000 {
		t.Fatalf("%+v %v", quote, err)
	}
	raw, err := os.ReadFile("../../../../../docs/contracts/paneles-sim-v1/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if json.Unmarshal(raw, &doc) != nil || doc["info"].(map[string]any)["version"] != "0.1.0-simulated" {
		t.Fatal("versión simulada")
	}
	if _, ok := doc["paths"].(map[string]any)["/api/v1/cotizaciones"]; !ok {
		t.Fatal("operación ausente")
	}
	client.Token = "incorrecta"
	if _, err = client.Quote(context.Background(), 494); !errors.Is(err, domain.ErrContract) {
		t.Fatal("401 externo debe requerir revisión de integración")
	}
}

/**
 * Usa mutaciones del contrato para detectar cambios incompatibles del vendedor.
 *
 * @method TestDetectsBreakingSupplierChanges
 * @param {*testing.T} t - Contexto de prueba.
 * @returns {void} Falla si un cambio inválido pasa como oferta correcta.
 */
func TestDetectsBreakingSupplierChanges(t *testing.T) {
	for _, body := range []string{
		strings.Replace(validQuote, `"potenciaPanelW":450`, `"potenciaNominalW":450`, 1),
		strings.Replace(validQuote, `"cantidad":2`, `"cantidad":"2"`, 1),
		strings.Replace(validQuote, `"totalCentavos":50000000`, `"totalCentavos":50000001`, 1),
		strings.Replace(validQuote, `"moneda":"ARS"`, `"moneda":"USD"`, 1),
		strings.Replace(validQuote, `"sku":"panel",`, "", 1),
		validQuote + `{}`, "null", `<html>cold start</html>`,
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/cotizaciones" || r.Method != "POST" || r.Header.Get("Authorization") != "Bearer "+testToken {
				t.Error("llamada fuera del contrato")
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(body))
		}))
		client, _ := NewClient(server.URL, testToken, time.Second)
		_, err := client.Quote(context.Background(), 494)
		server.Close()
		if !errors.Is(err, domain.ErrContract) {
			t.Fatalf("se aceptó %s: %v", body, err)
		}
	}
}

/**
 * Verifica apertura del circuito, recuperación, timeout y aislamiento de errores.
 *
 * @method TestFailureProtection
 * @param {*testing.T} t - Contexto de prueba.
 * @returns {void} Falla si una dependencia lenta o caída propaga llamadas sin límite.
 */
func TestFailureProtection(t *testing.T) {
	var calls atomic.Int32
	var status atomic.Int32
	status.Store(503)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(int(status.Load()))
		_, _ = w.Write([]byte(validQuote))
	}))
	defer server.Close()
	client, _ := NewClient(server.URL, testToken, time.Second)
	for i := 0; i < 4; i++ {
		if _, err := client.Quote(context.Background(), 494); !errors.Is(err, domain.ErrUnavailable) {
			t.Fatal(err)
		}
	}
	if calls.Load() != 3 {
		t.Fatalf("circuito: %d llamadas", calls.Load())
	}
	client.mu.Lock()
	client.openUntil = time.Now().Add(-time.Second)
	client.mu.Unlock()
	status.Store(200)
	if _, err := client.Quote(context.Background(), 494); err != nil {
		t.Fatal("recuperación", err)
	}
	status.Store(409)
	if _, err := client.Quote(context.Background(), 494); !errors.Is(err, domain.ErrStock) {
		t.Fatal("stock", err)
	}
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(150 * time.Millisecond):
		}
	}))
	defer slow.Close()
	bounded, _ := NewClient(slow.URL, testToken, 30*time.Millisecond)
	start := time.Now()
	if _, err := bounded.Quote(context.Background(), 494); !errors.Is(err, domain.ErrUnavailable) || time.Since(start) > time.Second {
		t.Fatal("timeout", err)
	}
}

/**
 * Evita que una redirección reenvíe credenciales a otro destino.
 *
 * @method TestRejectsRedirect
 * @param {*testing.T} t - Contexto de prueba.
 * @returns {void} Falla si se sigue la redirección del vendedor.
 */
func TestRejectsRedirect(t *testing.T) {
	var hits atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1) }))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 307) }))
	defer redirect.Close()
	client, _ := NewClient(redirect.URL, testToken, time.Second)
	if _, err := client.Quote(context.Background(), 494); !errors.Is(err, domain.ErrContract) || hits.Load() != 0 {
		t.Fatal("redirección", err)
	}
}

/**
 * Verifica que solo una goroutine prueba la recuperación del vendedor.
 *
 * @method TestHalfOpenAllowsOneConcurrentProbe
 * @param {*testing.T} t - Contexto de prueba con vendedor HTTP bloqueado.
 * @returns {void} Falla si la recuperación abre varias llamadas simultáneas.
 */
func TestHalfOpenAllowsOneConcurrentProbe(t *testing.T) {
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		select {
		case entered <- struct{}{}:
		default:
		}
		<-release
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(validQuote))
	}))
	defer server.Close()
	defer unblock()
	client, _ := NewClient(server.URL, testToken, 3*time.Second)
	client.openUntil = time.Now().Add(-time.Second)
	client.failures = 3
	done := make(chan error, 1)
	go func() { _, err := client.Quote(context.Background(), 494); done <- err }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("la prueba de recuperación no llegó al vendedor")
	}
	for i := 0; i < 8; i++ {
		if _, err := client.Quote(context.Background(), 494); !errors.Is(err, domain.ErrUnavailable) {
			t.Fatal("se permitió otra prueba", err)
		}
	}
	if calls.Load() != 1 {
		t.Fatal("más de una llamada en recuperación")
	}
	unblock()
	if err := <-done; err != nil {
		t.Fatal("recuperación fallida", err)
	}
	client.mu.Lock()
	defer client.mu.Unlock()
	if !client.openUntil.IsZero() || client.failures != 0 || client.probe {
		t.Fatal("el circuito no volvió a cerrado")
	}
}
