# Relación entre los prácticos y las decisiones del proyecto

Revisión 2026-11-08. Se analizaron los nueve PDF de la carpeta `Practico` proporcionada por el equipo: 177 páginas. Se conservan las tecnologías ya elegidas; las diferencias se justifican por el problema del proyecto y sus costos. No se afirma que una herramienta sea superior para todos los sistemas.

## Fuentes y método de lectura

La paginación indicada es la del archivo PDF, comenzando en 1. Algunos títulos internos no coinciden con el nombre del archivo: por ejemplo, `Clase 2.pdf` se presenta como «Clase 1 Plus» y `Clase 3.pdf` como práctico de Clase 2. Por eso las referencias usan el nombre físico del PDF.

Los PDF 1 a 8 contienen páginas rasterizadas. Se renderizaron las 159 páginas, se corrigió la orientación de las clases 2 a 8 y se aplicó OCR español local. Se inspeccionaron las páginas gráficas/de poco texto y las vistas necesarias para interpretar los ejemplos. `Clase 9.pdf` contiene texto extraíble. El OCR puede alterar símbolos de código: se tomaron los conceptos, sin copiar snippets corruptos. Los enlaces a demos y ejercicios son referencias de clase, no instrucciones de ejecución automática ni requisitos nuevos del dominio.

No se ejecutaron las demos externas de las clases ni se copiaron sus credenciales de ejemplo. Se ejecutó el proyecto propio con Go, MySQL, HTTP, navegador y caída del vendedor. El inventario con conteo y hashes queda en las evidencias de esta revisión.

## Cobertura de los nueve archivos y estado vigente 2026-11-09

| PDF | Páginas | Contenido aplicado | Implementación y decisión |
|---|---:|---|---|
| Clase1.pdf |23| Microservicios/Go/REST, cohesión, contratos e interfaces, pp.4–23 | Cuatro servicios de negocio+gateway; contratos e interfaces; simulador externo no cuenta para el mínimo. D1/D2/D8/D9. |
| Clase2.pdf |11| Products/Orders, cliente HTTP por interfaz, stock/precio y timeout2 s, pp.2–11 | Consumo resuelve fichas; Solar consulta stock/oferta directa por SKU. Cotizar cambia selección, sin comprar ni descontar inventario. D1/D2/D5/D9/D10. |
| Clase3.pdf |36| Mongo/driver oficial, repositorios y Controller/Service/Repository, pp.3–35 | Mongo real para paneles/estudios; SQL privados conservan relaciones/transacción. Dominio Solar independiente de drivers. D2/D3/D4. |
| Clase4.pdf |18| Cache-aside, TTL, invalidación, ccache/Memcached, fallback y medición, pp.2–17 | Valkey compartido, TTL 60 s, generación invalidada, memoria 64 MiB y respaldo. Comparación reproducible con/sin caché. D6/D7/D13. |
| Clase5.pdf |12| Broker, ACK manual, QoS1, fanout, DLQ y backoff, pp.2–11 | Rabbit real, outbox/confirm, prefetch1, deduplicación, retry1/5/30 s y reproceso. Direct por tipo en lugar de difundir todos los eventos. D4/D5/D10. |
| Clase6.pdf |32| Solr/índice, ranking/filtros/orden, SearchEngine y reindexación, pp.4–32 | Meilisearch conservado, derivado desde MySQL; filtros/páginas/orden, outbox automática y rebuild/swap; SQL degradado. D6. |
| Clase7.pdf |17| Goroutines/channels/Mutex/context y race, pp.1–17 | Context/deadlines, locks locales de limitador/circuito/balanceador, transacciones para consistencia durable, pruebas concurrentes y detector de carreras. D2/D4/D5/D10/D12. |
| Clase8.pdf |10| HA/resiliencia, NGINX, balanceo/healthchecks/retries/fallback, pp.1–9 | Dos Solar, round-robin, retiro/reingreso observado, timeout/circuito, recuperación de broker y SQLfallback. Se conserva gateway Go. D10/D12/D13. |
| Clase9.pdf |18| Logs/métricas, correlación, Prometheus/Grafana y demo logs MySQL, pp.2–18 | Dashboard/alertas Prometheus, JSON/stdout+Loki y trazas Jaeger con W3C propagado por evento. D8/D11/D13. |

