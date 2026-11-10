# Servicio Electrodomésticos

Estado 2026-11-09. MySQL privado administra fichas de equipos para construir consumos. La API pública mantiene listado, filtros, orden, paginación y detalle de fichas activas; el resolver interno por lote requiere credencial técnica y lo usa Consumo. La administración de fichas exige rol autorizado y versión esperada.

schema.sql embebido crea tabla/índices; las seis semillas didácticas son idempotentes y no sobrescriben fichas existentes. Las modificaciones administrativas generan outbox local. RabbitMQ notifica cambios y el worker lee el estado canónico, espera la indexación Meilisearch y luego invalida la generación de caché Valkey. Hay reindexación temporal/swap y respaldo SQL cuando fallan dependencias derivadas.

La búsqueda principal de la interfaz se trasladó a paneles; esta API y su índice se conservan para compatibilidad. Solo este servicio recibe las credenciales de su base. [Contrato](../../docs/contracts/catalog-v1/openapi.json), [medición de caché](../../docs/CARGA-FINAL.md), [fallas](../../docs/POSTMORTEM.md). Construcción desde la raíz mediante infra/backend.Dockerfile.
