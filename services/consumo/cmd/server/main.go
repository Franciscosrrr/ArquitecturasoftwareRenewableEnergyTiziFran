package main

import (
	transport "renewable.local/consumo/internal/http"
	"renewable.local/consumo/internal/integration"
	"renewable.local/platform"
)

func main() {
	shutdown := platform.Init("consumo")
	defer shutdown()
	catalog := integration.NewCatalog(platform.Env("CATALOG_URL", "http://electrodomesticos:8080"), platform.Secret("INTERNAL_TOKEN"))
	platform.Serve(transport.NewHandler(platform.Secret("HEALTHCHECK_TOKEN"), transport.Config{APIKey: platform.Secret("M2M_API_KEY"), DeniedKey: platform.Secret("M2M_DENIED_KEY"), Catalog: catalog}))
}
