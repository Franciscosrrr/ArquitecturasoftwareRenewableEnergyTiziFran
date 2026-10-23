# Observabilidad y diagnóstico local

Arrancar con `scripts/compose.ps1 up -d --build` o `sh scripts/compose.sh up -d --build`. Abrir http://localhost:16686, seleccionar `gateway` y buscar trazas. Una simulación por fichas produce spans de gateway, cliente HTTP, Consumo, resolución de catálogo, Electrodomésticos, consulta MySQL y cálculo. La estimación M2M no necesita una consulta MySQL.

La sección «Referencia del cálculo» muestra traceId. Buscarlo en Jaeger o abrir `http://localhost:16686/trace/TRACE_ID`. La cabecera X-Request-Id permite además correlación provista por el consumidor. `traceparent` se propaga con OpenTelemetry a través del proxy y del cliente interno.

Ver logs con `scripts/compose.ps1 logs --no-log-prefix gateway consumo electrodomesticos` (o el script sh equivalente). Cada línea de operación HTTP es JSON con service, route, method, status, duration_ms, request_id, trace_id y span_id. No se registran bodies, selecciones, claves o contraseñas. Las rutas desconocidas usan una etiqueta fija para no copiar datos arbitrarios a logs.

Los endpoints `/health/live` y `/health/ready` requieren el secreto operativo de `.env`. No pegarlo en capturas ni documentos. Los contenedores ejecutan automáticamente `/healthcheck`. Para un reporte sin claves: `docker compose ps`.

Si MySQL no responde, el catálogo y la simulación por fichas devuelven 503. Consumo mantiene operativa su API M2M. El circuito del cliente de catálogo limita llamadas durante el fallo. Para recuperar un ensayo local: iniciar MySQL, esperar su healthcheck y reintentar luego de hasta 10 s. No borrar el volumen para solucionar una credencial desincronizada; recuperar el `.env` correspondiente.

Jaeger almacena trazas en memoria y su exportación puede tardar unos segundos. Reiniciar Jaeger borra su historial. El directorio `docs/evidence/entrega2` conserva una respuesta y una traza reales exportadas. El muestreo actual es 100% para facilitar revisión local; no es una política definitiva de retención.

D11 registra como siguiente paso métricas agregadas y dashboard de latencia, tráfico, errores, circuito y cola. Esas métricas todavía no están implementadas; no confundir el visor de trazas con un tablero de métricas.
