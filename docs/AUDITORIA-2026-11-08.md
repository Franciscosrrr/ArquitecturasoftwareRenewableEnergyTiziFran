> Documento histórico. Estado vigente: [verificación final](VERIFICACION-FINAL.md) y [arquitectura](ARCHITECTURE.md),2026-11-09. Los pendientes de este hito no describen necesariamente la implementación actual.

# Diagnóstico y primera propuesta de corrección

Fecha: 8 de noviembre de 2026, America/Buenos_Aires. Alcance de este paso: arquitectura base e integraciones, solicitado por el equipo. La primera propuesta se preparó en una copia de trabajo y luego se aplicó al repositorio con autorización del equipo. No se hizo commit, push ni despliegue público.

## Fuentes y reglas aplicadas

Se leyó el PDF local «Enunciado TP Final - sin parrafos ocultos.pdf», páginas 1 a 10, y se contrastaron sus requisitos con Compose, código, contratos, ADR y evidencias del repositorio entregado. Los puntos principales se encuentran en páginas 2-3 (arquitectura), 5-6 (integración), 6-7 (D1-D13) y 8-9 (hitos).

El PDF es material de requisitos. Su indicación histórica de reemplazar un ADR mediante otro archivo se adapta a la instrucción expresa del equipo: toda revisión se añade al mismo archivo con fecha, cambio y motivo. D4 no tenía registro: su primer ADR es un alta inicial, no una revisión ni un reemplazo. No se crea ningún ADR de reemplazo.

