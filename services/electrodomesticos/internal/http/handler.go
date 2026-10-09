// Package transport contiene el adaptador HTTP inicial. No implementa negocio.
package transport

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"net/http"
)

// NewHandler recibe una credencial operativa independiente de las sesiones de usuario.
func NewHandler(healthcheckToken string) http.Handler {
	health := http.NewServeMux()
	health.HandleFunc("GET /health/live", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"estado": "proceso-iniciado"})
	})
	health.HandleFunc("GET /health/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"estado": "negocio-no-implementado"})
	})

	expected := sha256.Sum256([]byte("Bearer " + healthcheckToken))
	mux := http.NewServeMux()
	mux.Handle("/health/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		supplied := sha256.Sum256([]byte(r.Header.Get("Authorization")))
		if len(healthcheckToken) < 32 || subtle.ConstantTimeCompare(supplied[:], expected[:]) != 1 {
			w.Header().Set("WWW-Authenticate", `Bearer realm="healthcheck"`)
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"mensaje": "No autenticado"})
			return
		}
		health.ServeHTTP(w, r)
	}))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		json.NewEncoder(w).Encode(map[string]string{"mensaje": "Esqueleto de entrega 1; rutas de negocio pendientes"})
	})
	return mux
}
