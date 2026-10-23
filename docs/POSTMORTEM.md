# Postmortem - caída controlada de MySQL

Ensayo local realizado el 22 de octubre de 2026, zona America/Buenos_Aires (UTC-03:00). Solo se detuvo MySQL del proyecto Compose renewable-energy-entrega2. No se borraron volúmenes ni se afectaron otros proyectos.

## Impacto

La simulación por fichas dejó de obtener el catálogo y respondió 503 controlado. La API M2M con parámetros externos siguió funcionando: HTTP 200 en 6.253 ms. No hubo guardados anónimos ni se sustituyeron fichas por valores inventados.

## Línea temporal

| Hora local | Hecho observado |
|---|---|
| 21:03:50 | baseline |
| 21:03:52 | mysql_stopped |
| 21:04:02 | catalog_unavailable |
| 21:04:02 | m2m_remains_available |
| 21:04:03 | mysql_start_requested |
| 21:04:13 | catalog_recovered |
| 21:04:13 | catalog_restart_requested |
| 21:04:14 | catalog_restart_idempotent |

## Detección y respuesta

Cuatro solicitudes por catálogo devolvieron 503. Duraciones: 2013.956 ms, 2014.714 ms, 1761.045 ms, 24.369 ms. Las tres primeras alcanzaron la dependencia; el circuito evitó otra consulta en la cuarta, reduciendo su duración. Readiness de Electrodomésticos refleja la pérdida de MySQL y el gateway deja de estar listo para el flujo completo, pero conserva el direccionamiento de rutas que pueden responder.

Se solicitó iniciar MySQL y el catálogo se recuperó en 10.10 s desde esa acción, incluyendo espera de recuperación y la ventana del circuito. Un reinicio posterior de Electrodomésticos verificó que las semillas conservan 6 fichas frente a 6 iniciales. No fue necesario reiniciar Consumo ni gateway ni modificar datos.

## Causa

Falla deliberada de una dependencia indispensable del catálogo: una única instancia MySQL. El cálculo M2M no depende de ese catálogo, por lo que el aislamiento por ruta mantuvo su disponibilidad.

## Qué funcionó y qué falta

Timeout HTTP, circuit breaker, respuesta recuperable, separación entre proceso vivo/readiness y semillas idempotentes. Faltan métricas/alertas sobre el circuito y MySQL, una política de copias/restauración y observación sostenida de recuperación. El ensayo no demuestra alta disponibilidad ni tolerancia a caída entre réplicas.

Reproducir voluntariamente con Python disponible: `python tests/failure_probe.py`, desde la raíz con el sistema iniciado. El script detiene temporalmente la base de este proyecto y la vuelve a iniciar incluso si falla una comprobación del período de caída. [Datos originales](evidence/entrega2/fallo-controlado.json).
