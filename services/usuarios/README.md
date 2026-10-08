# Estado ejecutable de entrega 1

Módulo Go independiente con cmd/server, paquetes internos y Dockerfile. Solo liveness funciona; readiness responde 503 y negocio 501. El transporte inicial usa net/http sin dependencias externas; Gin y los drivers se incorporarán al implementar los casos reales. No es evidencia de persistencia ni del patrón completo en funcionamiento.

Dentro de esta carpeta: `go test ./...` y `go run ./cmd/server`. Con Docker, usar el perfil `estructura` de la raíz.

## Diseño previsto

# usuarios

Estado del negocio: diseño pendiente de implementación; el arranque mínimo ejecutable está descrito arriba.

## Responsabilidad

Registro, login, perfil, roles y sesiones.

## Organización

En capas: Handler → servicio de identidad → repositorio.

## Dependencias

MySQL usuarios_db; hashing, JWT y claves.

Ver [estructura general](../../docs/STRUCTURE.md). Las dependencias y el código se construirán dentro de Docker. No se declara un módulo, una imagen ni una API ya funcionando.
