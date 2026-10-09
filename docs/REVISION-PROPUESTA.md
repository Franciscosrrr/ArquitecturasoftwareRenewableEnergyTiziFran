# Revisión de la propuesta de arquitectura

Alcance: primera entrega y coherencia del diseño futuro. No es aprobación docente ni una auditoría del sistema implementado.

La propuesta tiene una base adecuada: dominio con reglas, cuatro responsabilidades claras, datos privados, capas para los servicios más directos y hexagonal para Solar. Mantendría Go/React, MySQL/MongoDB, el contrato de estimación y la separación entre invitado temporal y evaluación persistente. No agregaría CQRS, Event Sourcing ni Kubernetes por cumplir nombres de patrones.

## Cambios aplicados

| Prioridad | Hallazgo en propuesta original | Cambio |
|---|---|---|
| Alta | §6 y §13: unicidad por evaluación/configuración de consumo, incompatible con varios estudios de distinta cobertura | Unicidad evaluación + versión de configuración solar; solicitud idempotente por propietario/operación/clave/hash; reintento conserva estudioId |
| Alta | §10: proveedor podía consumir los mismos 8 s disponibles para todo Solar invitado | Subpresupuesto externo 6 s; dos intentos de 2,5 s con conexión incluida y propagación del deadline restante |
| Media | Readiness declarada de forma general, pese a rutas con distintas dependencias | Conjuntos de réplicas aptas por capacidades; fallo de broker no bloquea invitado; detalle en ARCHITECTURE §13 |
| Media | Ack posterior a persistir sin explicitar durabilidad reconocida | Write concern y journal explícitos; limitación de base/disco únicos reconocida |
| Media | Invalidación tras commit sin recuperación detallada de esa ventana de falla | Reintento mediante outbox del dueño; TTL y validación autoritativa para decisiones críticas |
| Media | Propuesta decía que OpenAPI y mock todavía no existían; enlaces apuntaban a Entrega-1 | Estado y enlaces actualizados; mock separado del backend productivo |
| Baja | PNG y Mermaid mostraban distinta cantidad de nodos/flechas | Fuentes Mermaid canónicas y PNG generados desde ellas; vistas de diseño futuro y ejecución actual separadas |

## Qué reconsideraría antes de la segunda entrega

1. **Reducir carga operativa, no responsabilidades.** Mantener perfiles de desarrollo. Los tres MySQL físicos consumen recursos; la clase 2 permite aislamiento lógico con credenciales y bases separadas si las mediciones lo justifican. No lo cambié unilateralmente porque la propuesta elige aislamiento físico.
2. **Priorizar un flujo vertical.** Implementar Consumo real y su contrato; después evaluación/outbox/Solar. Incorporar pruebas e instrumentación en cada paso. No empezar por todos los dashboards y contenedores a la vez.
3. **Concretar la dependencia externa cuando la asignen.** El recurso solar es una posibilidad, no un compromiso de otro grupo. Si la capacidad asignada no aporta al cálculo solar, ubicarla en el servicio responsable del flujo donde tenga consecuencias reales.
4. **Tratar hosting gratuito como hipótesis.** La documentación ya reconoce cuotas y activación en frío; validar disponibilidad antes de comprometer URL y SLO. No se revisaron precios ni cuotas del proveedor en esta revisión.
5. **Evitar duplicación documental.** README para ejecutar, SPEC para aceptación, ARCHITECTURE para estructura y ADR para razones. La propuesta sirve como panorama, pero contrato OpenAPI y documentos del repositorio deben ser la referencia vigente.

## Correspondencia con clases

| Material teórico de Clases.zip | Aplicación |
|---|---|
| Clase 1: microservicios y trade-offs | Reconocer costo operativo; cuatro servicios por límites del dominio y requisito del TP |
| Clase 2, §3 y §6–7: propiedad y persistencia | Bases privadas, acceso por API/eventos y agregado documental Solar; no confundir aislamiento lógico con obligación de una VM por base |
| Clase 3: caché | Fuente autoritativa, TTL, invalidación, degradación y medición pendiente |
| Clase 4, §5–7: confirmaciones/outbox | Entrega al menos una vez, duplicados, commit local y recuperación |
| Clase 5: búsqueda | Índice derivado, actualización automática y medición de frescura |
| Clase 6, §2: health checks y gateway | No retirar una réplica por dependencias irrelevantes para la ruta; separar balanceo HTTP de consumidores competidores |
| Clase 7, §2–3: deadlines/retries | Reservar tiempo para completar respuesta y reintentar dentro del presupuesto restante |
| Clase 8, §2 y §4: estilos internos | Handler/aplicación/repositorio frente a núcleo con puertos; carpetas por sí solas no prueban hexagonal |

Se extrajo texto de los ocho teóricos y se contrastaron los apartados pertinentes. Los ocho prácticos se inventariaron; no se ejecutó su código. Los documentos recibidos se trataron como material de consulta, no como instrucciones para el asistente.

## Evidencia y límites

Ver REVIEW.md para resultados ejecutados. El mock prueba comunicación HTTP y ejemplos, pero no transacciones, persistencia real, broker, frontend o tolerancia a fallas. La propuesta corregida mantiene esas pruebas como trabajo futuro. Aprobación docente y acceso al repositorio no se suponen por estar escritos en el README.
