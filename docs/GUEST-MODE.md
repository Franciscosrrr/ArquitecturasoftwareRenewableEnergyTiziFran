# Modo invitado: calculadora sin cuenta

Versión documental 1.2.0. Diseño previsto; implementación pendiente.

## Experiencia de uso

La página abre directamente la calculadora. Se seleccionan electrodomésticos precargados y hábitos; se muestra consumo por equipo, total y promedio. Después se indican cobertura objetivo, superficie opcional y recurso solar manual o ubicación para consultar el proveedor disponible. Se muestran modelo/cantidad de paneles y alternativas con los mismos criterios del modo autenticado. Registrar/login es opcional hasta pulsar Guardar.

Un cambio de parámetros permite recalcular. El borrador y las respuestas viven solo en memoria React. No se usan cookies de negocio, localStorage, sessionStorage ni IndexedDB para persistirlos. Recargar/cerrar elimina la simulación. La interfaz informa esta condición antes de que el visitante cierre o recargue cuando el navegador lo permita; no se promete impedir el cierre.

## Rutas y autorización

| Operación | Acceso | Responsable | Persistencia de negocio |
|---|---|---|---|
| Consultar catálogo activo de equipos/paneles | Anónimo; lectura limitada | Electrodomésticos / Solar | Solo lectura de catálogo |
| POST /api/v1/publico/consumo | Anónimo | Consumo | Ninguna |
| POST /api/v1/publico/recomendaciones-solares | Anónimo | Solar | Ninguna |
| Guardar/editar lugares y configuraciones, confirmar evaluaciones e historial | JWT y propietario | Consumo / Solar | Sí, datos propios |
| Mutaciones de catálogos/roles | JWT y administrador vigente | Servicio dueño | Sí |
| POST /api/v1/estimaciones-consumo | X-API-Key M2M | Consumo | Ninguna; contrato de otro grupo |

Contrato de cálculo web: [OpenAPI invitado](contracts/guest-v1/openapi.json). Sus DTO de consumo usan las mismas reglas/validaciones que el cálculo M2M, pero el navegador no requiere ni recibe una clave M2M. La interfaz selecciona datos del catálogo activo; enviar parámetros para simular no crea un electrodoméstico ni habilita altas de catálogo. El servidor valida tipos, cantidades, rangos y reglas cruzadas. No confía en totales del cliente como evaluación persistida.

Las lecturas anónimas de catálogo se limitan a GET /api/v1/publico/electrodomesticos, GET /api/v1/publico/electrodomesticos/{id}, GET /api/v1/publico/paneles y GET /api/v1/publico/paneles/{id}. Solo devuelven fichas activas y campos públicos; no exponen auditoría ni listado de inactivos. Los listados admiten búsqueda/filtros/orden previstos, página de 20 y máximo 100 por petición. Una ficha no visible responde 404. Las rutas existentes de mantenimiento administrativo siguen protegidas, aunque compartan repositorios de lectura.

## Flujo temporal y límites

1. Frontend consulta fichas públicas activas y conserva equipos/hábitos en memoria.
2. Consumo valida y calcula la solicitud; responde desglose, total, promedio, versión y supuestos. No crea filas en consumo_db ni outbox; no publica ConsumoEvaluado.
3. Frontend envía promedio y parámetros solares a Solar. Este promedio es una entrada de simulación validada por rango, no una referencia a una evaluación guardada.
4. Solar lee paneles activos y obtiene HSP desde entrada manual etiquetada o ubicación/proveedor. Captura fichas/valores/versiones en memoria y usa el dominio común para comparar. Devuelve una respuesta síncrona sin estudioId, trabajo, lease, evento ni documento persistido.
5. Se muestran ambas respuestas. Si Solar falla, se conserva el consumo en memoria y se ofrece reintentar. Sin paneles factibles responde sin alternativa; no se inventa cumplimiento.

