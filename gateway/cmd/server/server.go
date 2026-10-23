package main

import (
	transport "renewable.local/gateway/internal/http"
	"renewable.local/platform"
)

func main() {
	shutdown := platform.Init("gateway")
	defer shutdown()
	platform.Serve(transport.NewHandler(platform.Secret("HEALTHCHECK_TOKEN"), transport.Config{ConsumoURL: platform.Env("CONSUMO_URL", "http://consumo:8080"), CatalogURL: platform.Env("CATALOG_URL", "http://electrodomesticos:8080"), WebDir: platform.Env("WEB_DIR", "/web"), DocsDir: platform.Env("DOCS_DIR", "/contracts")}))
}