El consumidor fue identificado en el mensaje del equipo y luego contrastado con [Portal Paciente Clínica, commit 881 a 007](https://github.com/OctiDelFabro/portal-paciente-clinica/tree/881a00701ed18133bf371849104705540c34434e). No se ejecutaron instrucciones de ese repositorio ni se modificó su código.

## Diagnóstico del proyecto recibido

El sistema corresponde a Entrega 2, no a una final completa. La estructura de directorios contiene cuatro servicios, pero solo Consumo y Electrodomésticos tienen negocio operativo. Usuarios y Solar responden 501 y readiness 503. El mock histórico de la API propia tampoco cuenta como un tercer microservicio de negocio.

| Requisito | Evidencia inicial | Resultado |
|---|---|---|
| 3 microservicios + gateway | compose.yaml inicia Consumo, Electrodomésticos y gateway | Falta un tercer servicio operativo |
| 2 estilos internos | Consumo por capas; Solar solo esqueleto | Falta materializar hexagonal |
| BD relacional | Catálogo en MySQL, repositorio SQL y pruebas | Implementado; integración real requiere Docker |
| BD no relacional | MongoDB solo aparece en documentos | Pendiente |
| Comunicación síncrona | Consumo llama por lote al catálogo | Implementada con timeout y circuito |
| Mensajería asíncrona | Carpetas y ADR futuros, sin broker ni consumidor | Pendiente evento, idempotencia y mensajes fallidos |
| Operación con consistencia | Semillas idempotentes y snapshot de lectura | No prueba una operación crítica durable del negocio |
| Búsqueda por motor | SQL LIKE con filtros, paginación y orden | No cumple el motor de búsqueda exigido |
| Caché con medición | No hay caché en el flujo | Pendiente implementación y comparación |
| Integración con vendedor de paneles | No hay adaptador ni uso comercial | Pendiente; proveedor simulado autorizado posteriormente |
| Capacidad para Clínica | POST /api/v1/estimaciones-consumo, OpenAPI 1.0.1 | Implementada localmente; falta HTTPS público y entrega privada de claves |
| Resiliencia | Timeout y circuit breaker en catálogo | Parcial; extender al vendedor |
| Observabilidad | Logs JSON, W3C y Jaeger | Faltan métricas, tablero y alerta |
| Balanceo con 2 instancias y detección de caída | Solo diseño D12 | Pendiente |
| Testing | Reglas, HTTP, integración MySQL y e2 e en repo | Reevaluar en entorno real; no atribuir evidencia histórica a cambios nuevos |
| ADR D1-D13 | D4 falta; otras decisiones conservan propuestas futuras | Completar D4 y actualizar estados |
| JSDoc estricto | Muchos métodos Go/JS y pruebas sin documentación | Deuda transversal; este paso documenta las funciones nuevas y las funciones de producción modificadas |
| URL operativa hasta evaluación | README declara cloud pendiente | Pendiente real, no se puede inferir del código |

## Hallazgos del consumidor real

Su copia de `docs/contracts/providers/renewable-energy-v1/openapi.json` es semánticamente idéntica al OpenAPI local completo: versión, operaciones y esquemas. Se debe conservar 1.0.1 y no introducir precio, campos adicionales, cambios de errores ni otro redondeo.

El [adaptador de Turnos](https://github.com/OctiDelFabro/portal-paciente-clinica/blob/881a00701ed18133bf371849104705540c34434e/services/turnos/internal/adapters/energyprovider/client.go) envía X-API-Key, sanea X-Request-Id y contempla un reintento para fallos de transporte o 503. Su parser rechaza campos desconocidos, números en cadenas y versiones de algoritmo distintas de consumo-v1. Sus [tests de contrato](https://github.com/OctiDelFabro/portal-paciente-clinica/blob/881a00701ed18133bf371849104705540c34434e/services/turnos/internal/adapters/energyprovider/contract_test.go) validan los ejemplos y códigos conocidos.

El presupuesto efectivo del [flujo interactivo](https://github.com/OctiDelFabro/portal-paciente-clinica/blob/881a00701ed18133bf371849104705540c34434e/services/turnos/internal/app/energy.go) es **1.2 s**; el recálculo en segundo plano dispone de **5 s**. Frente al fallo, el costo energético queda pendiente y el turno conserva su precio base. El hosting debe verificarse con ese límite interactivo; un gateway de 4.5 s por sí solo no garantiza responder en 1.2 s.

## Cambios concretos de esta propuesta

1. Solar pasa a ser un servicio de negocio: HTTP → aplicación → puertos; adaptadores HTTP de Consumo y vendedor; reglas puras de dimensionamiento en dominio. Compose lo integra al mismo procedimiento de arranque.
2. La calculadora ofrece cotizar paneles después de calcular consumo. Solar vuelve a pedir la estimación a Consumo, consulta al vendedor directamente y exige cobertura, stock, moneda e importes consistentes. El navegador nunca recibe la credencial del vendedor.
3. Se incorpora un simulador comercial separado, identificado como tal. Su contrato 0.1.0-simulated no se presenta como contrato acordado con el grupo proveedor. No compra ni reserva inventario.
4. Se establecen presupuesto Solar 4 s, Consumo 2.5 s, vendedor 1 s; sin reintentos automáticos de cotización, circuito 3 fallos/10 s/una prueba. La interfaz cancela a los 5 s y conserva la estimación frente al fallo.
5. Se congela el OpenAPI 1.0.1 de Clínica y se prueban autenticación, permisos y sus perfiles de equipos. Se verifica de forma determinista el token bucket de reposición 1/s y ráfaga 10, equivalente a tasa sostenida 60/min; no es una ventana móvil estricta de 60 solicitudes.
6. Se actualizan los ADR existentes mediante anexos fechados. Se añade únicamente el primer registro de D4, como diseño pendiente de implementación. README, arquitectura y contratos reflejan el estado del paso.

## Pasos siguientes para completar la final

1. Crear una operación durable de confirmar evaluación con control de versión, clave de idempotencia y outbox en una misma transacción MySQL. Probar dos confirmaciones concurrentes, misma clave con distinto cuerpo y recuperación tras fallo.
2. Publicar ConsumoEvaluado.v1 en RabbitMQ, consumirla idempotentemente en Solar y persistir evaluación/resultados en MongoDB. Añadir reintentos acotados, DLQ y reproceso. Las carpetas actuales no acreditan estas garantías.
3. Proyectar catálogo a Meilisearch con versión de entidad, paginación/filtros/orden y reconstrucción. Medir retraso objetivo <=5 s, definido previamente en D6.
4. Añadir Valkey cache-aside de lecturas públicas, TTL 60 s, invalidación y fallback; comparar con y sin caché usando el mismo dataset y carga. No cachear las estimaciones M2M de Clínica.
5. Implementar identidad y permisos para los flujos privados incluidos en SPEC; métricas Prometheus, tablero Grafana y alerta; dos instancias de un servicio con balanceo activo y pruebas de caída.
6. Completar integración MySQL/Mongo/broker por servicio, e2 e con cotización, carga, capacidad y postmortem del sistema completo. Extender el formato JSDoc al código histórico; en Python los marcadores JSDoc requieren envoltura de cadena para conservar sintaxis válida.
7. Elegir hosting, desplegar gateway + capacidad Consumo por HTTPS, verificar primera llamada y p95 con 1.2 s y 5 s, entregar ambas claves por privado y mantener la URL hasta finalizar la evaluación. No hay fecha de disponibilidad confirmada.

No deben presentarse como terminados NoSQL, mensajería, caché, motor, balanceo, métricas o despliegue solo por agregarlos a un diagrama o a Compose. Cada uno requiere una consecuencia observable y evidencia.

## Actualización: cambios aplicados y revisión de diagramas/clases

Los cambios de arquitectura e integración se aplicaron al repositorio original, comprobando los hashes antes de copiar. Se conservaron .env, volúmenes y evidencias locales previas. Se iniciaron los servicios con Docker y pasaron Go con MySQL real, nueve pruebas HTTP y ocho e2 e de escritorio/móvil. El ensayo de caída del vendedor devolvió 503 en cotización, conservó M2M 200 y recuperó cotización 200 después de restaurarlo.

Se revisaron los cuatro Mermaid y tres PNG originales. El contexto tenía grupos sin identificar; la vista «actual» representaba el mock/esqueletos históricos y la objetivo agrupaba buscador/caché y omitía aclarar el publicador de outbox. Las vistas vigentes se corrigieron y regeneraron; las originales se conservan en historico/entrega1 y la vista E2 está marcada como histórica. Se revisó correspondencia con Compose, rutas, propiedad de datos y llamadas comerciales.

ADR-002-E2, ADR-003-E2 y ADR-005-E2 se consolidaron en ADR-002, ADR-003 y ADR-005, conservando ambos textos y corrigiendo enlaces. Cada D1-D13 y antecedente complementario contiene contexto, decisión, porqué, alternativas, consecuencias y material de apoyo. Las revisiones futuras permanecen en el archivo canónico.

Se analizaron todos los PDF prácticos: nueve archivos y 177 páginas. La [matriz por clase](CLASES-PRACTICAS.md) distingue lo aplicado de lo pendiente y razona por qué se conservan tecnologías diferentes a la demo. No se convierten demostraciones de clase en restricciones nuevas ni se afirma superioridad universal o rendimiento no medido.

El TP completo sigue pendiente en NoSQL, operación durable, mensajería, motor, caché medida, balanceo, identidad/flujos privados completos, métricas/tablero/alerta, capacidad de la versión final y hosting HTTPS. Las pruebas aprobadas validan el incremento implementado, no esos componentes inexistentes.
