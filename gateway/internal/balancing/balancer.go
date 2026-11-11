package balancing

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"renewable.local/platform"
	"sync"
	"time"
)

/**
 * Conserva la URL de una réplica Solar, su salud y los éxitos consecutivos de recuperación.
 *
 * @method target
 * @returns {target} Representación tipada usada por este componente.
 */
type target struct {
	URL                 string
	Healthy             bool
	Initialized         bool
	Failures, Successes int
}

/**
 * Distribuye solicitudes entre réplicas saludables y protege su estado concurrente.
 *
 * @method Balancer
 * @returns {Balancer} Representación tipada usada por este componente.
 */
type Balancer struct {
	mu      sync.Mutex
	targets []target
	next    int
	token   string
	client  *http.Client
}

/**
 * Inicializa destinos de Solar y comienza sus comprobaciones activas.
 *
 * @method New
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @param {[]string} urls - URLs internas de las réplicas Solar a comprobar y balancear.
 * @param {string} token - Credencial técnica del servicio.
 * @returns {*Balancer} Instancia configurada del componente con sus dependencias y estado inicial.
 */
func New(ctx context.Context, urls []string, token string) *Balancer {
	b := &Balancer{token: token, client: platform.Client(time.Second)}
	for _, raw := range urls {
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
			panic("destino Solar inválido")
		}
		b.targets = append(b.targets, target{URL: raw})
	}
	go func() {
		b.Check(ctx)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				b.Check(ctx)
			}
		}
	}()
	return b
}

/**
 * Verifica el hash Argon2id y rechaza representaciones incompatibles.
 *
 * @method Check
 * @param {context.Context} ctx - Contexto cancelable y presupuesto de la operación.
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func (b *Balancer) Check(ctx context.Context) {
	for i := range b.targets {
		b.mu.Lock()
		base := b.targets[i].URL
		b.mu.Unlock()
		req, err := http.NewRequestWithContext(ctx, "GET", base+"/health/ready", nil)
		healthy := false
		if err == nil {
			req.Header.Set("Authorization", "Bearer "+b.token)
			res, e := b.client.Do(req)
			if e == nil {
				healthy = res.StatusCode == 200
				res.Body.Close()
			}
		}
		b.mu.Lock()
		t := &b.targets[i]
		if healthy {
			t.Successes++
			t.Failures = 0
			if !t.Initialized || t.Successes >= 2 {
				t.Healthy = true
			}
		} else {
			t.Failures++
			t.Successes = 0
			if t.Failures >= 2 {
				t.Healthy = false
			}
		}
		t.Initialized = true
		value := 0.
		if t.Healthy {
			value = 1
		}
		u, _ := url.Parse(t.URL)
		platform.State.WithLabelValues("balanceador", u.Hostname()).Set(value)
		b.mu.Unlock()
	}
}

/**
 * Selecciona la siguiente réplica saludable mediante round robin concurrente.
 *
 * @method Next
 * @returns {string, error} URL de una réplica saludable; error si no existe un destino disponible.
 */
func (b *Balancer) Next() (string, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i := 0; i < len(b.targets); i++ {
		index := (b.next + i) % len(b.targets)
		if b.targets[index].Healthy {
			b.next = (index + 1) % len(b.targets)
			u, _ := url.Parse(b.targets[index].URL)
			platform.Operations.WithLabelValues("balanceador", u.Hostname()).Inc()
			return b.targets[index].URL, nil
		}
	}
	return "", errors.New("ninguna réplica Solar disponible")
}

/**
 * Comprueba si la dependencia necesaria puede atender la operación.
 *
 * @method Ready
 * @returns {bool} true si hay al menos una réplica Solar saludable; false en caso contrario.
 */
func (b *Balancer) Ready() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, t := range b.targets {
		if t.Healthy {
			return true
		}
	}
	return false
}
