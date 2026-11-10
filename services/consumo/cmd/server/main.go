package main

import (
	"context"
	"log"
	transport "renewable.local/consumo/internal/http"
	"renewable.local/consumo/internal/integration"
	"renewable.local/consumo/internal/private"
	"renewable.local/platform"
	"time"
)

/**
 * Configura dependencias del proceso y ejecuta el servidor con apagado controlado.
 *
 * @method main
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func main() {
	shutdown := platform.Init("consumo")
	defer shutdown()
	internal := platform.Secret("INTERNAL_TOKEN")
	catalog := integration.NewCatalog(platform.Env("CATALOG_URL", "http://electrodomesticos:8080"), internal)
	cfg := transport.Config{APIKey: platform.Secret("M2M_API_KEY"), DeniedKey: platform.Secret("M2M_DENIED_KEY"), Catalog: catalog, InternalToken: internal}
	if platform.Env("CAPABILITY_ONLY", "false") != "true" {
		db, err := platform.OpenSQL()
		if err != nil {
			log.Fatal("Configuración MySQL de Consumo inválida")
		}
		defer db.Close()
		store := &private.Store{DB: db}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err = store.Migrate(ctx); err != nil {
			cancel()
			log.Fatal("No se pudo inicializar persistencia Consumo")
		}
		cancel()
		cfg.Private = &private.Service{Store: store, Catalog: catalog}
		go platform.RetryLoop(context.Background(), store.Relay)
	}
	platform.Serve(transport.NewHandler(platform.Secret("HEALTHCHECK_TOKEN"), cfg))
}
