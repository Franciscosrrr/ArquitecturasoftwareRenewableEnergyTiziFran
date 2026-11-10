> Documento histórico. Estado vigente: [verificación final](VERIFICACION-FINAL.md) y [arquitectura](ARCHITECTURE.md),2026-11-09. Los pendientes de este hito no describen necesariamente la implementación actual.

# Entrega 2 - Dominio propio de punta a punta

Fecha del hito: viernes 23 de octubre de 2026. Preparación y ensayos locales: 22 de octubre, zona America/Buenos_Aires. Integrantes: Tiziano Nicolas (2407889) y Francisco Skrie (2423159).

El dominio está aprobado y los grupos fueron asignados por la cátedra, según confirmó el equipo. Aún no se recibió el contrato/documentación del proveedor. Se anuncia un único bonus: **tests e2 e**. El repositorio mantiene un alcance final más amplio; esta matriz describe solo el hito operativo.

## Matriz de aceptación del enunciado 6.3

| Requisito | Evidencia de entrega 2 | Estado |
|---|---|---|
| Al menos un servicio operativo | Consumo calcula; Electrodomésticos consulta MySQL; gateway conecta y sirve React | Implementado y probado localmente |
| Funcionalidad a compartir funcionando | POST /api/v1/estimaciones-consumo, contrato 1.0.1, M2M, cálculo real | Implementado; sin URL cloud todavía |
| Un tipo de almacenamiento integrado | Catálogo MySQL en el flujo React → gateway → Consumo → Electrodomésticos → MySQL | Integración real verificada |
| Logs estructurados y correlacionados | JSON, trace_id, request_id y span_id en los tres procesos | Evidencia exportada |
| Primera traza distribuida | 8 spans y tres servicios en Jaeger | Evidencia exportada |
| D2, D6, D7, D9, D10, D12, D13 | ADR específicos en docs/adr | Documentados; distinguir decisiones futuras de implementación |
| Validación D1, D3, D5 | Propiedad exclusiva de base, repositorio probado, HTTP por lote con presupuesto y circuito | Validado en el flujo actual |
| Primera D11 | Logs JSON y trazas OTLP/Jaeger; métricas/dashboard definidos como próximos pasos | Primera versión implementada |
| D8 actualizado si cambia contrato | API compartida conserva versión, campos y fórmulas; metadatos de documentación actualizados | Sin cambio de interfaz M2M |
| Bonus anunciado | Playwright/Chromium en escritorio y móvil | 6 aprobados, 0 fallos, 0 omitidos |

No se contabilizan Usuarios ni Solar como servicios de negocio operativos. Los ADR D6/D7/D12 documentan la búsqueda especializada, caché y réplicas previstas para la final, sin atribuirlas a este arranque. D9 es provisional por falta del contrato ajeno; no se implementó ni se simuló una integración externa desconocida.

## Guion para demostrarlo

1. En la raíz, ejecutar `.\scripts\compose.ps1 up -d --build` (o `sh scripts/compose.sh up -d --build`). Abrir http://localhost:8080.
2. Agregar iluminación: cantidad 4, 5 horas/día, 30 días, factor 1. Agregar heladera: cantidad 1, 24 horas/día, 30 días, factor 0,3. Agregar lavarropas: cantidad 1, 12 ciclos en el período.
3. Calcular: **47.4 kWh** de período y **1.58 kWh/día**. Mostrar desglose y referencia de traza.
4. Cambiar cantidad: el resultado anterior desaparece. Recalcular. Recargar: el borrador se pierde y no hay historial anónimo.
5. Ejecutar `.\scripts\demo-m2m.ps1`: la capacidad para otro grupo usa el mismo algoritmo con entrada externa. No mostrar `.env` ni copiar su clave a la presentación.
6. Abrir http://localhost:16686 y buscar la traza del cálculo. Explicar el span SQL y la ausencia de acceso directo de Consumo a la base de otro servicio.
7. Ejecutar `.\scripts\verify.ps1` para reproducir tests Go, MySQL, HTTP y los seis tests de navegador. La primera descarga de imágenes puede tardar; preparar el equipo antes de exponer.

## Evidencia guardada

- [Respuesta real](evidence/entrega2/respuesta-real.json): escenario mixto, sin ids de evaluación persistida.
- [Traza distribuida](evidence/entrega2/traza-distribuida.json): traceId `dd645b366efd526afec437513d99995b`; servicios consumo, electrodomesticos, gateway; 8 spans con relaciones padre/hijo.
- [Logs correlacionados](evidence/entrega2/logs-correlacionados.jsonl): 3 operaciones con ese traceId, sin cuerpos ni credenciales.
- [Tests Go e integración MySQL](evidence/entrega2/tests-go.txt): incluye los tests de reglas, cliente, salud y dependencia real. La cobertura de application de Consumo se informa por paquete; no es cobertura global del proyecto.
- [Tests HTTP reales](evidence/entrega2/tests-http.txt): nueve casos, sin usar el mock como backend.
- [Reporte e2 e](evidence/entrega2/e2e/resultados.json): 6 tests exitosos, 0 inesperados, 0 inestables; duración 16.24 s.
- [Vista escritorio](evidence/entrega2/e2e/calculadora-escritorio.png) y [vista móvil](evidence/entrega2/e2e/calculadora-movil.png).
- [Ensayo de tráfico](evidence/entrega2/carga.json) y [muestra de recursos](evidence/entrega2/recursos.txt).
- [Caída y recuperación reales](evidence/entrega2/fallo-controlado.json), analizadas en [POSTMORTEM](POSTMORTEM.md).

## Capacidad observada y alcance de la medición

Doce solicitudes calientes a 0,5 req/s: {'200': 12}; p50 6.880 ms y p95 24.433 ms en loopback IPv4 local. Ráfaga de 40 con concurrencia 5: {'200': 10, '429': 30}. El límite observado es el token bucket de 1 req/s y ráfaga 10, no una saturación medida de CPU ni de MySQL. El percentil es descriptivo de una muestra breve; no certifica capacidad productiva ni un SLO. La API medida es cálculo por parámetros, por lo que no se atribuye su latencia a una consulta de catálogo.

## Qué queda para los siguientes hitos

Frontend completo con identidad/guardados/solar; tres o más servicios de negocio; segundo tipo de almacenamiento; eventos durables/outbox e idempotencia; Meilisearch; caché y comparación; contrato del proveedor e integración relevante; URL HTTPS pública; balanceo funcional; dashboard, alertas y capacidad sostenida. Ver arquitectura para deuda técnica y límites.

## Entrega al repositorio

El código y esta documentación se incorporan al repositorio local. La CI está preparada, pero su ejecución remota depende de subir los cambios. La integración a main y la entrega a la cátedra siguen el flujo de revisión del equipo. No se publicaron claves ni se contrató hosting.
