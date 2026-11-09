# Ensayo final — broker, réplica y dependencias,2026-11-09

Entorno: proyecto renewable-energy-entrega2 del repositorio original, con datos persistentes y límites aplicados. Prueba reversible completada; [16 observaciones](evidence/final-2026-11-09/fallas.json) y [ejecución](evidence/final-2026-11-09/fallas-ejecucion.txt). Los ensayos anteriores se conservan después de esta sección como historia.

## Impacto y línea temporal

| Segundos desde inicio | Observación |
|---:|---|
|2.875|broker detenido|
|3.14|evaluación confirmada sin broker|
|3.484|broker reiniciado|
|9.812|outbox recuperada y estudio completado|
|28.062|broker recreado con identidad y volumen conservados|
|29.109|alta de catálogo tras reconexión ociosa visible en menos de 5 s|
|33.094|evento duplicado entregado|
|36.437|evento incompatible en DLQ|
|38.187|reproceso administrado completa mensaje válido y conserva inválido|
|42.828|réplica Solar 2 detenida|
|69.078|balanceo excluye caída y alerta activa|
|88.656|réplica recuperada|
|92.578|vendedor caído conserva evaluación y encola reintento|
|97.0|reintento comercial completado|
|99.469|buscador y caché caídos: respaldo SQL|
|115.265|reinicios conservan SQL, sesión y Mongo|

RabbitMQ detenido: confirmar devolvió 201 con publicación PENDIENTE y no perdió evaluación. Al iniciar broker, la outbox se publicó y el estudio se completó. Reentregar el mismo evento conservó un único estudio. Un evento incompatible terminó en DLQ; el reproceso administrado conservó el inválido y completó uno compatible.

Una réplica Solar detenida se retiró del reparto: cuatro respuestas posteriores vinieron de Solar 1, mientras Prometheus mostraba RenewableTargetDown firing para solar-2. Después del reinicio y dos checks exitosos volvieron ambas instancias. El vendedor detenido produjo trabajo recuperable; la evaluación siguió200 y el estudio terminó después de restaurarlo. Caché y buscador caídos juntos devolvieron200 desde sql-degradado. Reiniciar Usuarios/Consumo/Solar conservó sesión, configuración y documento Mongo.

## Detección, causa y respuesta

Se observaron estado durable/publicación, respuestas, cabecera de réplica, contadores, alerta de Prometheus y trazas/logs. La causa fue la parada deliberada de dependencias únicas y de una de dos réplicas. El gateway mantiene la capacidad de enrutar a una réplica sana; outbox, confirms y deduplicación desacoplan el commit SQL del procesamiento.

Durante el desarrollo del ensayo se encontró que el SDK de caché interpretaba MaxRetries=0 como su valor por defecto y mantenía intentos de conexión adicionales. Eso retrasaba el respaldo hasta el timeout del gateway. Se corrigió a MaxRetries=-1, un intento de dial y contexto total de 300 ms para caché, conservando el contexto original para buscador/SQL. El ensayo final volvió a detener ambas dependencias y comprobó el200 de respaldo. También se separó X-Gateway-Instance-ID de X-Instance-ID para que la observación identifique la réplica real.

## Correcciones detectadas al repetir la verificación

El primer ensayo HTTP posterior mostró un alta ausente durante más de 8 s; la outbox confirmó el envío unos 9 s después del commit. El relay no observaba el cierre de un canal AMQP mientras estaba inactivo y conservaba el backoff acumulado de una caída anterior. Ambos relays ahora observan NotifyClose en espera y la reconexión vuelve a 1 s después de una ejecución estable de al menos 15 s. La prueba ampliada recrea el broker, comprueba que su DLQ persistente conserva mensajes y exige visibilidad del alta en menos de 5 s después de reconectar.

Durante el diagnóstico se observó OOMKilled con el límite de RabbitMQ de 256 MiB, al compartir recursos con CLI Erlang. Se ajustó a 384 MiB, con watermark preventivo absoluto de 192 MiB y schedulers acotados. La sonda periódica ahora usa HTTP autenticado sin iniciar otra VM Erlang. Los scripts conservan en .env el hostname anterior para mantener la identidad y ruta de datos del nodo al recrear; las instalaciones nuevas usan un nombre estable.

Una segunda repetición encontró dos resultados por la búsqueda parcial predeterminada de Meilisearch: el identificador separado por guion compartía un token con una ficha residual del primer fallo. Se usa un identificador único sin separador y limpieza registrada con addCleanup; no se cambió el comportamiento del buscador para satisfacer la prueba. Las dos fichas propias residuales se desactivaron. La suite final de 17 HTTP y los módulos Go/race pasaron. Ambos intentos previos se conservan en la evidencia.

## Qué se mantiene como límite

Gateway, bases, broker y motor tienen una instancia local; el ensayo no acredita HA integral ni recuperación de pérdida de disco. Retry1/5/30 s es una política acotada, con posible demora por cola; una DLQ incompatible requiere corregir/versionar el productor. La fuente de irradiación y vendedor reales siguen pendientes. Antes de uso productivo hacen falta copias/restauración, capacidad sostenida y política operativa del equipo. La evidencia demuestra las fallas concretas ensayadas.

## Consumo de recursos detectado por el equipo

