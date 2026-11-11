# Índice de documentación vigente

Revisión: 11/11/2026. Los documentos siguientes describen el código actual. Los archivos y secciones de entregas anteriores se conservan como antecedentes fechados; un pendiente antiguo no sustituye el estado vigente.

| Necesidad | Documento |
|---|---|
| Dominio, integrantes, arranque y accesos | [README principal](../README.md) |
| Funcionalidades, reglas y aceptación | [SPEC](../SPEC.md) |
| Servicios, estilos internos, datos y limitaciones | [Arquitectura](ARCHITECTURE.md) |
| Contexto, contenedores, evaluación y cloud | [Diagramas: fuentes y PNG](diagrams/README.md) |
| D1–D13: contexto, decisión, alternativas y porqué | [Índice ADR](adr/README.md) |
| Configuración, credenciales y puesta en marcha | [Configuración](CONFIGURACION.md) |
| Responsabilidades de carpetas y paquetes | [Estructura](STRUCTURE.md) |
| Capacidad que consume Clínica | [Contrato M2M y ejemplos](contracts/README.md) |
| Cotización consumida del otro grupo | [Comparador de Proveedores 1.1.0](contracts/comparador-proveedores-v1.1.0/README.md) |
| Cotización, comparación y respaldo solar | [Integración Solar](contracts/SOLAR.md) |
| Cuenta, permisos, guardados e historial | [API privada 1.2.0](contracts/private-v1/README.md) |
| Mensajería y esquema de eventos | [Contratos de eventos](contracts/events/v1/README.md) |
| Uso sin cuenta | [Modo invitado](GUEST-MODE.md) |
| Archivo, restauración, eliminación y búsqueda | [Experiencia de cliente](EXPERIENCIA-CLIENTE.md) |
| Transacción crítica, versiones e idempotencia | [Consistencia](CONSISTENCY-REVIEW.md) |
| Logs, métricas, trazas, tablero y alertas | [Observabilidad](OBSERVABILITY.md) |
| Pruebas ejecutadas y límites de su evidencia | [Verificación vigente](VERIFICACION-FINAL.md) |
| Carga y efecto de la caché | [Resultados y análisis](CARGA-FINAL.md) |
| Caídas, impacto, causas y recuperación | [Postmortem](POSTMORTEM.md) |
| Recursos y decisiones de capacidad/costo | [Recursos locales](RECURSOS-LOCALES.md), D13 |
| API M2M publicada, cold start y continuidad | [Render](DESPLIEGUE-GRATUITO.md) |
| Web HTTPS desde la PC y URL temporal | [Publicación web](PUBLICACION-WEB.md) |
| Material de las nueve clases y diferencias justificadas | [Relación con los prácticos](CLASES-PRACTICAS.md) |
| Formato de comentarios y alcance de su revisión | [Documentación de código](DOCUMENTACION-CODIGO.md) |
| Resultado de esta auditoría | [Revisión documental](REVISION-DOCUMENTACION-2026-11-11.md) |
| Rama principal, hash y ficha Word | [Cierre de entrega](ENTREGA.md) |

## Contratos y versiones

| Capacidad | Versión vigente | Formato |
|---|---|---|
| Estimación M2M de Clínica | 1.0.1, congelada | [OpenAPI](contracts/v1/openapi.json) |
| Lectura y resolución de catálogo de equipos | 1.1.0 | [OpenAPI](contracts/catalog-v1/openapi.json) |
| Consumo público y comparación temporal | 1.2.0 | [OpenAPI](contracts/guest-v1/openapi.json) |
| Cotización solar pública | 1.1.0 | [OpenAPI](contracts/solar-v1/openapi.json) |
| Búsqueda de paneles | 1.1.0 | [OpenAPI](contracts/panel-search-v1/openapi.json) |
| Cuenta y operaciones privadas | 1.2.0 | [OpenAPI](contracts/private-v1/openapi.json) |
| Comparador de Proveedores consumido | 1.1.0 | [OpenAPI original recibido](contracts/comparador-proveedores-v1.1.0/openapi.yaml) |
| Eventos internos | 1.0.0 | [AsyncAPI y esquemas](contracts/events/v1/README.md) |
| Simuladores comerciales conservados | 0.1.0-simulated / 0.2.0-simulated | [v1](contracts/paneles-sim-v1/openapi.json), [v2](contracts/paneles-sim-v2/openapi.json) |

## Antecedentes

DELIVERY-1, DELIVERY-2, REVIEW, SECURITY-AUDIT, las propuestas de diseño y ARCHITECTURE-ENTREGA-1 conservan decisiones y verificaciones de sus fechas. Las evidencias de 08/09 de noviembre no se presentan como ensayos del proveedor real del 11/11. PUBLICACION-CLINICA conserva el historial del túnel M2M local retirado; el endpoint operativo de Clínica se documenta en Render. Los ADR mantienen sus anexos y se actualizan dentro del mismo archivo, sin duplicar su número.
