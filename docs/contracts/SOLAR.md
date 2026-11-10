# Cotización solar y vendedor simulado

Fecha 2026-11-08. Proveedor real todavía sin contrato; el equipo autorizó una simulación comercial. No constituye integración acreditada entre grupos.

## Operaciones versionadas

| Interfaz | Operación | Versión | Autenticación |
|---|---|---|---|
| Web → gateway → Solar | POST /api/v1/publico/solar/cotizaciones | solar-v1 / 1.0.0 | Pública, limitada por tráfico |
| Solar → Consumo directo | POST /api/v1/publico/consumo-catalogo | catalog-v1 existente | Dentro de la red backend |
| Solar → vendedor simulado directo | POST /api/v1/cotizaciones | paneles-sim-v1 / 0.1.0-simulated | Authorization: Bearer PANEL_PROVIDER_API_KEY |

Contratos: [Solar](solar-v1/openapi.json), [vendedor](paneles-sim-v1/openapi.json). Ejemplos completos: [solicitud Solar](solar-v1/examples/solicitud.json), [respuesta Solar](solar-v1/examples/respuesta.json), [solicitud comercial](paneles-sim-v1/examples/solicitud.json), [oferta comercial](paneles-sim-v1/examples/respuesta.json).

## Reglas y consecuencias

Solar recibe consumo por catálogo, horasSolPico entre 0.5 y 8 y factorRendimiento entre 0.1 y 1. Rechaza cadenas numéricas, nulos, parámetros con más de seis decimales, campos extra y cuerpos de más de 256 KiB. Consumo valida los equipos según catalog-v1. Energía diaria positiva y hasta 100000 kWh; potencia objetivo máxima 10000000 W. Una estimación de consumo cero sigue siendo válida en Consumo, pero no requiere dimensionar ni cotizar paneles: Solar devuelve 422.

potenciaObjetivoW=ceil(promedioDiarioKWh*1000/(horasSolPico*factorRendimiento)). Se toma la energía diaria redondeada a seis decimales del contrato de Consumo y se opera con racionales. Cantidad de paneles=ceil(potenciaObjetivoW/potenciaPanelW). El vendedor informa modelo, stock y precio en centavos de ARS; Solar valida cobertura, stock y total exacto. Las cotizaciones son lecturas sin reserva ni compra; repetir no duplica pedidos. X-Request-Id es correlación, no idempotencia.

Ejemplo de 47.4 kWh en 30 días, 4 HSP y rendimiento 0.8: promedio 1.58; objetivo 494 W; el simulador ofrece dos paneles de 450 W y stock 1000. Precio didáctico de cada panel: 25000000 centavos; total 50000000 centavos. No es tarifa energética ni una oferta comercial real. El dato externo afecta cantidad, viabilidad por stock e importe de la oferta.

## Tiempos, errores y usuario

Solar usa 4 s globales, hasta 2.5 s hacia Consumo y hasta 1 s hacia el vendedor. Sin reintentos automáticos. Circuito comercial: tres fallos consecutivos, abierto diez segundos y una prueba posterior. La respuesta HTTP saliente se acota a 64 KiB y debe ser JSON en 200. No se siguen redirecciones con la credencial.

| HTTP Solar | Código | Acción observable |
|---|---|---|
| 400 | FORMATO_INVALIDO | Corregir JSON, estructura o campos desconocidos |
| 409 | STOCK_INSUFICIENTE | No presentar una instalación como disponible |
| 413 | CUERPO_DEMASIADO_GRANDE | Reducir solicitud |
| 415 | TIPO_CONTENIDO_NO_SOPORTADO | Enviar application/json |
| 422 | VALIDACION | Revisar equipos/consumo/parámetros |
| 429 | LIMITE_DE_TRAFICO | Esperar Retry-After |
| 500 | ERROR_INTERNO | Diagnóstico por traceId |
| 502 | CONTRATO_PROVEEDOR_INVALIDO | Revisar cambio del vendedor; no confiar en oferta |
| 503 | PROVEEDOR_NO_DISPONIBLE | Mantener cálculo previo y permitir reintento |

