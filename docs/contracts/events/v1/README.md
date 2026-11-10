# Eventos de dominio 1.0.0

[AsyncAPI 3.0](asyncapi.json), [ConsumoEvaluado JSON Schema](ConsumoEvaluado.schema.json) y [ElectrodomesticoActualizado JSON Schema](ElectrodomesticoActualizado.schema.json).

AMQP 0.9.1 en RabbitMQ: exchange durable direct `renewable.events`. Routing `consumo.evaluado.v1` -> cola durable `solar.estudios`; `catalogo.actualizado.v1` -> `catalogo.indice`. Exchange direct permite añadir otra cola/suscriptor con la misma routing key (fan-out explícito por bindings) y evita difundir ambos tipos a todos los workers. El emisor deriva propietario y snapshot del servidor, nunca de un actor enviado por el navegador. Auth AMQP por secreto, red interna, sin puerto publicado.

ConsumoEvaluado contiene eventId, estudioId, evaluacionId, propietarioId, snapshot congelado (consumo/fichas/entrada/lugar), solar (HSP manual, factorRendimiento, coberturaObjetivo, superficieM2 opcional/null), tipo, traceparent y requestId. ElectrodomesticoActualizado contiene eventId, tipo, id y version. No se envían contraseñas ni JWT/API keys. Un evento de catálogo provoca lectura del estado canónico actual, por lo que una entrega fuera de orden no restaura una versión vieja.

Publicación persistente obligatoria + publisher confirms; outbox en transacción local y sent_at solo tras confirmación. Entrega al menos una vez; un crash entre confirmación y sent_at puede duplicar. Solar conserva eventId único, hash, lease y documento por estudio, y ACK solo después del resultado confirmado con journaling de Mongo. El mismo evento completado no vuelve a cotizar. Conflicto de contenido, esquema desconocido o parámetros inválidos: mensaje no procesable.

Prefetch 1 por worker; dos instancias consumen competitivamente. Trabajo con presupuesto 12 s; proveedor HTTP 1 s sin retry interno. Fallas transitorias se republican con confirms a `renewable.retry` y cola `.retry`, vencimientos 1/5/30 s y dead-letter a principal; después de tres reintentos se envía a `renewable.dlx` y `.dlq`. Mensajes incompatibles van directo a DLQ. Headers `attempt` y `reason` son técnicos. La expiración por mensaje en una cola compartida puede retrasarse detrás de otro mensaje; las demoras no son un SLA exacto.

El ACK del original ocurre después de confirmar retry/DLQ. Si falla esa publicación: NACK con requeue y reconexión con backoff 1..15 s. La outbox reintenta con lease de 10 s; no se eliminan evaluaciones cuando el broker cae. Un lease Solar de 30 s protege procesamiento simultáneo; si cae el worker se recupera por entrega/reintento. No hay exactly-once ni transacción entre MySQL, broker y Mongo. Perder los volúmenes o corrupción de disco requiere backup externo.

Reproceso: administrador autenticado consulta las colas y republica hasta 50 mensajes compatibles. Los inválidos se dejan sin ACK y retornan a DLQ al cerrar el canal; así no bloquean revisar los válidos posteriores. Conservar la causa y corregir/versionar el productor antes de modificar un mensaje manualmente.
