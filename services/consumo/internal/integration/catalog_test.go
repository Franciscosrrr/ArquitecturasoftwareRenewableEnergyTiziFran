package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

/**
 * Comprueba el comportamiento y los invariantes indicados por CircuitOpensAndRecovers.
 *
 * @method TestCircuitOpensAndRecovers
 * @param {*testing.T} t - Contexto de verificación de la prueba.
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func TestCircuitOpensAndRecovers(t *testing.T) {
	var calls atomic.Int32
	healthy := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if !healthy {
			w.WriteHeader(503)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`[{"id":"a","modo":"POTENCIA","potenciaW":10,"energiaPorCicloKWh":0,"factorFuncionamiento":1}]`))
	}))
	defer srv.Close()
	c := NewCatalog(srv.URL, "test")
	for range 4 {
		if _, err := c.Resolve(context.Background(), []string{"a"}); err == nil {
			t.Fatal("unavailable accepted")
		}
	}
	if calls.Load() != 3 {
		t.Fatal("circuit did not stop fourth request")
	}
	c.openUntil = time.Now().Add(-time.Second)
	healthy = true
	if _, err := c.Resolve(context.Background(), []string{"a"}); err != nil {
		t.Fatal(err)
	}
	if !c.openUntil.IsZero() || calls.Load() != 4 {
		t.Fatal("circuit did not recover")
	}
}

/**
 * Comprueba el comportamiento y los invariantes indicados por CancellationStopsCall.
 *
 * @method TestCancellationStopsCall
 * @param {*testing.T} t - Contexto de verificación de la prueba.
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func TestCancellationStopsCall(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer srv.Close()
	c := NewCatalog(srv.URL, "test")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := c.Resolve(ctx, []string{"a"}); err == nil {
		t.Fatal("cancelled request succeeded")
	}
}
