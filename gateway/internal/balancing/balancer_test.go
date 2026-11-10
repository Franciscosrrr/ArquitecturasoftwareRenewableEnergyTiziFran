package balancing

import (
	"context"
	"net/http"
	"net/http/httptest"
	"renewable.local/platform"
	"sync/atomic"
	"testing"
)

/**
 * Comprueba el comportamiento y los invariantes indicados por HealthRetirementRecoveryAndDistribution.
 *
 * @method TestHealthRetirementRecoveryAndDistribution
 * @param {*testing.T} t - Contexto de verificación de la prueba.
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func TestHealthRetirementRecoveryAndDistribution(t *testing.T) {
	var down atomic.Bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token" {
			w.WriteHeader(401)
			return
		}
		if down.Load() {
			w.WriteHeader(503)
		}
	})
	one := httptest.NewServer(handler)
	defer one.Close()
	two := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer two.Close()
	b := &Balancer{targets: []target{{URL: one.URL}, {URL: two.URL}}, token: "token", client: platform.Client(1000000000)}
	ctx := context.Background()
	b.Check(ctx)
	a, _ := b.Next()
	z, _ := b.Next()
	if a == z || !b.Ready() {
		t.Fatal("no distribuye")
	}
	down.Store(true)
	b.Check(ctx)
	if !b.targets[0].Healthy {
		t.Fatal("retiró antes del umbral")
	}
	b.Check(ctx)
	for i := 0; i < 10; i++ {
		url, err := b.Next()
		if err != nil || url != two.URL {
			t.Fatal("dirige a réplica caída")
		}
	}
	down.Store(false)
	b.Check(ctx)
	if b.targets[0].Healthy {
		t.Fatal("reintegró antes del umbral")
	}
	b.Check(ctx)
	if !b.targets[0].Healthy {
		t.Fatal("no reintegró")
	}
}
