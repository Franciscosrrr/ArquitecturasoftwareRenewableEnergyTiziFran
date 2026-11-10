package main

import (
	"context"
	"log"
	"renewable.local/platform"
	transport "renewable.local/usuarios/internal/http"
	"renewable.local/usuarios/internal/identity"
	"time"
)

/**
 * Configura dependencias del proceso y ejecuta el servidor con apagado controlado.
 *
 * @method main
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func main() {
	shutdown := platform.Init("usuarios")
	defer shutdown()
	db, err := platform.OpenSQL()
	if err != nil {
		log.Fatal("Configuración MySQL de Usuarios inválida")
	}
	defer db.Close()
	store := &identity.Store{DB: db, Secret: platform.Secret("JWT_SECRET")}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if store.Migrate(ctx) != nil || store.Bootstrap(ctx, platform.Env("ADMIN_EMAIL", "admin@renewable.local"), platform.Secret("ADMIN_PASSWORD")) != nil {
		log.Fatal("No se pudo inicializar identidad")
	}
	platform.Serve(transport.NewHandler(platform.Secret("HEALTHCHECK_TOKEN"), transport.Config{Store: store, InternalToken: platform.Secret("INTERNAL_TOKEN"), SecureCookie: platform.Env("COOKIE_SECURE", "false") == "true"}))
}
