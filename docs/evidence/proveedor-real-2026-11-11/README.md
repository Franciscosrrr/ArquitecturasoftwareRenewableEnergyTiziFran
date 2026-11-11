# Proveedor real y respaldo simulado — 11/11/2026

Integración probada contra **Comparador de Proveedores 1.1.0**, URL entregada por el grupo, desde Solar y su réplica. [Contrato y decisiones](../../contracts/comparador-proveedores-v1.1.0/README.md). No se incluyen claves.

| Comprobación | Resultado | Evidencia |
|---|---|---|
| Suite Solar y detector de carreras | Aprobada | [Go](go-solar-race.txt) |
| OpenAPI: método, ruta, parámetros, autenticación, campos/tipos y estados | Aprobada | [Contrato actual](go-openapi-current.txt) |
| Mutación incompatible del parámetro producto | Detectada: fallo esperado del test | [Prueba negativa](contrato-incompatible-detectado.txt) |
| HTTP completo con proveedor real | 21 aprobadas | [HTTP](http-real.txt) |
| Navegador escritorio/móvil con proveedor real | 14 aprobadas | [Navegador](e2e-real.txt), [reporte](e2e-real/resultados.json) |
| HTTP con conexión comercial rechazada | 2 aprobadas | [Respaldo](http-respaldo.txt) |
| Identidad, evaluaciones y estudios con respaldo | 3 aprobadas | [Historial](historial-respaldo.txt) |
| Navegador escritorio/móvil con respaldo | 4 aprobadas | [Navegador](e2e-respaldo.txt), [reporte](e2e-simulado/resultados.json) |
| Restauración del proveedor real | HTTP 200 y origen externo | [Respuesta](respuesta-real-recuperada.json) |
| Contrato de Clínica y revisión de secretos | Hash conservado; ninguna clave local en archivos compartibles revisados | [Resumen](resumen.json) |

El ensayo de caída cambió temporalmente **solo las dos réplicas Solar locales** a `https://127.0.0.1:1`, una conexión rechazada. No se alteró la instancia del otro grupo. Se verificó una [respuesta simulada identificada](respuesta-respaldo.json), el historial persistido y los avisos visibles. Después se restauraron URL y clave reales; las dos réplicas quedaron saludables. No se afirma haber causado una falla en Azure ni medido un SLA del proveedor.

Primer intento de navegador: 12 aprobadas y dos pruebas nuevas fallaron al buscar un nombre de botón que no incluía la flecha decorativa. Se corrigió el selector y el texto antiguo del pie de página, que decía siempre simulado. La ejecución final de los 14 casos pasó. Se conserva el [registro inicial](e2e-real-primer-intento.txt) para trazabilidad.

Capturas revisadas: [oferta real escritorio](e2e-real/proveedor-externo-escritorio.png), [oferta real móvil](e2e-real/proveedor-externo-movil.png), [respaldo móvil](e2e-simulado/proveedor-simulado-movil.png), además de las vistas de cuenta y catálogo en los mismos directorios.

Límites: solo se dimensionan las fichas locales existentes, 450/550 W, con superficies didácticas no certificadas por el proveedor. La conexión externa es HTTP, con excepción académica explícita; no se presenta como TLS entre grupos. La API de Clínica en Render no se modifica por estos cambios. La web completa depende de la PC/Docker y los cambios de código aún requieren commit/push para incorporarse a la entrega de GitHub.

Web pública restaurada y cotización externa comprobadas por HTTPS: [observación](https-web.json). Su hostname es temporal y puede cambiar al reiniciar; la URL de Clínica en Render es independiente.
