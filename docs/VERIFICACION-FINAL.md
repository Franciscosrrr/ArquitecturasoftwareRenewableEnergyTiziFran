# Implementación y verificación final local

> Actualización posterior del 09/11/2026: [M2M publicado por HTTPS temporal con Cloudflare](PUBLICACION-CLINICA.md). URL https://emails-bingo-preference-packets.trycloudflare.com; certificado, contrato y 200/401/403 comprobados. Requiere PC/Docker/red activos; la URL puede cambiar y no hay garantía de uptime. Los estados de preparación anteriores se conservan como antecedente.

Fecha 2026-11-09. Cambios aplicados al repositorio original y sistema completo funcionando en http://localhost:8080. Las entregas anteriores permanecen históricas. La evidencia final procede de servicios reales; el mock inicial es opcional y no reemplaza dependencias de estas pruebas.

## Qué quedó implementado

| Requisito | Implementación / evidencia |
|---|---|
| Servicios y estilos | Usuarios, Consumo, Electrodomésticos y Solar + gateway. Capas en Consumo y hexagonal en Solar; dominio sin drivers. |
| Relacional / NoSQL | Tres MySQL privados y MongoDB solar_db; usuarios de aplicación separados y volúmenes. |
| Operación principal | Configuración/lugar versionados, confirmación transaccional, snapshot inmutable, idempotencia y outbox. |
| Comunicación | HTTP con deadlines/circuitos y RabbitMQ real con publisher confirms, ACK manual, retry, DLQ y reproceso. |
| Búsqueda | Interfaz de paneles solares: texto, potencia, superficie, moneda/precio, orden y páginas. Índice solar separado con marca Mongo/leases/reconciliación; respaldo Mongo. El índice previo de equipos y su API se conservan. |
| Caché | Valkey cache-aside, TTL 60 s, invalidación por generación, respaldo y comparación de 400+400 lecturas. |
| Balanceo | Dos Solar, round-robin, checks activos, retiro/reingreso y métricas por instancia. |
| Resiliencia | Broker/vendedor/réplica/caché/buscador detenidos y recuperados; datos y sesión conservados. |
| Observabilidad | JSON/stdout+Loki, Prometheus, Grafana provisionado y alertas; traceparent conecta HTTP y worker. |
| Web | Invitado, cuenta, lugares, guardados/editado/duplicado, historial, comparación, variantes y administración de fichas/roles/DLQ/índice. |
| M2M Clínica | OpenAPI 1.0.1 y redondeo intactos, sin precio; 401/403/429, cuota 60/min y ráfaga 10. |
| Integración comercial | Simulador autorizado por el equipo; oferta/stock por SKU cambian la alternativa seleccionada. Falta probar el proveedor real asignado. |
| Documentación | D1–D13 canónicos, duplicados consolidados, contexto/porqué/clases/costos; fuentes y PNG vigentes; JSDoc revisado. |

## Pruebas ejecutadas

| Comprobación | Resultado | Evidencia |
|---|---|---|
| Go: todos los módulos y MySQL real | Aprobado | [log](evidence/final-2026-11-09/go-unitarias-y-race.txt) |
| Detector de carreras | Plataforma/limitador, balanceador y clientes con circuito: aprobado | mismo log |
| HTTP real |19 casos aprobados en 46.047 s; añade archivo sin versión, eliminación, propiedad, historial y sincronización solar | [log vigente](evidence/experiencia-cliente-2026-11-09/http.txt) |
| Chromium escritorio/móvil |12 aprobados en 58.2 s; 0 fallos, omitidos o flaky; flujo autenticado/anónimo, archivo, eliminación y búsqueda solar | [reporte vigente](evidence/experiencia-cliente-2026-11-09/e2e/resultados.json) |
| Fallas y recuperación |16 observaciones completadas; alerta firing verificada | [datos](evidence/final-2026-11-09/fallas.json) y [postmortem](POSTMORTEM.md) |
| Carga |800 lecturas de catálogo,15 M2M y 10 confirmaciones correctas;10 estudios completados | [análisis](CARGA-FINAL.md) |
| Contratos |8 OpenAPI,2 JSONSchema y Blueprint contra esquema oficial válidos | [resultado](evidence/final-2026-11-09/contratos-validados.json) |
| Imagen cloud preparada |200/401/403, hash de contrato,128 MiB y cierre ante caída de Consumo: aprobado localmente | [resultado](evidence/final-2026-11-09/cloud-local.json) |
| Observabilidad |6 targets up,9 paneles,3 datasources OK, logs de negocio y trazas con 6 servicios | [resultado](evidence/final-2026-11-09/observabilidad.json) |
| Estática |13 ADRs numerados,219 funciones Go,31 funciones JS nombradas y 103 unidades Python revisadas; sin faltantes detectados en ese alcance ni secretos reales | [resultado](evidence/final-2026-11-09/revision-estatica.json) |

Cobertura de sentencias en reglas puras: Consumo 93.6%, Solar 89.4%; cliente de catálogo 94.6% y cliente comercial 81.0%. La métrica corresponde a esos paquetes. Los adaptadores con bases/broker tienen pruebas HTTP/integración reales separadas y esa ejecución no se incluye en el porcentaje unitario de Go. Se ensayan fronteras de energía/superficie/precios, stock, redondeo y eventos incompatibles. La última corrección de selección solar se comprobó además con [pruebas y cobertura del dominio](evidence/final-2026-11-09/solar-reglas-final.txt). El porcentaje del dominio supera el objetivo de 80%; no se presenta como cobertura de ramas ni de todo el backend.

