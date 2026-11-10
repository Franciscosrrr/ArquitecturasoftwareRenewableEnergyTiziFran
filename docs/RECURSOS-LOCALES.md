# Recursos del entorno local

El equipo tiene 16 GB. Durante la primera compilación se observó alrededor de 95 % de RAM del host y disco saturado: coexistían el entorno anterior y el final, sin límites de contenedores, más compilaciones y caché de archivos de WSL. Ese estado no se considera adecuado para uso sostenido.

Se detuvo el proyecto Docker anterior conservando volúmenes y `.env`, se redujeron buffers de MySQL (64 MiB, performance_schema desactivado, máximo 30 conexiones), se limitó WiredTiger a 256 MiB y la indexación a un hilo/128 Mb. Compose aplica memoria y CPU a cada contenedor; Valkey limita sus datos a 64 MiB con LFU. Los límites no sustituyen mediciones y no describen capacidad de producción.

La primera comprobación después del ajuste mostró aproximadamente 1.2 GiB sumados entre contenedores y 3.7 GiB libres de RAM Windows (uso 76.3%). WSL también conserva memoria del kernel y caché, así que la suma de `docker stats` no equivale a VmmemWSL. El pico de construir imágenes se mide por separado del reposo. Las bases conservan journaling/commit durable; no se desactivó fsync para reducir consumo.

Los scripts fijan compilación secuencial por defecto y Go compila con 2 hilos y presupuesto blando de 512 MiB. El build de Compose debe respetar COMPOSE_BAKE=false para no delegar un lote paralelo al builder. Las pruebas se ejecutan después de compilar, no simultáneamente con otra compilación completa. No se inicia un segundo proyecto salvo para pruebas aisladas y se detiene al finalizar.

Para iniciar: `scripts/compose.ps1 up -d --build` (PowerShell) o `sh scripts/compose.sh up -d --build`. Para liberar servicios al terminar: mismo script con `stop`; conserva los datos. `down` conserva volúmenes, pero `down -v` los elimina y no forma parte del procedimiento. Para inspeccionar: `docker stats --no-stream`. No se cambió la configuración global de Ubuntu/WSL ni se borraron cachés de imágenes o volúmenes.

