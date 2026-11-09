# Arquitectura implementada

Revisión 2026-11-09. El entorno completo es local, con cuatro microservicios de negocio, un gateway, dos instancias Solar y dependencias reales. [Contenedores](diagrams/contenedores-actuales.png), [contexto](diagrams/contexto.png), [flujo durable](diagrams/evaluacion-durable.png) y [unidad M2M cloud desplegada](diagrams/cloud-gratuito.png).

## Responsabilidades y datos

| Componente | Responsabilidad | Datos propios |
|---|---|---|
| Gateway Go | React, contratos, validación de sesión, rutas, cuotas, balanceo | Buckets/estado de health en memoria; ninguna base de negocio |
| Usuarios | Registro Argon2 id, login/JWT, refresh rotativo, logout, roles | MySQL usuarios_db: usuarios y sesiones |
| Consumo, capas | Reglas decimales, lugares/configuraciones, confirmación consistente | MySQL consumo_db: snapshots, versiones, idempotencia, outbox |
| Electrodomésticos, capas | Fichas de equipos para el selector, API/índice compatible, invalidación y worker | MySQL electrodomesticos_db, Meilisearch derivado y Valkey descartable |
| Solar, hexagonal | Búsqueda de paneles, comparación de alternativas, cotización comercial y trabajo asíncrono | MongoDB solar_db: paneles, pendientes de índice y estudios deduplicados |
| Vendedor simulado | Oferta, stock y precio por potencia/SKU | Referencias didácticas; no compra ni reserva |

Cada base tiene un usuario de aplicación diferente y volumen persistente. Ningún servicio consulta tablas/colecciones de otro. Gateway conoce destinos HTTP y consulta Usuarios para la sesión/rol vigente, limpia cabeceras de identidad aportadas por el cliente y transmite el actor validado con secreto interno. Ser administrador no habilita datos privados de otros propietarios. Los accesos internos y las métricas tienen credenciales técnicas independientes de API keys M2M y tokens de usuario.

## Dos estilos y dependencias

Consumo usa capas de transporte, aplicación/cálculo, resolver por HTTP y repositorio SQL; no mezcla las fórmulas con Gin o SQL. Solar mantiene dominio puro (racionales, comparación por factibilidad/costo/superficie), aplicación con puertos Estimator/Supplier/ModelSupplier y adaptadores HTTP, proveedor, Mongo y AMQP. El worker coordina deduplicación/persistencia y delega la selección en RunStudy; el dominio no importa drivers. No se utiliza el gateway como bus interno para integrar al vendedor.

## Flujos

El invitado obtiene fichas activas, estima consumo, compara paneles con HSP manuales y puede cotizar contra el vendedor. No crea lugares, evaluaciones, estudios ni mensajes de negocio; el borrador vive en memoria del navegador. Al autenticarse se conserva ese borrador hasta Guardar explícitamente. Guardar recalcula en servidor y conserva las fichas/versiones leídas en un único lote.

Confirmar una configuración requiere versión de configuración/lugar e Idempotency-Key. MySQL bloquea y comprueba ambos registros propios, congela el snapshot e inserta evaluación, registro de idempotencia y outbox juntos. Repetir la misma entrada recupera la evaluación; una entrada distinta con la misma clave devuelve 409. Un catálogo modificado exige actualizar explícitamente la configuración. El bloqueo es local a Consumo; no existe una transacción distribuida con la lectura del catálogo.

El relay publica ConsumoEvaluado.v1 con confirms. Solar recibe con prefetch1, persiste claim/evento único, calcula y consulta al vendedor por SKU. Falta de stock cambia la alternativa; incompatibilidad se rechaza. Mongo confirma journaling antes del ACK. El usuario consulta PENDIENTE, EN_PROCESAMIENTO, COMPLETADO, SIN_ALTERNATIVA o ERROR. Cambiar cobertura crea otro estudio sobre la evaluación inmutable. Confirmación y estudio no son una única transacción: la consistencia entre servicios es eventual y explícita.

Un cambio administrativo de equipo se confirma con su outbox. Rabbit entrega ElectrodomesticoActualizado.v1 al worker del catálogo, que lee el estado canónico más reciente y espera la tarea Meilisearch antes de invalidar la generación de caché. No depende de que un usuario busque. Objetivo saludable de atraso≤5 s, medido en integración. Texto, categoría, modo, potencia, páginas y orden se resuelven en el índice. Reindexación usa temporal/swap. Si índice o caché fallan, hay respaldo SQL etiquetado; la validación de fichas siempre evita caché.

La búsqueda visible para el cliente es de paneles: texto, potencia, superficie por unidad, moneda/precio, orden y páginas. Solar usa otro índice Meilisearch y Mongo canónico; la marca pendiente se escribe junto a la ficha, con leases entre réplicas y limpieza condicionada a versión. Una reconciliación cada minuto permite reconstruir el índice perdido; ante caída se filtra sobre Mongo. El cálculo comercial valida las fichas canónicas y no confía en el índice eventualmente desactualizado. El API/índice previo de equipos se conserva para compatibilidad y el selector de equipos continúa disponible.

