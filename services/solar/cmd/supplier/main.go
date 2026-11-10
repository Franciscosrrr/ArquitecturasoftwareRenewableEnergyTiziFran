package main

import (
	"renewable.local/platform"
	"renewable.local/solar/internal/adapters/simulator"
)

/**
 * Arranca el simulador comercial sin crear pedidos ni reservar stock.
 *
 * @method main
 * @returns {void} Mantiene el simulador activo hasta apagar el proceso.
 */
func main() {
	shutdown := platform.Init("proveedor-paneles-sim")
	defer shutdown()
	platform.Serve(simulator.NewHandler(platform.Secret("HEALTHCHECK_TOKEN"), platform.Secret("PANEL_PROVIDER_API_KEY")))
}
