# Mock ejecutable de la primera entrega

Implementación: [mock/server.py](../../mock/server.py), Python estándar sin dependencias externas. Es una herramienta de prueba local; el backend del sistema mantiene Go. No es un despliegue de producción ni sustituye al proveedor asignado.

## Arranque

Desde la raíz: `.\scripts\compose.ps1 up --build` en PowerShell o `sh scripts/compose.sh up --build` en Linux/macOS. Ambos generan automáticamente HEALTHCHECK_TOKEN sin guardarlo ni imprimirlo. URL: http://localhost:8080. Alternativa de desarrollo con Python 3.12 o superior: `python mock/server.py`, con HEALTHCHECK_TOKEN aleatorio ya definido en el entorno (mínimo 32 caracteres). Escucha solo localhost fuera del contenedor; Compose publica únicamente en 127.0.0.1.

## Rutas disponibles

| Ruta | Comportamiento |
|---|---|
| GET /health/live | Exige Authorization: Bearer con HEALTHCHECK_TOKEN; 401 sin credencial válida, 200 con estado mínimo en español si está autorizado |
| GET /openapi.json | Contrato M2M original |
| GET /guest-openapi.json | Contrato invitado original |
| POST /api/v1/estimaciones-consumo | Cálculo decimal; exige X-API-Key de prueba |
| POST /api/v1/publico/consumo | Igual cálculo sin credenciales |
| POST /api/v1/publico/recomendaciones-solares | Cálculo con panel ilustrativo fijo de 500 W y 2,5 m²; recurso MANUAL |

M2M: clave `mock-consumidor` permite calcular; `mock-sin-permiso` produce 403; ausente/desconocida produce 401. Son constantes públicas exclusivamente del mock. Ninguna credencial real está incluida.

## Escenarios y errores

- Solicitud de ejemplo: 200; total 47,4 y promedio 1,58 kWh.
- JSON mal formado: 400. Content-Type distinto de application/json: 415. Más de 256 KiB: 413.
- Campos/tipos/rangos/variantes inválidos, ids repetidos, más de seis decimales o diasUso mayor al período: 422.
- Rate limit real en memoria: token bucket de 60/minuto y ráfaga 10 para consumo; 10/minuto y ráfaga 2 para Solar. El mock agrupa M2M por su única credencial válida y visitantes por IP/ruta; no confía en X-Forwarded-For.
- Para probar fallas de forma determinista, enviar `X-Mock-Scenario: 429`, `500` o `503` sobre una solicitud válida. 429 y 503 incluyen Retry-After. Esta cabecera solo existe en el mock; no se añade al contrato productivo.
- Solar por UBICACION devuelve 503: no hay proveedor asignado ni HSP inventadas. Recurso MANUAL produce resultados calculados con el catálogo ilustrativo fijo, incluido consumo cero o superficie insuficiente.
- X-Request-Id se devuelve como correlación si es válido; si no se envía, se genera. Todas las respuestas incluyen `X-Mock: entrega-1`.

## Ejemplos de uso

Desde raíz (Windows: reemplazar curl por curl.exe):

```sh
curl -i http://localhost:8080/api/v1/estimaciones-consumo -H "Content-Type: application/json" -H "X-API-Key: mock-consumidor" --data-binary @docs/contracts/v1/examples/solicitud-valida.json
curl -i http://localhost:8080/api/v1/estimaciones-consumo -H "Content-Type: application/json" -H "X-API-Key: mock-consumidor" --data-binary @docs/contracts/v1/examples/solicitud-invalida-dias.json
curl -i http://localhost:8080/api/v1/publico/recomendaciones-solares -H "Content-Type: application/json" --data-binary @docs/contracts/guest-v1/examples/solicitud-solar-manual.json
curl -i http://localhost:8080/api/v1/estimaciones-consumo -H "Content-Type: application/json" -H "X-API-Key: mock-consumidor" -H "X-Mock-Scenario: 503" --data-binary @docs/contracts/v1/examples/solicitud-valida.json
```

## Pruebas y límites

`.\scripts\compose.ps1 --profile verificar run --build --rm verificar` (Linux/macOS: `sh scripts/compose.sh --profile verificar run --build --rm verificar`) o `python -m unittest discover -s mock -p "test_*.py" -v`. Las pruebas inician un servidor HTTP en puerto efímero, envían solicitudes y contrastan respuestas con el subconjunto de esquemas usado y los ejemplos.

No hay BD, archivos de resultados, outbox, eventos, login real, frontend, historial ni endpoints de catálogo implementados. Los contadores técnicos expiran en memoria. La validación propia de esquemas no certifica conformidad integral OpenAPI; admite las restricciones utilizadas por estos contratos. Se requiere Content-Length, sin transferencia chunked. El servidor estándar es adecuado para esta simulación local, [no para producción](https://docs.python.org/3.12/library/http.server.html).

Los cálculos del mock son un oráculo de ejemplos, no una segunda biblioteca compartida con el backend. Al implementar Consumo en Go se deben ejecutar las mismas pruebas de contrato y retirar el cálculo simulado de los caminos productivos.
