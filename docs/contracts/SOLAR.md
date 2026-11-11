# Integración Solar vigente — 11/11/2026

La calculadora, la comparación temporal y los estudios guardados utilizan **Comparador de Proveedores 1.1.0**. El vendedor simulado queda como respaldo identificado ante un fallo técnico. Las decisiones y pruebas anteriores se conservan en D9/D10 y en las evidencias fechadas.

## Operaciones

| Tramo | Operación | Contrato |
|---|---|---|
| Navegador → Gateway → Solar | POST /api/v1/publico/solar/cotizaciones | [Solar 1.1.0](solar-v1/openapi.json) |
| Navegador → Gateway → Solar | POST /api/v1/publico/recomendaciones-solares | [Invitado 1.2.0](guest-v1/openapi.json) |
| Cuenta → confirmación → outbox → worker Solar | ConsumoEvaluado.v1 | [Evento](events/v1/README.md), [API privada 1.2.0](private-v1/README.md) |
| Solar → Consumo directo | POST /api/v1/publico/consumo-catalogo | [Catálogo](catalog-v1/openapi.json) |
| Solar → proveedor real directo | GET /api/v1/proveedores/recomendados?producto=SKU&cantidad=N&limite=1 | [Comparador 1.1.0](comparador-proveedores-v1.1.0/README.md) |
| Solar → simulador interno | POST /api/v2/cotizaciones | [Simulador por modelo](paneles-sim-v2/openapi.json) |

La clave del proveedor se envía únicamente desde Solar por X-API-Key. No se inicia esta comunicación desde React ni se utiliza nuestro Gateway como bus interno. La operación no compra ni reserva.

## Datos y selección

Consumo aplica las reglas de energía. Solar dimensiona con HSP, rendimiento, cobertura, superficie y fichas técnicas locales. La cantidad se calcula localmente; para los modelos existentes se utiliza la equivalencia explícita PANEL-SIM-450 → PAN-450W-MONO y PANEL-SIM-550 → PAN-550W-MONO. No se deduce potencia de un nombre ni se inventa una superficie entregada por el proveedor.

El proveedor es autoridad de precio y stock. Sus números decimales ARS se convierten exactamente a centavos y el total incluye descuento aplicado y flete. Se consulta cada alternativa técnicamente factible y se elige la oferta disponible de menor costo total. El precio didáctico del catálogo no invalida una oferta real. La búsqueda enriquece los resultados con precio unitario/stock antes de aplicar filtros monetarios, orden y página; el precio filtrado no incluye descuento ni flete.

Una oferta real conserva proveedor, SKU comercial, referencia de petición y fecha. El campo cotizacionId queda vacío porque ese grupo no persiste cotizaciones; referenciaProveedor sirve para correlación. Nuestros estudios guardan la foto recibida y su origen, sin congelar un precio ni reservar inventario en la empresa.

## Fallas y tiempos

Una conexión fallida, timeout, 429, 5xx o respuesta incompatible hace recalcular **todo el estudio** con el simulador. Respuesta, web e historial identifican el origen simulado. No se mezclan precios reales y simulados en una misma selección. Si falla también el simulador, la solicitud devuelve error controlado y el estudio durable conserva su política de reintentos.

200 con lista vacía y 404 PRODUCTO_NO_ENCONTRADO informan ausencia de oferta: se prueba otro modelo. No activan un stock ficticio. Una entrada inválida o cancelación no activa respaldo. Timeout comercial: 1 s; presupuesto público Solar: 4 s; worker: 12 s. Circuito de tres fallos/10 s y una prueba de recuperación. Retry-After de 429 extiende la espera cuando corresponde. Las llamadas comerciales no tienen reintentos inmediatos; mensajería conserva retry 1/5/30 s y DLQ para fallas que no pudieron resolverse.

La URL recibida es HTTP y requiere la excepción académica explícita limitada a su IP. La clave permanece privada, pero el tramo carece de TLS. No se presenta como HTTPS entre grupos. La web pública usa HTTPS hasta su entrada en Cloudflare; son tramos diferentes.

## Ejemplos, arranque y comprobación

[Solicitud](solar-v1/examples/solicitud.json), [oferta real adaptada](solar-v1/examples/respuesta-proveedor-real.json) y [respaldo identificado](solar-v1/examples/respuesta-respaldo-simulado.json). Las respuestas son ejemplos observados, no precios vigentes garantizados. El ejemplo previo [respuesta.json](solar-v1/examples/respuesta.json) sigue describiendo el modo didáctico.

[Configuración y arranque](../CONFIGURACION.md). La API M2M 1.0.1 de Clínica no cambia por esta integración. [Resultados real/respaldo](../evidence/proveedor-real-2026-11-11/README.md), [postmortem](../POSTMORTEM.md), [D9](../adr/ADR-009.md#proveedor-real-20261111) y [D10](../adr/ADR-010.md#proveedor-real-20261111).
