# Electrodomésticos - Entrega 2

Catálogo público de lectura en MySQL 8.4. Listado/búsqueda literal, filtros categoría/modo, paginación, orden y detalle de fichas activas. Endpoint interno de resolución por lote autenticado con Bearer técnico, usado por Consumo. Sin mantenimiento administrativo público.

schema.sql embebido define la tabla e índices; la inicialización crea seis referencias didácticas mediante INSERT IGNORE transaccional, sin duplicar ni sobrescribir filas existentes. Cada ficha informa fuente y versión. Solo este servicio recibe las credenciales de su base. Las migraciones incrementales posteriores y el índice Meilisearch permanecen pendientes.

Prueba con base real: perfil verificar-go de Compose. Dockerfile desde raíz mediante infra/backend.Dockerfile y ARG SERVICE=services/electrodomesticos.