Referencias: [límites Compose](https://docs.docker.com/reference/compose-file/services/), [paralelismo Compose](https://docs.docker.com/compose/how-tos/environment-variables/envvars/), [caché WiredTiger](https://www.mongodb.com/docs/manual/core/wiredtiger/), [opciones Meilisearch](https://www.meilisearch.com/docs/learn/self_hosted/configure_meilisearch_at_launch).


### Ajuste de compilación y verificación 2026-11-09

El detector de carreras agotó el límite de 768 MiB al compilar el codec MsgPack incluido por defecto en Gin. Se usa el build tag oficial `nomsgpack` tanto en compilación como en tests: las APIs de este proyecto solo exponen JSON/HTML y no utilizan binding/rendering MsgPack. Así se elimina trabajo de una funcionalidad opcional sin ampliar la memoria del contenedor. El formato JSON, los contratos y su serializador permanecen iguales. Referencia: [documentación oficial Gin](https://github.com/gin-gonic/gin/blob/master/docs/doc.md#build-without-msgpack-rendering-feature). No se considera un test aprobado hasta repetirlo con la misma configuración de build.


La verificación completa también excedió 768 MiB al compilar el SDK de caché. Se establece un presupuesto de 1536 MiB solo para el contenedor temporal verificar-go, con GOMEMLIMIT 384 MiB, GOGC 50 y un único paquete a la vez. Los servicios conservan sus límites de ejecución. La suite estándar incluye todos los módulos; el detector de carreras se aplica a los estados concurrentes propios: limitador/plataforma, balanceador y clientes HTTP con circuitos. No se instrumenta con race la totalidad de drivers de infraestructura. La caché de compilación de verificación evita recompilar dependencias en cada intento. El presupuesto del compilador no describe consumo en reposo ni capacidad de producción.


## Muestra final y elección del equipo

Con los compiladores y Chrome de prueba finalizados, quedó un único proyecto con 18 contenedores: 1326.5 MiB sumados en docker stats, 2.41 GiB libres de RAM Windows y 84.6%de uso total del host. La muestra de disco del mismo archivo se interpreta como un instante, no un promedio sostenido. [Datos y límites reales](evidence/final-2026-11-09/recursos.json).

Se liberó caché de archivos de Linux después de compilar; no se eliminaron bases, imágenes ni cachés de compilación en disco. El usuario eligió mantener únicamente los límites del proyecto. `.wslconfig` no existe y no se aplicó un máximo global de 5 GB ni un reinicio global de WSL. El límite de verificación de 1536 MiB solo se usa al ejecutar esa prueba temporal y termina al salir el contenedor; no queda un compilador activo. Para trabajar en otra cosa, `scripts/compose.ps1 stop` apaga el proyecto y conserva datos.

## Corrección posterior del broker

El presupuesto de RabbitMQ se ajustó de 256 a 384 MiB al observar una terminación OOM durante el diagnóstico. El healthcheck HTTP autenticado reemplaza la CLI que arrancaba otra VM Erlang cada 5 s; ahora consulta disponibilidad y alarmas cada 10 s. El umbral preventivo absoluto es 192 MiB, con dos schedulers para el servidor y uno para CLI manual. La identidad persistente del nodo se conserva antes de recrear. Los relays detectan canales cerrados aun sin eventos y reconectan antes de la próxima alta. La muestra de recursos anterior a este ajuste se conserva como antecedente; la final se registra nuevamente después de las pruebas. No se modifica WSL global.

## Medición posterior a la corrección del broker

1436.6 MiB sumados en docker stats, Windows con 2.39 GiB libres y 84.7% de memoria usada. Un solo proyecto con 18 contenedores y sin verificadores activos. La medición anterior se conserva en recursos-previo-broker.json; la vigente está en [recursos.json](evidence/final-2026-11-09/recursos.json). El host incluye caché/VM de WSL y otras aplicaciones; no se aplicó configuración global.


## Actualización 2026-11-09 — retención de trazas

La revisión de los contenedores detectó Jaeger detenido por OOM tras horas de retención ilimitada en memoria. Se mantiene su límite de 128 MiB, con un máximo de 512 trazas, cola de 256 spans, cuatro workers y presupuesto Go de 96 MiB. No se modifica WSL. La ventana es temporal y descarta trazas antiguas; la auditoría de negocio sigue en SQL/Mongo. Véanse los anexos añadidos a ADR-011/ADR-013 y la evidencia de esta revisión de experiencia de cliente.


## Limpieza operativa y comisión A1 — 2026-11-09

El usuario confirmó la entrega del ZIP Render a Clínica y dejó su prueba de backend a cargo de ese grupo. Se registró comisión A1 y se eliminó el pendiente de número de grupo: el enunciado revisado no lo solicita.

Por solicitud explícita se detuvieron los dos contenedores de renewable-clinica-publica y se retiraron los 18 contenedores detenidos de renewable-energy-final-dev junto con su red vacía. Se conservaron doce volúmenes del entorno antiguo, doce del vigente y todas las imágenes. Los dieciocho contenedores de renewable-energy-entrega2 y los dos de renewable-web-publica se mantuvieron activos con los mismos IDs e instantes de arranque. Los proyectos ajenos no cambiaron y no se modificó WSL globalmente.

[Inventario previo](evidence/operacion-local-2026-11-09/docker-antes.json), [resultado Docker](evidence/operacion-local-2026-11-09/docker-despues.json), [continuidad HTTP](evidence/operacion-local-2026-11-09/continuidad.json) e [identificación](evidence/operacion-local-2026-11-09/identificacion.json). Las mediciones anteriores son históricas, no una muestra de consumo después de esta limpieza.
