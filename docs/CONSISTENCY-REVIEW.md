# Consistencia de la confirmación

La operación con efectos es confirmar una configuración propia. Filtra propietario, bloquea configuración y lugar por versión, congela snapshot e inserta evaluación, idempotencia y outbox en la misma transacción. Unicidad(config_id,config_version,place_version) y clave(owner_id,clave). Repetir entrada/clave devuelve la evaluación; cambiar entrada con la misma clave409. Cambiar la superficie exige otra versión del lugar. Variantes solares se conservan como estudios distintos y tienen idempotencia propia por evaluación/clave.

Las llamadas de catálogo ocurren antes de los locks y no participan del commit SQL; el snapshot corresponde a esa lectura versionada. No se prometen locks distribuidos sobre el catálogo. Guardar y actualizar recalcula explícitamente con fichas vigentes; una evaluación previa nunca cambia.

Publicar y confirmar broker no es atómico con SQL. Outbox conserva pendientes y puede duplicar después de un crash entre publisher confirm y sent_at. Solar deduplica de forma durable por eventId/estudio, conserva hash y lease, y ACK después del resultado Mongo con journaling. Errores/leases/reintentos no crean otro estudio para el mismo evento. Persistencia y deduplicación son locales; no se afirma exactly-once ni HA de discos.

[Pruebas](evidence/final-2026-11-09/README.md): confirmación concurrente, clave/cuerpo distintos, versiones obsoletas, cambios de lugar, broker detenido, duplicados y reinicio de servicios. [D4](adr/ADR-004.md#revision-final-20261109).
