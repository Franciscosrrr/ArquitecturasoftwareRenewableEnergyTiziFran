package main

import (
	"context"
	"log"
	"renewable.local/platform"
	"renewable.local/solar/internal/adapters/external"
	transport "renewable.local/solar/internal/adapters/http"
	"renewable.local/solar/internal/adapters/persistence"
	"renewable.local/solar/internal/application"
	"renewable.local/solar/internal/worker"
	"strings"
	"time"
)

/**
 * Configura dependencias del proceso y ejecuta el servidor con apagado controlado.
 *
 * @method main
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func main() {
	shutdown := platform.Init("solar")
	defer shutdown()
	mode := platform.Env("PANEL_PROVIDER_MODE", "simulado")
	base := platform.Env("PANEL_PROVIDER_URL", "http://proveedor-paneles-sim:8080")
	if mode != "simulado" && mode != "externo" {
		log.Fatal("Modo proveedor inválido")
	}
	if mode == "externo" && !strings.HasPrefix(base, "https://") && !(base == "http://20.226.84.78" && platform.Env("PANEL_PROVIDER_ALLOW_HTTP", "false") == "true") {
		log.Fatal("El proveedor externo requiere HTTPS o la excepción académica explícita para su IP acordada")
	}
	estimator, err := external.NewClient(platform.Env("CONSUMO_URL", "http://consumo:8080"), "", 2500*time.Millisecond)
	if err != nil {
		log.Fatal("CONSUMO_URL inválida")
	}
	supplier, err := external.NewClient(platform.Env("PANEL_SIMULATOR_URL", "http://proveedor-paneles-sim:8080"), platform.Secret("PANEL_PROVIDER_API_KEY"), time.Second)
	if err != nil {
		log.Fatal("URL comercial inválida")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	store, err := persistence.Open(ctx)
	cancel()
	if err != nil {
		log.Fatal("No se pudo inicializar MongoDB Solar")
	}
	defer store.Client.Disconnect(context.Background())
	planner := &application.ResilientSupplier{Primary: supplier, Fallback: supplier, Catalog: store, External: mode == "externo"}
	if mode == "externo" {
		key := platform.Secret("PANEL_COMPARATOR_API_KEY")
		if key == "" {
			log.Fatal("Falta la credencial privada del Comparador de Proveedores")
		}
		client, err := external.NewClient(base, key, time.Second)
		if err != nil {
			log.Fatal("URL comercial inválida")
		}
		planner.Primary = &external.Comparator{Client: client, Catalog: store}
	}
	consumer := worker.Consumer{Store: store, Supplier: planner}
	go consumer.Run(context.Background())
	service := application.Service{Estimator: estimator, Supplier: planner, Mode: mode}
	platform.Serve(transport.NewFullHandler(platform.Secret("HEALTHCHECK_TOKEN"), service, store, platform.Secret("INTERNAL_TOKEN")))
}
