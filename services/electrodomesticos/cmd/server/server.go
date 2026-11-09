package main

import (
	"context"
	"log/slog"
	"os"
	"renewable.local/electrodomesticos/internal/features"
	transport "renewable.local/electrodomesticos/internal/http"
	"renewable.local/electrodomesticos/internal/repository"
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
	shutdown := platform.Init("electrodomesticos")
	defer shutdown()
	store, err := repository.Open(platform.Env("MYSQL_USER", "catalogo"), platform.Secret("MYSQL_PASSWORD"), platform.Env("MYSQL_ADDR", "mysql:3306"), platform.Env("MYSQL_DATABASE", "electrodomesticos_db"))
	if err != nil {
		slog.Error("Configuración MySQL inválida")
		os.Exit(1)
	}
	defer store.DB.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err = store.Migrate(ctx); err != nil {
		slog.Error("No se pudo inicializar el catálogo")
		os.Exit(1)
	}
	runtime := features.New(store)
	if err = runtime.Init(ctx); err != nil {
		slog.Error("No se pudo inicializar outbox")
		os.Exit(1)
	}
	runtime.Start(context.Background())
	platform.Serve(transport.NewHandler(platform.Secret("HEALTHCHECK_TOKEN"), transport.Config{Store: store, Features: runtime, InternalToken: platform.Secret("INTERNAL_TOKEN")}))
}
