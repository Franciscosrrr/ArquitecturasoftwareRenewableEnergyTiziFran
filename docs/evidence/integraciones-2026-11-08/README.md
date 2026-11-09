# Verificación del incremento aplicado

Fecha de revisión: 2026-11-08, America/Buenos_Aires. Los cambios están aplicados al repositorio original. No se hizo commit, push ni despliegue público.

## Comprobaciones ejecutadas

- Se construyeron las imágenes y se arrancó Compose: MySQL, Jaeger, Electrodomésticos, Consumo, Solar, proveedor-paneles-sim y gateway.
- La suite Go pasó en Docker con MySQL real. El repositorio del catálogo registró cobertura 82.8% en ese paquete; no se trata de cobertura global. Las 246 pruebas/subpruebas y un skip de go-tests.jsonl corresponden a la primera ejecución nativa previa, conservada como antecedente.
- verificar-real ejecutó nueve pruebas HTTP, todas aprobadas: M2M, permisos, JSON/límites, catálogo/MySQL, cálculo y contratos publicados.
- Playwright ejecutó ocho e2e en escritorio/móvil, todos aprobados, contra el sistema real. La prueba Solar cotiza contra el simulador y después inyecta un 503 en la respuesta del navegador para comprobar consumo conservado. Los resultados nuevos están en e2e/, sin sobrescribir Entrega 2.
- Se detuvo y restauró el proceso proveedor-paneles-sim: cotización 503, M2M 200, recuperación 200. Ver fallo-vendedor.json y POSTMORTEM.
- Se verificó la migración idempotente de la nueva credencial, sin rotar secretos anteriores. No se incluyen claves reales.
- OpenAPI completo de Clínica 1.0.1 semánticamente igual al local, y regresiones de sus equipos aprobadas.
- Cuatro fuentes Mermaid renderizadas y revisadas. Tres PNG históricos conservados. Diagramas vigentes contrastados con Compose, rutas y llamadas.
- Nueve PDF prácticos analizados, 177 páginas; inventario y hashes en pdf-inventory.json. Los PDF no se copiaron al repositorio.
- D1-D13 sin archivos duplicados por número; contexto, decisión y porqué explícitos en todos los registros y antecedentes complementarios. Se corrigen enlaces a los E2 consolidados.

## Alcance y límites

La prueba UI previa con fixtures HTTP (ui-smoke.json/ui-*.png) se conserva separada del e2e Docker real. Las evidencias nuevas no convierten el simulador en integración real entre grupos. Faltan NoSQL, outbox/broker, operación durable, motor de búsqueda, caché medida, identidad completa, balanceo, métricas/tablero/alerta y HTTPS. No se ejecutaron una prueba de capacidad de la revisión completa ni la suite completa del repositorio de Clínica.

Las pruebas adicionales de concurrencia del limitador y del circuito pasaron con -race; la salida está en race-linux.txt. El MinGW del host no soporta 64 bits, por lo que el detector se ejecuta en un contenedor Linux temporal con GCC, sin cambiar el compilador del usuario.
