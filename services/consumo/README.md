# Estado ejecutable de entrega 1

Módulo Go independiente con cmd/server, paquetes internos y Dockerfile. Solo liveness funciona; readiness responde 503 y negocio 501. El transporte inicial usa net/http sin dependencias externas; Gin y los drivers se incorporarán al implementar los casos reales. No es evidencia de persistencia ni del patrón completo en funcionamiento.

Dentro de esta carpeta: `go test ./...` y `go run ./cmd/server`. Con Docker, usar el perfil `estructura` de la raíz.

## Diseño previsto

# consumo

Estado del negocio: diseño pendiente de implementación; el arranque mínimo ejecutable está descrito arriba.

## Responsabilidad

Lugares, configuraciones privadas editables, snapshots, evaluaciones, outbox y capacidad pública.

## Organización

En capas: Handler → servicio de consumo/reglas → repositorios/integraciones.

## Dependencias

MySQL consumo_db, Electrodomésticos HTTP y RabbitMQ.

Ver [estructura general](../../docs/STRUCTURE.md). Las dependencias y el código se construirán dentro de Docker. No se declara un módulo, una imagen ni una API ya funcionando.

## Cálculo invitado

POST /api/v1/publico/consumo calcula entradas temporales sin sesión y sin consultar/escribir consumo_db ni generar outbox/eventos. Comparte reglas con evaluación autenticada y con la capacidad M2M; sus handlers y autorizaciones son distintos. Guardados e historial siguen requiriendo JWT y propietario. Ver [modo invitado](../../docs/GUEST-MODE.md).