## Diferencias conservadas y por qué convienen aquí

### Catálogo MySQL y SQL explícito frente al ejemplo MongoDB

`Clase 3.pdf` enseña cómo aislar el driver detrás de repositorios; no exige que toda entidad use MongoDB ni prescribe GORM. El catálogo ya usa consultas MySQL parametrizadas, unicidad de semillas y una transacción de snapshot para total/página. Con `database/sql` el aislamiento, los parámetros y Commit/Rollback de estas pocas consultas son explícitos. Cambiarlo por MongoDB o un ORM no mejora esa regla del negocio y exigiría volver a validar persistencia y consultas.

La ventaja es concreta para este repositorio de lecturas; se acepta escribir SQL y mantener migraciones. MongoDB ya almacena documentos solares con índice único, CAS y journaling; el segundo tipo de almacenamiento está integrado. Ver [D3](adr/ADR-003.md#revision-vigente-20261108) y [transacciones Go](https://go.dev/doc/database/execute-transactions).

### Valkey frente a ccache/Memcached

`Clase 4.pdf` enseña tanto el beneficio de la RAM local como el costo y las ventajas de una caché compartida. El diseño anterior seleccionó Valkey: conservarlo permite compartir claves/expiración entre réplicas y reutilizar operaciones simples de claves y contadores. Se mantiene cache-aside con TTL e invalidación; no se presupone consistencia fuerte por ser distribuida.

ccache evita el salto de red, pero distribuye copias por proceso. Memcached sería una alternativa suficiente para fichas; el beneficio de conservar Valkey aquí es evitar cambiar la elección y disponer de esas primitivas compartidas para la evolución prevista. Se aceptan el salto de red y otra dependencia. La carga final compara consultas calientes con bypass y con caché; se informan latencias y llamadas evitadas, sin atribuir una mejora universal a la herramienta. La compatibilidad citada se limita a Redis OSS 7.2 y anteriores; no se extiende indiscriminadamente a todas las versiones de Redis. [D7](adr/ADR-007.md#revision-vigente-20261108), [documentación Valkey](https://valkey.io/topics/migration/).

### Meilisearch frente a Solr

`Clase 6.pdf` usa Solr y explica sus ventajas de schema/analyzers, filtros `fq`, relevancia y boosts/eDisMax. El diseño previo eligió Meilisearch. Para una interfaz de catálogo con nombres/categorías, filtros y orden acotados, conservar una API de documentos/configuración reduce el trabajo de integrar ese alcance. Se preserva la separación de fuente canónica/índice. Este adaptador agrupa Search, indexación y reconstrucción en el paquete features; no introduce una interfaz SearchEngine separada porque solo existe un motor implementado y comparte sus tareas de sincronización. Se acepta más acoplamiento de ese adaptador a Meilisearch; el dominio de consumo sigue independiente y el HTTP del motor se prueba contra el servicio real.

No es una afirmación de mayor velocidad o relevancia. Se acepta menos control que un diseño Solr con analizadores y consultas avanzadas. Si el dominio pide ese control, Solr debe reconsiderarse en el mismo ADR. Filtros, orden, paginación, reconstrucción y sincronización automática ya se implementan; integración mide el objetivo saludable de 5 s. [D6](adr/ADR-006.md#revision-vigente-20261108), [filtros Meilisearch](https://www.meilisearch.com/docs/capabilities/filtering_sorting_faceting/getting_started).

### Gateway Go frente a NGINX

`Clase 8.pdf` muestra NGINX y balanceo con healthchecks. El gateway Go existente ya conserva contratos, correlación, límites y presupuestos de cada operación. Mantenerlo evita sumar otro salto/proceso para esta demostración y permite probar esos comportamientos en el mismo componente. Esa es una ventaja de integración para el proyecto, no un argumento de que un proxy propio rinda más o sea más maduro que NGINX.

El costo aceptado es implementar y probar selección concurrente, healthchecks y retiro/reingreso. El código actual balancea dos réplicas y el ensayo retira/reintegra una de ellas con estado y distribución observables. [D12](adr/ADR-012.md#revision-final-20261109) documenta la implementación y sus límites y conserva NGINX/HAProxy como alternativas.

### Logs JSON/stdout y Jaeger frente a logs en MySQL

`Clase 9.pdf` persiste logs en `app_logs` y Grafana consulta MySQL. Conservar stdout evita que cada operación de negocio tenga que escribir telemetría en SQL y mantiene el diagnóstico de una caída de la base separado de esa base. OpenTelemetry/Jaeger añade spans con relación padre/hijo para este flujo distribuido. La correlación por ID de la clase sigue presente.

Se pierde la consulta SQL simple del ejemplo; Loki agrega/retiene logs 72 h y Grafana centraliza métricas/logs/trazas. Jaeger en memoria pierde trazas al reiniciar y no reemplaza Prometheus. El alcance implementado y sus límites de retención/notificación se describen en OBSERVABILITY.md. [D11](adr/ADR-011.md#revision-vigente-20261108).

### Timeouts y degradación frente a copiar retries/fallback de las demos

Se usan los patrones de `Clase 8.pdf`, pero no se copian esperas 1/2/4 s a un flujo cuyo presupuesto total es 4 s y cuyo consumidor interactivo dispone de 1.2 s. La reconexión al broker real y los reintentos de mensajes admiten backoff acotado. En cotización se conserva el consumo válido y se informa indisponibilidad: devolver «paneles más vendidos» no demostraría cobertura, precio o stock disponibles.

El simulador también mejora el alcance del mock siempre exitoso de `Clase 1.pdf`: prueba falta de stock, contratos incompatibles, tiempos y caída, porque esos casos cambian una decisión comercial. Cotizar sigue siendo distinto de la orden que descuenta inventario en `Clase 2.pdf`. [D9](adr/ADR-009.md#revision-vigente-20261108) y [D10](adr/ADR-010.md#revision-vigente-20261108).

## Criterios que no se copian como garantías generales

- Una caché distribuida comparte datos, pero no implica consistencia total. TTL, invalidación, carreras y la fuente de verdad deben resolverse.
- Un Mutex protege memoria de un proceso; la operación durable entre réplicas necesita restricciones/transacciones/deduplicación en almacenamiento.
- Un channel o goroutine local no acredita la mensajería de dominio exigida por el TP.
- Un mock o una prueba de contrato local no acredita que el grupo vendedor real haya acordado o desplegado ese contrato.
- Los tiempos, credenciales, puertos de demostración y porcentajes de disponibilidad de clase no son mediciones ni acuerdos del proyecto.

Estas precisiones conservan el propósito didáctico y explican las condiciones necesarias al aplicarlo a un sistema distribuido real.

## Aplicación posterior del práctico 6 a paneles — 2026-11-09

La búsqueda del cliente se traslada a paneles sin sustituir Meilisearch por Solr ni agregar infraestructura. Se usan los conceptos de fuente canónica, índice derivado, filtros, orden, páginas y reconstrucción: Mongo conserva fichas y pendientes atómicos; Solar sincroniza por leases y verifica la tarea del motor. Un respaldo Mongo conserva lectura ante caída. La elección previa y sus límites frente a los analizadores/boosts de Solr siguen justificados en D6; no se atribuye superioridad general de rendimiento. El API/índice de equipos se mantiene para compatibilidad y el selector del cálculo.
