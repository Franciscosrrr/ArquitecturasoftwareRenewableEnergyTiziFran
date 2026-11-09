# Inicialización del catálogo

El esquema principal está embebido en [repository/schema.sql](../internal/repository/schema.sql); el repositorio aplica semillas idempotentes sin sobrescribir fichas. [features/catalog.go](../internal/features/catalog.go) crea catalogo_outbox repetiblemente. El índice y la caché son derivados/reconstruibles, no migraciones de la fuente MySQL. No se mantiene un segundo esquema en esta carpeta.
