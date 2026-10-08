# Estructura inicial y dependencias

## Estructura incluida

README y SPEC en raíz; docs contiene arquitectura, diagramas, ADR y contratos. Cada servicio y gateway tiene go.mod, cmd/server/main.go, paquetes internal y Dockerfile. go.work reúne los cinco módulos sin compartir modelos de dominio. mock contiene servidor local y pruebas HTTP; compose.yaml define mock, verificación y el perfil estructura.

Los paquetes internos contienen puntos de extensión documentados; el arranque y los handlers de estado compilan. Las capas y los puertos de negocio no están implementados. El frontend conserva su README: React pertenece a los hitos posteriores.

## Organización prevista al implementar

| Componente | Organización interna prevista |
|---|---|
| Servicios en capas | cmd para inicio; internal/http para handlers; application para casos/reglas; repository e integration para acceso a datos y clientes; migrations y pruebas |
| Solar hexagonal | cmd para ensamblado; domain para reglas; application para casos; ports para contratos propios; adapters/http, messaging, persistence y external; worker y pruebas |
| Gateway | Rutas, JWT/API key, proxy, selector de instancias, health checks, límites y correlación |
| Frontend | Vistas de identidad, lugares, Mis consumos, equipos, historial, paneles, estudios y administración |
| Infra | Compose, Dockerfiles, inicialización, redes, volúmenes, observabilidad y perfil de integración |

Se agregan archivos Go, Dockerfiles y Compose para el hito inicial. Cada servicio tiene módulo y build propios; sus migraciones reales siguen pendientes. El monorepo no obliga a desplegarlos juntos.

## Dependencias externas previstas

Go/Gin para HTTP, GORM con driver MySQL en repositorios relacionales, driver oficial MongoDB en Solar, cliente RabbitMQ, clientes Meilisearch/Valkey, biblioteca JWT, Argon2id, gobreaker y OpenTelemetry. React/JavaScript para la web. go test, Testcontainers para Go y k6 para verificación.

La estructura usa Go 1.26 y solo biblioteca estándar; su Dockerfile fija Go 1.26.1. El mock usa Python 3.12.15 en Docker, sin paquetes pip. Gin, GORM y demás bibliotecas son dependencias previstas, todavía no importadas. Al incorporarlas se fijarán versiones y go.sum. Los tags de las imágenes no son digests inmutables; fijar digests tras verificar el primer build. Se usarán ediciones gratuitas autohospedadas y ninguna dependencia SaaS paga.

## Dependencias internas y externas

Consumo llama a Electrodomésticos al agregar equipos. Solar consume ConsumoEvaluado y consulta evaluaciones cuando genera variantes. Electrodomésticos/Solar consultan Usuarios para mutaciones administrativas. Los servicios verifican JWT con claves públicas, sin leer usuarios_db. Solar integrará al proveedor asignado mediante su adaptador, si la capacidad corresponde a su negocio; otra capacidad puede obligar a revisar el servicio responsable.

Los worker/outbox/indexador son procesos del servicio dueño. Un contrato compartido no compartirá repositorios, entidades con tags ORM ni funciones privadas de negocio entre servicios. El algoritmo público y el de evaluaciones se reutiliza dentro de Consumo, evitando dos fórmulas divergentes.

## Modo invitado

Los handlers de consumo temporal llaman al cálculo sin repositorio ni outbox. Solar incorpora el caso de uso síncrono `RecomendarTemporal` con puertos de lectura de paneles/recurso solar, separado del caso durable que persiste estudios. Ambos comparten reglas del dominio. Las dependencias se ensamblan por ruta para que las rutas invitadas no dependan de la disponibilidad de identidad/guardados/broker. No se agrega un quinto microservicio. Ver [detalle](GUEST-MODE.md).
