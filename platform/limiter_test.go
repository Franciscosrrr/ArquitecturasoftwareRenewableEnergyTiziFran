package platform

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

/**
 * Verifica la cuota sostenida de sesenta por minuto y la ráfaga de diez.
 *
 * @method TestM2MRateAndBurst
 * @param {*testing.T} t - Contexto de prueba sin esperas reales.
 * @returns {void} Falla si la política de tráfico acordada cambia.
 */
func TestM2MRateAndBurst(t *testing.T) {
	limiter := NewLimiter()
	now := time.Now()
	limiter.now = func() time.Time { return now }
	for i := 0; i < 10; i++ {
		if !limiter.Allow("clinica") {
			t.Fatal("ráfaga")
		}
	}
	if limiter.Allow("clinica") {
		t.Fatal("undécima solicitud")
	}
	if !limiter.Allow("otro-cliente") {
		t.Fatal("aislamiento")
	}
	for i := 0; i < 60; i++ {
		now = now.Add(time.Second)
		if !limiter.Allow("clinica") || limiter.Allow("clinica") {
			t.Fatal("reposicion: una solicitud por segundo")
		}
	}
	now = now.Add(time.Minute)
	for i := 0; i < 10; i++ {
		if !limiter.Allow("clinica") {
			t.Fatal("recuperación de ráfaga")
		}
	}
	if limiter.Allow("clinica") {
		t.Fatal("capacidad máxima")
	}
}

/**
 * Comprueba que solicitudes concurrentes no exceden la ráfaga de una identidad.
 *
 * @method TestConcurrentLimiterPreservesBurst
 * @param {*testing.T} t - Contexto de prueba de concurrencia.
 * @returns {void} Falla si los accesos simultáneos duplican tokens.
 */
func TestConcurrentLimiterPreservesBurst(t *testing.T) {
	limiter := NewLimiter()
	fixed := time.Now()
	limiter.now = func() time.Time { return fixed }
	var allowed atomic.Int32
	var workers sync.WaitGroup
	for i := 0; i < 100; i++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			if limiter.Allow("clinica") {
				allowed.Add(1)
			}
		}()
	}
	workers.Wait()
	if allowed.Load() != 10 {
		t.Fatalf("ráfaga concurrente: %d, esperadas 10", allowed.Load())
	}
}