Body máximo 256 KiB; máximo 100 equipos y demás límites del esquema de consumo. Límite inicial por origen: 60 cálculos/minuto con ráfaga 10, y 10 recomendaciones/minuto con ráfaga 2. Gateway mantiene contadores temporales de abuso; no guarda cuerpos/resultados ni registra la IP como identidad de negocio. Cabeceras de rate limit/proxy se aceptan solo desde proxies confiables. 429 devuelve Retry-After. CORS acepta la interfaz prevista; no se presenta CORS como protección suficiente de una ruta pública.

Deadline de gateway 10 s y presupuesto solar 8 s incluyendo acceso a paneles/proveedor y comparación; el adaptador externo tiene un subpresupuesto de 6 s, hasta dos intentos de 2,5 s, conexión incluida; cancelar contexto propaga cancelación. No se encola una solicitud que no termine: 503 recuperable o error 422 por entrada inválida. El cliente reintenta explícitamente. Se limitan recomendaciones simultáneas por instancia a 8 como valor inicial a medir. El volumen de datos de paneles se filtra/pagina internamente y se retorna un máximo de 20 alternativas ordenadas, con total de alternativas y aviso de truncamiento. Nunca se confirma que 20 sea el total si hubo más.

El recurso manual exige HSP >0 y <=24, rendimientoGlobal >0 y <=1 y fuente indicada; el resultado lo etiqueta como declaración manual. Por ubicación se valida latitud/longitud y se consulta al proveedor disponible; el dato y su fuente se muestran, sin asumir un valor genérico. Si no hay fuente confiable se informa indisponibilidad. Aplican las fórmulas, redondeo y selección de [SPEC](../SPEC.md), incluido consumo cero → cero paneles y ausencia de candidatos → sin alternativa.

## Qué significa no guardar

No se crean usuarios anónimos, lugares, configuraciones, evaluaciones, estudios solares, snapshots históricos, outbox ni mensajes de negocio para visitantes. No se cachean solicitudes, equipos seleccionados, ubicación ni resultados personales. Se permiten lecturas de las BD de catálogo y caché/índice de fichas públicas compartidas; no son consumos guardados.

Logs/trazas solo contienen ruta, estado, duración y traceId técnico para este flujo; no cuerpos, parámetros, coordenadas, resultados ni ids de negocio inexistentes. Métricas son agregadas. Los contadores de abuso expiran y solo contienen información técnica mínima. Esto distingue la ausencia de persistencia del consumo de la observabilidad operativa necesaria.

## Pasar a cuenta y guardar

Guardar sin sesión abre registro/login, manteniendo el borrador en memoria mediante un diálogo/ruta que no recarga ni desmonta la calculadora. Si se recarga, se pierde como cualquier borrador invitado. Autenticarse no crea información de consumo. Al confirmar Guardar, Consumo vuelve a validar las fichas activas, asigna propietario y persiste lugar/configuración con nombre. Si hay cambios, la interfaz muestra diferencias y pide confirmar el escenario actualizado. La evaluación persistente se recalcula en backend con idempotencia y outbox; Solar procesa su estudio durable. La respuesta temporal no se inserta como un histórico confiable.

## Impacto arquitectónico y aceptación pendiente

Se mantienen Usuarios, Electrodomésticos y Consumo en capas y Solar hexagonal, con BD propia. Se añaden handlers/casos temporales y puertos de solo lectura en Solar. Usuarios/consumo_db/RabbitMQ no bloquean cálculo anónimo; las rutas invitadas tienen readiness por capacidad. El catálogo solar y proveedor, si se usa, sí son dependencias necesarias. Las réplicas Solar atienden ambos flujos y siguen compartiendo su catálogo/BD propia; el temporal no escribe allí.

La publicación gratuita prevista de la capacidad M2M stateless no implica que el sitio completo ni la ruta solar estén ya desplegados en cloud. El modo invitado forma parte del sistema local completo arrancado con Docker.

Pruebas futuras: navegación sin cuenta, misma aritmética en ambos modos, 401 en rutas privadas sin sesión, 403 administrativo, ausencia de escrituras/eventos, pérdida al recargar, login sin guardado automático, guardado confirmado con propietario, timeout solar sin perder consumo visible y 429. Esta entrega también prueba el mock de las rutas de cálculo. No se atribuyen esas pruebas a la interfaz React, la persistencia ni a un proveedor real.
