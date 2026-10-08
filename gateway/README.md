# Estado ejecutable de entrega 1

Módulo Go independiente con cmd/server, paquetes internos y Dockerfile. Solo liveness funciona; readiness responde 503 y negocio 501. El transporte inicial usa net/http sin dependencias externas; Gin y los drivers se incorporarán al implementar los casos reales. No es evidencia de persistencia ni del patrón completo en funcionamiento.

Dentro de esta carpeta: `go test ./...` y `go run ./cmd/server`. Con Docker, usar el perfil `estructura` de la raíz.

## Diseño previsto

# gateway

Estado del negocio: diseño pendiente de implementación; el arranque mínimo ejecutable está descrito arriba.

## Responsabilidad

Entrada única, rutas anónimas delimitadas, JWT para guardados/perfil/administración, API key para M2M, límites y balanceo Solar con salud.

## Organización

Componente de infraestructura en Go; sin reglas energéticas ni BD de negocio.

## Dependencias

Claves públicas Usuarios, destinos HTTP y DNS de Compose.

Ver [estructura general](../docs/STRUCTURE.md). Las dependencias y el código se construirán dentro de Docker. No se declara un módulo, una imagen ni una API ya funcionando.

## Política de acceso

Sin sesión: interfaz, registro/login, consulta de catálogos activos, POST /api/v1/publico/consumo y POST /api/v1/publico/recomendaciones-solares. Solo GET/POST de lectura o cálculo permitidos expresamente; no se usa un comodín que publique rutas administrativas. Privadas: lugares, guardados, evaluación durable, historial, perfil y administración. La ruta /api/v1/estimaciones-consumo mantiene X-API-Key para otro grupo. Los límites anónimos usan contadores técnicos temporales; no guardan consumo. Ver [modo invitado](../docs/GUEST-MODE.md).
