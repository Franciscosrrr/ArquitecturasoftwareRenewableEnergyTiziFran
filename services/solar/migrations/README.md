# Inicialización de Solar

[persistence/mongo.go](../internal/adapters/persistence/mongo.go) configura índices de estudios, unicidad de evento, semillas de paneles y la marca pendiente de indexación para documentos antiguos. [panel_index.go](../internal/adapters/persistence/panel_index.go) coordina las tareas derivadas con lease/versión. No hay scripts SQL para Mongo ni un segundo inicializador pendiente en esta carpeta. Las semillas no sustituyen el catálogo del proveedor real.
