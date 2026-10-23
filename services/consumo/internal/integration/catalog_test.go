package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

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