Durante builds coexistían dos proyectos y compilación paralela; Windows llegó a alrededor de 95%de memoria, frente al 93%de la captura del usuario, y disco saturado. Se detuvo el duplicado, limitaron contenedores/buffers, construyó por servicio y separó el compilador temporal de la ejecución normal. El usuario eligió no cambiar WSL global. La muestra final y los presupuestos figuran en [RECURSOS-LOCALES](RECURSOS-LOCALES.md).

---

## Ensayos históricos conservados

# Postmortem - caída controlada de MySQL

Ensayo local realizado el 22 de octubre de 2026, zona America/Buenos_Aires (UTC-03:00). Solo se detuvo MySQL del proyecto Compose renewable-energy-entrega2. No se borraron volúmenes ni se afectaron otros proyectos.

## Impacto

La simulación por fichas dejó de obtener el catálogo y respondió 503 controlado. La API M2M con parámetros externos siguió funcionando: HTTP 200 en 6.253 ms. No hubo guardados anónimos ni se sustituyeron fichas por valores inventados.

## Línea temporal

| Hora local | Hecho observado |
|---|---|
| 21:03:50 | baseline |
| 21:03:52 | mysql_stopped |
| 21:04:02 | catalog_unavailable |
| 21:04:02 | m2 m_remains_available |
| 21:04:03 | mysql_start_requested |
| 21:04:13 | catalog_recovered |
| 21:04:13 | catalog_restart_requested |
| 21:04:14 | catalog_restart_idempotent |

## Detección y respuesta

Cuatro solicitudes por catálogo devolvieron 503. Duraciones: 2013.956 ms, 2014.714 ms, 1761.045 ms, 24.369 ms. Las tres primeras alcanzaron la dependencia; el circuito evitó otra consulta en la cuarta, reduciendo su duración. Readiness de Electrodomésticos refleja la pérdida de MySQL y el gateway deja de estar listo para el flujo completo, pero conserva el direccionamiento de rutas que pueden responder.

Se solicitó iniciar MySQL y el catálogo se recuperó en 10.10 s desde esa acción, incluyendo espera de recuperación y la ventana del circuito. Un reinicio posterior de Electrodomésticos verificó que las semillas conservan 6 fichas frente a 6 iniciales. No fue necesario reiniciar Consumo ni gateway ni modificar datos.

## Causa

Falla deliberada de una dependencia indispensable del catálogo: una única instancia MySQL. El cálculo M2M no depende de ese catálogo, por lo que el aislamiento por ruta mantuvo su disponibilidad.

## Qué funcionó y qué falta

Timeout HTTP, circuit breaker, respuesta recuperable, separación entre proceso vivo/readiness y semillas idempotentes. Faltan métricas/alertas sobre el circuito y MySQL, una política de copias/restauración y observación sostenida de recuperación. El ensayo no demuestra alta disponibilidad ni tolerancia a caída entre réplicas.

Reproducir voluntariamente con Python disponible: `python tests/failure_probe.py`, desde la raíz con el sistema iniciado. El script detiene temporalmente la base de este proyecto y la vuelve a iniciar incluso si falla una comprobación del período de caída. [Datos originales](evidence/entrega2/fallo-controlado.json).

## Ensayo adicional - vendedor de paneles simulado - 2026-11-08

Se detuvo únicamente proveedor-paneles-sim del proyecto. No se borraron volúmenes ni se detuvo MySQL, Consumo, gateway o Solar. El script restauró el vendedor en un bloque finally. Es una prueba local real de dependencia simulada, no una caída del grupo proveedor externo ni evidencia de alta disponibilidad.

| Evento | Hora America/Buenos_Aires | HTTP | Duración ms |
|---|---|---:|---:|
| baseline_quote | 2026-11-08 T22:41:39.075832-03:00 | 200 | 85.55 |
| vendor_stopped_1 | 2026-11-08 T22:41:39.925061-03:00 | 503 | 525.234 |
| vendor_stopped_2 | 2026-11-08 T22:41:40.431126-03:00 | 503 | 505.807 |
| vendor_stopped_3 | 2026-11-08 T22:41:40.938462-03:00 | 503 | 507.41 |
| vendor_stopped_4 | 2026-11-08 T22:41:40.942074-03:00 | 503 | 3.269 |
| m2 m_isolated | 2026-11-08 T22:41:40.944166-03:00 | 200 | 2.11 |
| vendor_recovery | 2026-11-08 T22:41:51.364120-03:00 | 200 | 22.901 |

Impacto: cuatro cotizaciones devolvieron 503 controlado. La API M2M respondió 200 y conservó 47.4 kWh, aislada del vendedor. Tras reiniciar y transcurrir la ventana del circuito, la cotización volvió a 200. El navegador también tiene e2 e que mantiene visible el consumo ante 503 y elimina una oferta anterior; ese error se inyecta en el e2 e, mientras este ensayo detiene un proceso real.

Detección: estado HTTP, código PROVEEDOR_NO_DISPONIBLE, duración y traceId en los resultados/logs. Causa: parada deliberada del único vendedor de prueba. El circuito reduce llamadas durante indisponibilidad; no crea stock ni un éxito ficticio. Falta instrumentación de métricas/alerta y el mismo ensayo contra el proveedor real acordado.

Evidencia: [fallo-vendedor.json](evidence/integraciones-2026-11-08/fallo-vendedor.json). El ensayo histórico MySQL de Entrega 2 anterior se conserva separado.