Se conservan los intentos que detectaron [reconexión tardía](evidence/final-2026-11-09/http-real-antes-reconexion.txt) y [una aserción dependiente de fichas residuales](evidence/final-2026-11-09/http-real-fixture-parcial.txt). Se corrigieron la detección de cierre/backoff y el aislamiento/limpieza de la prueba; la repetición final de los 17 casos pasó.

## Recursos y datos

Sistema completo y publicación aislada de Clínica: 20 contenedores; última muestra: 1646.5 MiB sumados en docker stats, Windows con 1.8 GiB libres y 88.5% de memoria usada. [Muestra vigente](evidence/experiencia-cliente-2026-11-09/recursos.json); la anterior se conserva. La memoria de Windows incluye las otras aplicaciones y la caché/VM de WSL; no equivale a la suma de contenedores. El usuario eligió conservar solo los límites del proyecto: no se creó `.wslconfig` ni se cambió el límite global de WSL. [Detalle](RECURSOS-LOCALES.md).

Se conservaron `.env`, el catálogo existente y las evidencias previas de Entrega 2. Los nuevos secretos se generan al arrancar, sin rotar los anteriores. Se guardó respaldo de archivos modificados en la carpeta de trabajo de esta revisión. La implementación se publicó en la rama deploy/estimaciones-consumo-20261109 y la CI del commit desplegado aprobó. El índice/rama de trabajo original se conservan y main no se fusionó; su revisión/integración queda como recordatorio de cierre.

## Pendientes externos explícitos

M2M está publicado con HTTPS mediante el túnel temporal documentado arriba. Por indicación del usuario, la prueba desde el backend de Clínica queda a cargo de ese grupo, con las claves que el usuario recibió por separado. El alojamiento M2M cloud ya está verificado en https://renewable-consumo-v1.onrender.com; no depende de la PC y usa claves nuevas. Debe acordarse la continuidad hasta la evaluación. Una URL fija e independiente de esta PC requiere hosting o cuenta/dominio para un despliegue apropiado. Render Free sigue como alternativa, con el límite de cold start de 1.2 s/5 s explícito. [Estado de publicación](PUBLICACION-CLINICA.md) y [alternativa Render](DESPLIEGUE-GRATUITO.md).

También falta recibir/adaptar/probar el contrato real del vendedor asignado. El simulador no acredita integración efectiva con otro grupo. HSP, paneles y precios siguen explícitamente didácticos; no se incorpora una fuente solar por ubicación ni una garantía de autonomía. Estas dependencias no se ocultan como cumplimiento externo realizado.

## Mejora de experiencia de cliente — 2026-11-09

[Archivo sin versiones artificiales, eliminación de guardados y búsqueda solar](EXPERIENCIA-CLIENTE.md). Pruebas posteriores: 19 HTTP y 12 de navegador aprobadas. Se conservan los resultados anteriores como antecedentes. Clínica mantiene OpenAPI 1.0.1.


La comprobación posterior incluyó respaldo Mongo con el motor detenido y reconstrucción automática del índice solar eliminado (29.407 s). Se recuperó Jaeger tras detectar OOM por retención ilimitada; conserva su límite de 128 MiB, con retención de 512 trazas, cola de 256 y cuatro workers. Se comprobó la retención con 1600 trazas y una nueva evaluación con traza de seis servicios. [Evidencia de experiencia de cliente](EXPERIENCIA-CLIENTE.md).


La [entrada web por Cloudflare](PUBLICACION-WEB.md) está publicada en https://conditional-wan-server-learning.trycloudflare.com, con autorización explícita del usuario. Sonda HTTPS y cuatro casos de navegador reales en escritorio/móvil aprobados; sesión tras recarga, guardados y búsqueda incluidos. El código de aplicación no cambió. Localhost y Clínica respondieron 200; el túnel M2M no se reinició. La [muestra posterior](evidence/publicacion-web-2026-11-09/recursos.json) incluye 22 contenedores y 1537.7 MiB sumados.


## Revisión documental estable — 2026-11-09

README, SPEC, arquitectura, contratos y bonus distinguen el sistema implementado, los accesos HTTPS desde la PC y el alojamiento M2M cloud verificado. La suite actual es 19 HTTP y 12 navegador locales, más cuatro casos de navegador HTTPS sobre un subconjunto. Las entregas y ensayos anteriores permanecen identificados como historia. No se cambia el OpenAPI M2M 1.0.1. El recordatorio de incorporar/revisar cambios en main y comprobar CI se dará al finalizar el despliegue; no se programa una alarma de fecha arbitraria ni se afirma haber publicado la entrega final.


## Publicación M2M cloud — 2026-11-09

Render Free, Docker, Oregon, commit 69850d800b4bbb0295d067b51387cd14324f25f3, despliegue manual/auto-deploy Off. [Sonda](evidence/render-cloud-2026-11-09/https.json): TLS/hash 1.0.1, ocho cálculos, permisos, errores, rutas privadas y cuota aprobados; p95 caliente 730.585 ms, cold start no medido. [Regresión Go](evidence/render-cloud-2026-11-09/go-gateway.txt) corrige contratos no M2M expuestos en la imagen y preserva los documentos del modo completo. [CI del commit](evidence/render-cloud-2026-11-09/ci.json). Las publicaciones de la PC se conservan. Main queda pendiente de revisión/fusión, como recordó el usuario.
