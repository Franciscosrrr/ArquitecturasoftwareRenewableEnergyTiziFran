package main

import (
	transport "renewable.local/gateway/internal/http"
	"renewable.local/platform"
)

/**
 * Configura dependencias del proceso y ejecuta el servidor con apagado controlado.
 *
 * @method main
 * @returns {void} Completa la operación o señala el fallo por el mecanismo del adaptador.
 */
func main() {
	shutdown := platform.Init("gateway")
	defer shutdown()
	platform.Serve(transport.NewHandler(platform.Secret("HEALTHCHECK_TOKEN"), transport.Config{ConsumoURL: platform.Env("CONSUMO_URL", "http://consumo:8080"), CatalogURL: platform.Env("CATALOG_URL", "http://electrodomesticos:8080"), SolarURLs: platform.Env("SOLAR_URLS", "http://solar:8080,http://solar-2:8080"), IdentityURL: platform.Env("IDENTITY_URL", "http://usuarios:8080"), InternalToken: platform.Secret("INTERNAL_TOKEN"), WebDir: platform.Env("WEB_DIR", "/web"), DocsDir: platform.Env("DOCS_DIR", "/contracts"), CapabilityOnly: platform.Env("CAPABILITY_ONLY", "false") == "true"}))
}
