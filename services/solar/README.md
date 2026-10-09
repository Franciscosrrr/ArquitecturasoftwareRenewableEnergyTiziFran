# Estado ejecutable de entrega 1

Módulo Go independiente con cmd/server, paquetes internos y Dockerfile. Liveness y readiness exigen Authorization: Bearer con HEALTHCHECK_TOKEN (mínimo 32 caracteres, sin valor por defecto). Con credencial válida, liveness devuelve 200 y readiness 503; negocio responde 501. Sin credencial operativa válida, las rutas /health/ devuelven 401 sin informar estado. Ver scripts/compose para generar la credencial local automáticamente. El transporte inicial usa net/http sin dependencias externas; Gin y los drivers se incorporarán al implementar los casos reales. No es evidencia de persistencia ni del patrón completo en funcionamiento.

Dentro de esta carpeta: `go test ./...` y `go run ./cmd/server`. Con Docker, usar el perfil `estructura` de la raíz.

## Diseño previsto

# solar

Estado del negocio: diseño pendiente de implementación; el arranque mínimo ejecutable está descrito arriba.

## Responsabilidad

Paneles administrativos y estudios/recomendaciones asociados a evaluaciones.

## Organización

Hexagonal: dominio/casos de uso y puertos; adaptadores Gin, MongoDB, mensajería y HTTP externo.

## Dependencias

MongoDB solar_db, RabbitMQ, Consumo HTTP, Usuarios y proveedor externo.

Ver [estructura general](../../docs/STRUCTURE.md). Las dependencias y el código se construirán dentro de Docker. No se declara un módulo, una imagen ni una API ya funcionando.

## Recomendación invitada

POST /api/v1/publico/recomendaciones-solares recibe consumo y parámetros temporales; lee paneles activos/recurso solar y devuelve resultado síncrono. No crea documentos de estudios, trabajos, leases, auditoría de negocio ni eventos. Un puerto de lectura de paneles y el dominio se comparten con el worker; la persistencia del estudio se usa solo en el flujo autenticado. Se balancea entre ambas instancias. Ver [modo invitado](../../docs/GUEST-MODE.md).
