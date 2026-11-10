# Servicio Consumo

Estado 2026-11-09. Go/Gin con reglas de aplicación separadas del transporte. POST /api/v1/estimaciones-consumo conserva OpenAPI M2M 1.0.1 y cálculo decimal exacto; las rutas públicas admiten parámetros externos o resolución de fichas vigentes por HTTP interno. El flujo invitado no escribe lugares, evaluaciones ni eventos.

El modo completo usa MySQL consumo_db para lugares/configuraciones propios, versiones, snapshots, evaluaciones, idempotencia y outbox. Confirmar bloquea configuración/lugar y comprueba sus versiones en una transacción; inserta evaluación e intención de publicación juntas. El relay publica con confirms y conserva trabajo al caer RabbitMQ. Solar procesa el evento de manera eventual.

Archivar/restaurar cambia solo estado. Eliminar marca eliminado_at, impide reabrir/confirmar el guardado y conserva el historial. Un lugar ocupado exige mover o eliminar sus consumos, incluidos archivados. Solo el propietario accede a sus recursos. Timeouts/circuitos protegen al catálogo; no hay transacción distribuida con su lectura.

CAPABILITY_ONLY=true conserva la capacidad externa sin inicializar SQL ni broker. Esta es la unidad cloud preparada, independiente del sistema completo. Las credenciales se suministran por entorno, sin valores reales en Git ni navegador. [Contrato M2M](../../docs/contracts/README.md), [API privada](../../docs/contracts/private-v1/README.md), [eventos](../../docs/contracts/events/v1/README.md).
