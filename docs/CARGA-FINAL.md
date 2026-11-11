# Carga y efecto de la caché — 2026-11-09

Ensayo contra la versión instalada en el repositorio original, con todas las dependencias reales, contenedores limitados y un solo proyecto activo. [Datos](evidence/final-2026-11-09/carga.json). No hubo latencia artificial ni compilación simultánea. La máquina Windows tiene 16 GB; los puertos locales no representan una medición cloud.

## Catálogo caliente: misma consulta, ocho clientes concurrentes

`GET /api/v1/publico/electrodomesticos?tamano=20&orden=nombre&q=referencia`, desde la red de pruebas hacia Electrodomésticos. Una lectura de preparación por escenario queda fuera de las 400 medidas. El bypass de caché exige el secreto interno y el gateway elimina esa cabecera de clientes externos. Se compara el mismo motor Meilisearch y consulta; el bypass no desactiva el buscador ni usa una base falsa.

| Escenario | Solicitudes | p50 ms | p95 ms | Máximo ms | req/s | Errores |
|---|---:|---:|---:|---:|---:|---:|
| Sin caché |400|44.548|51.475|56.96|172.93|0|
| Con caché |400|4.293|8.098|41.242|1398.46|0|

Todas las lecturas sin caché informaron origen Meilisearch y las 400 posteriores a la preparación informaron cache. En ese tramo repetitivo se evitaron 400 consultas al motor y el p95 bajó aproximadamente 84.3%. El catálogo se carga al abrir la calculadora y sus búsquedas se repiten; esta evidencia muestra una necesidad de lectura concreta. No extrapola ese hit ratio a consultas distintas ni predice tráfico orgánico. Con una consulta única o un catálogo mínimo, el costo operativo de Valkey puede no compensar; se debe revisar D7 con tráfico real. El TTL es 60 s y la invalidación usa generación después de confirmar la indexación.

## M2M y confirmación durable

| Flujo | Solicitudes | Concurrencia | p95 HTTP ms | Máximo ms | Resultado |
|---|---:|---:|---:|---:|---|
| M2M a cuota permitida |15|1|4.323|4.966|15 respuestas 200|
| Confirmar evaluación |10|1|21.726|22.745|10 respuestas 201|

La prueba M2M respeta 1 solicitud/s; no mide una tasa que el contrato impide. La confirmación utiliza diez configuraciones independientes con un equipo cada una, contexto autenticado, MySQL/outbox y dos workers Solar. El pacing 0.25 s se realiza antes de medir la solicitud. Su throughput incluye crear configuraciones y consultar estudios, por lo que no es la capacidad máxima de confirmación.

Se observaron 10/10 estudios completados dentro de 15 s, p95 de visibilidad 1.053 s y máximo 1.056 s. La observación parte de la respuesta de confirmación y usa consultas periódicas; incluye hasta el intervalo de polling y no conoce el instante exacto de finalización.

## Alcance de los resultados

El p95 del catálogo fue menor de 500 ms y el de confirmación menor de 1 s en esta carga; el objetivo saludable de estudios se cumplió en esta muestra. No son un SLA ni prueban 100 equipos por configuración bajo carga. Los límites numéricos/100 equipos se verifican por separado en pruebas del algoritmo/contrato. Las fallas se ensayan aparte en POSTMORTEM para no confundir carga saludable y recuperación.

Reproducir con el sistema iniciado: `scripts/compose.ps1 --profile verificar run --no-deps --rm verificar-real python -B tests/load_final.py` (usar compose.sh en Linux). Requiere un entorno de prueba: crea un usuario y diez escenarios propios para comprobar persistencia. La cuota de producción no se desactiva. Las métricas y el dashboard muestran hit/miss, latencia y distribución; [observabilidad](OBSERVABILITY.md).


## Alcance frente al proveedor real — 11/11/2026

Estos resultados corresponden a la carga ejecutada el 09/11, antes de integrar Comparador de Proveedores. No miden la capacidad máxima ni el p95 del ranking externo. Las comprobaciones del proveedor real y el respaldo del 11/11 se registran por separado en VERIFICACION-FINAL.md.