Archivar/restaurar lugares o configuraciones cambia solo estado con control de propiedad/versión, sin nueva versión de contenido. Eliminar marca eliminado_at y excluye el guardado de edición/confirmación, conservando evaluaciones y estudios. Un lugar con consumos guardados, incluidos archivados, exige moverlos o eliminarlos; no hay cascada silenciosa. La migración agrega columnas sin borrar filas.

## Distribución y fallas

Gateway distribuye round-robin Solar 1/2. Checks cada 5 s con timeout1 s retiran tras dos fallas y reintegran tras dos éxitos; no se reintenta automáticamente un POST en otra réplica. Ambas instancias comparten Mongo y cola, con deduplicación durable. No es HA integral: gateway, bases, broker y motor tienen una instancia local.

HTTP: catálogo 2 s con circuito3 fallas/10 s/una prueba; cotización vendedor 1 s, aplicación pública Solar 4 s, gateway 8 s y M2M 4.5 s. Worker 12 s con reintentos durables1/5/30 s antes deDLQ; mensajes incompatibles van directo. Confirmar consumo no espera al broker. Al caer el vendedor se conserva el consumo y un estudio recuperable; no se inventa precio, stock ni irradiación. [Postmortem](POSTMORTEM.md).

## Operación y observabilidad

Compose limita memoria/CPU, buffers de bases e indexación; scripts generan secretos una sola vez y compilan secuencialmente. Solo gateway8080, Jaeger16686, Prometheus9090 y Grafana3000 se publican en loopback. /health y /metrics requieren HEALTHCHECK_TOKEN. SQL, Mongo, Rabbit, caché, índice y Loki no exponen puerto al host.

Prometheus mide rutas normalizadas, latencia, errores, caché, outbox, workers y distribución. Grafana se provisiona con dashboard y fuentes Prometheus/Loki/Jaeger. Loki agrega logs JSON mediante buffer no bloqueante y retiene 72 h. Traceparent HTTP/evento conecta confirmación y worker; requestId se conserva como correlación. No se registran cuerpos privados ni credenciales. Jaeger usa memoria y pierde trazas al reiniciar; retiene hasta 512 trazas, con cola de 256 spans y cuatro workers dentro de 128 MiB. El presupuesto Go es 96 MiB. Se verificó la retención tras detectar y corregir un OOM por crecimiento ilimitado. Las alertas son de Prometheus; no se configura envío externo de notificaciones.

## Límites externos

Contrato M2M 1.0.1 de Clínica permanece intacto, sin precio. Render Free aloja esa capacidad Go sin estado; no aloja las bases durables de la aplicación completa. La capacidad ya se publicó por HTTPS mediante un túnel desde la PC, con claves dedicadas entregadas al usuario por separado. El proceso corre en https://renewable-consumo-v1.onrender.com, verificado por HTTPS y contrato; los túneles de la PC son accesos distintos. La comprobación desde el backend de Clínica queda a cargo de ese grupo por indicación del usuario. Su cold start puede superar1.2 s/5 s. El vendedor sigue simulado por autorización del equipo, pendiente de contrato real; eso no acredita la integración entre grupos que exige el enunciado. [Despliegue](DESPLIEGUE-GRATUITO.md).

Las decisiones y comparaciones con todos los prácticos se conservan en [D1–D13](adr/README.md) y [matriz de clases](CLASES-PRACTICAS.md). Los resultados ejecutados se distinguen de estos límites en [verificación](VERIFICACION-FINAL.md).

## Publicación posterior mediante Cloudflare — 2026-11-09

El usuario autorizó un origen en su PC. La publicación M2M usa Gateway/Consumo mediante un proyecto aislado de dos contenedores y un Quick Tunnel temporal: https://emails-bingo-preference-packets.trycloudflare.com. No utiliza las bases ni la red del sistema completo. Render es la publicación cloud principal, con claves propias; el túnel se conserva como alternativa local. El contrato M2M no cambia; no se presenta el túnel como hosting de Workers ni como dirección permanente. [Operación y evidencia](PUBLICACION-CLINICA.md).


## Entrada HTTPS de la web completa — 2026-11-09

La web también está publicada, con autorización explícita del usuario: https://conditional-wan-server-learning.trycloudflare.com. Otro proyecto de dos contenedores usa Nginx de 32 MiB y cloudflared de 64 MiB. Nginx marca la cookie de refresh como Secure, preserva Host/Origin y bloquea rutas técnicas; reutiliza el gateway y sus servicios/bases, sin modificar código de aplicación. El túnel web pertenece solo a su red; Nginx conecta además con la red del gateway. No monta credenciales ni abre puertos adicionales. Cuatro casos de navegador HTTPS y la sonda de sesión aprobaron. [Operación y evidencia](PUBLICACION-WEB.md).

Este acceso y el M2M son independientes. Ambos dependen de PC/Docker/red y sus hostnames pueden cambiar al reiniciar. El despliegue M2M cloud ya está verificado en https://renewable-consumo-v1.onrender.com, sin cuentas ni bases privadas. [Configuración y evidencia](DESPLIEGUE-GRATUITO.md).
