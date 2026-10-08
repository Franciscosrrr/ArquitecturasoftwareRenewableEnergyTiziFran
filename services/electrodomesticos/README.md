# Estado ejecutable de entrega 1

Módulo Go independiente con cmd/server, paquetes internos y Dockerfile. Solo liveness funciona; readiness responde 503 y negocio 501. El transporte inicial usa net/http sin dependencias externas; Gin y los drivers se incorporarán al implementar los casos reales. No es evidencia de persistencia ni del patrón completo en funcionamiento.

Dentro de esta carpeta: `go test ./...` y `go run ./cmd/server`. Con Docker, usar el perfil `estructura` de la raíz.

## Diseño previsto

# electrodomesticos

Estado del negocio: diseño pendiente de implementación; el arranque mínimo ejecutable está descrito arriba.

## Responsabilidad

Catálogo precargado, altas administrativas, búsqueda y proyección del índice.

## Organización

En capas: Handler → servicio de catálogo → repositorios/integraciones.

## Dependencias

MySQL electrodomesticos_db, Meilisearch, Valkey, RabbitMQ; Usuarios para rol vigente.

Ver [estructura general](../../docs/STRUCTURE.md). Las dependencias y el código se construirán dentro de Docker. No se declara un módulo, una imagen ni una API ya funcionando.

## Lectura anónima

Las consultas paginadas de fichas activas y sus detalles son públicas y pasan por gateway con límites. Datos administrativos, auditoría y todas las mutaciones requieren los permisos establecidos. Elegir una ficha para una simulación no modifica el catálogo.