La interfaz cancela a los cinco segundos. Se conserva el consumo previo tanto en timeout como en error de stock o contrato. No se ofrece una cotización de éxito inventada. El modo simulado se informa en pantalla. Las horas y el rendimiento son supuestos educativos explícitos, no datos meteorológicos ni garantía de producción.

## Arranque y migración

Ejecutar scripts/compose.ps1 up -d --build (o sh scripts/compose.sh up -d --build). Los scripts agregan PANEL_PROVIDER_API_KEY si falta sin rotar las claves existentes. Compose inicia Solar y proveedor-paneles-sim, sin exponer sus puertos al host. El gateway publica /solar-openapi.json. El proveedor simulado no depende del mock histórico de /estimaciones-consumo.

Al recibir contrato real: conservar una copia versionada publicada por el vendedor, adaptar sus operaciones/autenticación/monedas, ejecutar tests contra sus ejemplos y sandbox, acordar tiempos/errores/cuotas y actualizar D9 en el mismo archivo. PANEL_PROVIDER_MODE=externo exige HTTPS y PANEL_PROVIDER_URL configurada; cambiar solo esa URL sirve únicamente si el contrato real coincide con el de simulación. El arranque verifica una credencial comercial de al menos 32 caracteres. Ajustar esa validación si el contrato real establece otra forma de credencial.

## Verificación y fallo

Los tests Go comprueban dimensionamiento, stock, importes, respuesta simulada, mutaciones incompatibles, timeout, redirección, circuito y recuperación. El test HTTP integra Solar con servidores de prueba de Consumo y vendedor; no se presenta como MySQL real ni prueba del proveedor externo real.

Con Compose completo, calcular consumo, cotizar y detener únicamente proveedor-paneles-sim. La siguiente cotización debe devolver 503 y conservar el cálculo. Reiniciar el vendedor y verificar recuperación tras la ventana del circuito. Registrar tiempos y trazas en POSTMORTEM; el ensayo se ejecutó en esta revisión; sus tiempos y resultados están en POSTMORTEM y evidence/integraciones-2026-11-08/fallo-vendedor.json.


Verificación actualizada 2026-11-08: build/arranque Docker, MySQL real, suite HTTP y ocho e2 e del incremento aprobados. La caída de proveedor-paneles-sim y recuperación también fueron ejecutadas. No se prueban compras ni el proveedor externo real.


## Integración durable vigente2026-11-09

El flujo autenticado consume ConsumoEvaluado.v1 y compara fichas Mongo con su superficie/versión/precio. Para confirmar la alternativa comercial utiliza [vendedor por SKU0.2.0-simulated](paneles-sim-v2/openapi.json), operación POST /api/v2/cotizaciones, manteniendo v1 para la cotización temporal anterior. [Solicitud](paneles-sim-v2/examples/solicitud.json) y [respuesta](paneles-sim-v2/examples/respuesta.json). Sin stock se prueba otra alternativa factible; cambio incompatible de SKU/potencia/cantidad/moneda/precio de referencia conocido se rechaza. Si el precio de catálogo es desconocido se selecciona por superficie y se conserva la oferta confirmada aparte.

Worker con 12 s, HTTP vendedor1 s y circuito; retry durable1/5/30 s y DLQ. Claim/hash/evento único en Mongo con journaling y ACK posterior. Versionar otra cobertura conserva estudios previos. [API privada](private-v1/README.md), [eventos](events/v1/README.md), [postmortem final](../POSTMORTEM.md). El contrato M2M de Clínica no se altera por este flujo. Un vendedor real requiere adaptar y probar el cliente contra su contrato publicado; cambiar el modo en una variable no acredita esa integración.
