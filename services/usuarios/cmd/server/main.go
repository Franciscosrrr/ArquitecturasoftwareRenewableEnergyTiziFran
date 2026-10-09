package main

import (
	"log"
	"net/http"
	"os"
	transport "renewable.local/usuarios/internal/http"
	"time"
)

func main() {
	healthcheckToken := os.Getenv("HEALTHCHECK_TOKEN")
	if len(healthcheckToken) < 32 {
		log.Fatal("HEALTHCHECK_TOKEN debe tener al menos 32 caracteres; usar scripts/compose para el arranque local")
	}
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	server := &http.Server{Addr: addr, Handler: transport.NewHandler(healthcheckToken), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second}
	log.Fatal(server.ListenAndServe())
}
