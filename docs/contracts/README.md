# Capacidad pública: estimación de consumo

Proyecto: ArquitecturasoftwareRenewableEnergy. Proveedor: servicio Consumo, publicado a través del gateway. Contrato 1.0.1; ruta mayor /v1. Estado vigente 2026-11-09: implementación real Go desplegada en Render Free, https://renewable-consumo-v1.onrender.com, con contrato congelado y credenciales propias; el túnel temporal de la PC sigue como alternativa. El mock histórico sigue disponible por separado.

## Contrato formal

[OpenAPI 3.0.3 en JSON](v1/openapi.json), formato estándar procesable. Las reglas cruzadas y el redondeo descritos aquí son parte normativa del contrato. La especificación se puede importar en herramientas compatibles; no depende de instalar una herramienta para leerla.

## Utilidad y límites

Otro grupo envía equipos y hábitos para utilizar el consumo calculado en una cotización, planificación u otro flujo importante. La API acepta parámetros externos, sin exigir ids del catálogo ni acceso a lugares privados. No devuelve paneles ni guarda configuraciones/evaluaciones. La autenticación de integración es independiente del login de la web.

## Operación y autenticación

POST /api/v1/estimaciones-consumo. Content-Type: application/json. Header X-API-Key: credencial M2M que el equipo proveedor entregará por un canal privado. X-Request-Id es opcional para correlación, con hasta 64 letras, números, guion o guion bajo; no es clave de idempotencia.

Sistema real local: http://localhost:8080 después de ejecutar Compose. Mock histórico opcional: http://localhost:8081 con el perfil mock; no es una URL productiva. URL HTTPS actual desde la PC: https://emails-bingo-preference-packets.trycloudflare.com. URL cloud verificada: https://renewable-consumo-v1.onrender.com; usar las claves específicas de Render. No se incluyen claves reales. Un cliente sin clave válida recibe 401; autenticado sin permiso estimar-consumo recibe 403. El gateway aplica 60 solicitudes/minuto por cliente con ráfaga de hasta 10, ajustable mediante acuerdo documentado. Máximo cuerpo: 256 KiB; máximo 100 equipos.

## Entrada

| Campo | Regla |
|---|---|
| diasPeriodo | Entero de 1 a 366, obligatorio; la UI puede proponer 30 |
| equipos | Entre 1 y 100 objetos, obligatorios |
| id | Identificador propio del consumidor, 1–64 letras/números/_/-, sin repetidos |
| cantidad | Entero de 1 a 1000 |
| modo | POTENCIA o CICLOS, excluyentes; sin campos de la otra variante |
| potenciaW | En POTENCIA: número >0 y ≤1.000.000 W |
| horasPorDia | En POTENCIA: número entre 0 y 24 por día de uso |
| diasUso | En POTENCIA: entero entre 0 y diasPeriodo |
| factorFuncionamiento | En POTENCIA: número >0 y ≤1, explícito; usar 1 si la potencia ya es media |
| energiaPorCicloKWh | En CICLOS: número >0 y ≤1000 kWh por ciclo |
| ciclosPeriodo | En CICLOS: entero de 0 a 10.000 por unidad del equipo, durante todo el período |

Se rechazan campos desconocidos y valores nulos. Los números de entrada admiten como máximo seis decimales; no se redondean silenciosamente valores más precisos. Los límites técnicos evitan solicitudes sin cota y no representan una certificación eléctrica. id único, precisión decimal y diasUso <= diasPeriodo se validan como reglas de negocio además del esquema.

## Resultado y fórmulas

POTENCIA: kWh = potenciaW × cantidad × horasPorDia × diasUso × factorFuncionamiento / 1000. CICLOS: kWh = energiaPorCicloKWh × cantidad × ciclosPeriodo. Total = suma sin redondeos intermedios; promedio = total / diasPeriodo.

La respuesta 200 contiene desglose en el mismo orden de entrada, totalPeriodoKWh, promedioDiarioKWh, diasPeriodo, supuestos, versionAlgoritmo=consumo-v1 y traceId. Se calculan valores con precisión decimal y se redondea cada salida a seis decimales, mitad hacia arriba. No se promete igual cantidad de ceros finales en JSON. La suma de salidas redondeadas puede diferir del total redondeado por hasta una pequeña diferencia de presentación; el total usa valores internos sin redondear.

Misma entrada y algoritmo producen igual energía. traceId es metadato variable. La API no crea efectos de negocio persistentes, por lo que una repetición no duplica lugares ni evaluaciones; no requiere Idempotency-Key. Esto difiere de la operación interna de confirmar una evaluación guardada, que sí requiere idempotencia.

## Ejemplo comprobable

[Solicitud válida](v1/examples/solicitud-valida.json) y [respuesta 200](v1/examples/respuesta-200.json): 30 días, iluminación 6 kWh, heladera 32,4 kWh y lavarropas 9 kWh. Total 47,4 kWh; promedio 1,58 kWh/día. Es un escenario ilustrativo, no una medición ni una ficha universal.

[Solicitud inválida](v1/examples/solicitud-invalida-dias.json): declara 31 días de uso para un período de 30. Debe responder [422](v1/examples/respuesta-422.json) con el campo observado. Los ejemplos restantes de errores están en v1/examples y en OpenAPI.

## Errores y reintentos

| HTTP | Código estable | Interpretación |
|---|---|---|
| 400 | FORMATO_INVALIDO | JSON mal formado o framing inválido |
| 401 | NO_AUTENTICADO | Credencial ausente/incorrecta; corregir antes de repetir |
| 403 | SIN_PERMISO | Falta de permiso del cliente |
| 413 | CUERPO_DEMASIADO_GRANDE | Excede 256 KiB |
| 415 | TIPO_CONTENIDO_NO_SOPORTADO | Tipo diferente de application/json |
| 422 | VALIDACION | Esquema, tipos, rangos, ids repetidos o regla cruzada inválidos |
| 429 | LIMITE_DE_TRAFICO | Esperar Retry-After obligatorio |
| 500 | ERROR_INTERNO | Falla inesperada; diagnóstico privado por traceId |
| 503 | SERVICIO_NO_DISPONIBLE | Indisponibilidad temporal; Retry-After si se conoce |

Todos los errores de estas operaciones de negocio siguen codigo, mensaje, traceId y detalles. No exponen SQL, stack traces ni credenciales. Una plataforma cloud/proxy puede responder por sí misma durante cold start o antes de llegar a la aplicación; el consumidor debe validar tipo de contenido y tolerar una respuesta de infraestructura ajena al esquema, sin tratarla como 200 de negocio.

Presupuesto del reestimador de Clínica con instancia activa: 5 s; su flujo interactivo dispone de 1.2 s. Ante fallo de conexión o 503 en segundo plano, puede realizar una repetición con jitter dentro del presupuesto. No repetir 4 xx de validación/permiso. Si se acuerda Render Free, la primera activación puede exceder ese presupuesto: acordar un sondeo de disponibilidad separado antes del flujo o mostrar indisponibilidad temporal y reintento explícito. No se simula un éxito para ocultar cold start.

## Compatibilidad y verificación

Cambios de significado, unidades, fórmula, campos obligatorios o errores incompatibles requieren /v2 y transición acordada. La semántica de versionAlgoritmo y las reglas del cálculo no se cambian silenciosamente dentro de v1. Se conserva v1 hasta que el consumidor pueda migrar. Los esquemas actuales rechazan campos desconocidos. Cualquier campo nuevo de respuesta requiere un contrato acordado y la actualización de los consumidores antes de enviarlo; una extensión no acordada no se considera compatible con estos esquemas.

El proveedor verificará implementación contra OpenAPI, ejemplos y casos límite; el grupo consumidor tendrá tests que detecten incompatibilidades. La [guía de mock](MOCK.md) explica arranque, credenciales de prueba y escenarios ejecutables. El consumidor es Portal Paciente Clínica: consulta energía por equipos y calcula el costo con su tarifa. Ver la actualización 2026-11-08 y su presupuesto interactivo de 1.2 s.

## Distinción respecto del modo invitado web

Este contrato es la integración M2M de otro grupo y conserva X-API-Key. La calculadora web no exige registro ni dicha clave: usa rutas anónimas separadas de consumo y recomendación solar, documentadas en [GUEST-MODE.md](../GUEST-MODE.md) y [OpenAPI invitado](guest-v1/openapi.json). No se elimina seguridad M2M ni se expone una credencial en React al habilitar visitas sin cuenta.

## Revisión compatible 1.0.1

La revisión histórica 1.0.1 aclaró la disponibilidad del mock; la implementación Go ahora está operativa localmente y la publicación HTTPS sigue pendiente. El contrato invitado retira una cota de salida de superficie que rechazaba resultados matemáticos de solicitudes válidas; no cambian campos, rutas, fórmulas ni unidades. Los errores de validación del mock ahora señalan el campo observado.

## Implementación real de entrega 2

El arranque predeterminado usa gateway y Consumo Go, no el mock. Las credenciales M2M se generan aleatoriamente en el .env local excluido de Git. `scripts/demo-m2m.ps1` invoca el ejemplo sin imprimir la clave. No usar `mock-consumidor` en el sistema real. M2M_DENIED_KEY es una segunda credencial de demostración sin el permiso estimar-consumo, utilizada para verificar 403.

POST /api/v1/estimaciones-consumo aplica el mismo caso de uso que las rutas temporales web. No necesita MySQL para la entrada externa ni modifica datos de negocio. Se ejecutaron pruebas de contrato sobre sus respuestas reales y ejemplos. Ninguna de esas pruebas pretende sustituir el test del contrato que todavía no recibimos del proveedor asignado.

La traza usa un traceId W3C de 32 caracteres hexadecimales. X-Request-Id, si se suministra válido, se conserva como referencia del consumidor; ambos se registran en logs. El cliente no debe asumir que son iguales. El parser normaliza ceros y exponentes antes de crear los racionales y rechaza los valores fuera del rango o con más de seis decimales efectivos.

El [contrato adicional del catálogo](catalog-v1/openapi.json) define lectura pública y POST /api/v1/publico/consumo-catalogo. Es una API separada: usa ids propios, resuelve la potencia en MySQL y no acepta potencias enviadas por el navegador. No cambia la entrada M2M externa ni exige claves en React.

## Antecedente de coordinación - 2026-11-08: Portal Paciente Clínica

Esta sección actualiza las referencias históricas a consumidor desconocido. El consumidor es Portal Paciente Clínica. Utiliza la energía de equipos por consulta médica y aplica su propia tarifa en pesos; esta API no agrega precio. Se contrastó su copia del contrato en el commit 881 a 00701 ed18133 bf371849104705540 c34434 e: OpenAPI 1.0.1 semánticamente idéntico al nuestro. No se modifica v1/openapi.json, sus códigos, campos ni redondeo.

Su flujo interactivo tiene un presupuesto efectivo de 1.2 s; el reestimador en segundo plano, 5 s. Su parser es estricto con campos adicionales y tipos numéricos. Si el proveedor falla, muestra costo pendiente y vuelve a estimar después. El plazo de 5 s del mensaje inicial no describe por sí solo toda la experiencia del consumidor.

Tasa confirmada por implementación y test: token bucket de reposición 1 solicitud/s (60/min sostenidas), capacidad de ráfaga 10, 429 y Retry-After: 1. No equivale a una ventana móvil estricta de 60 solicitudes. La cuota es local por proceso; mantener una única instancia M2M hasta incorporar almacenamiento compartido de límites. Gateway limita por hash de X-API-Key; Consumo valida la credencial y aplica su límite conservador.

Pendientes de coordinación real antes del miércoles 11 de noviembre de 2026: URL HTTPS pública y fecha de disponibilidad; clave M2M con estimar-consumo y clave sin permiso por canal privado; hosting elegido y comportamiento de primera activación; confirmación operativa hasta terminar la evaluación. No hay URL ni fecha publicadas, y no se afirma un plan Render contratado. No incluir claves en Git, ejemplos ni frontend.

Claves locales: los scripts existentes generan M2M_API_KEY y M2M_DENIED_KEY en .env. Son credenciales del entorno local; no se entregaron al otro equipo desde esta tarea. La prueba TestClinicaV101 Compatibility congela el OpenAPI recibido y TestClinicaEquipmentProfiles comprueba equipos reales de prueba de Clínica. El test propio no sustituye la ejecución desde su despliegue contra nuestra futura URL.

Ejemplo de cardiología de 30 minutos: 4 luminarias de 18 W, aire de 1100 W con factor 0.5, PC de 120 W y un ciclo de electrocardiógrafo de 0.01 kWh: 0.381 kWh. El costo se calcula en Clínica, con su tarifa.

La API web de [cotización solar](solar-v1/openapi.json) y el [vendedor simulado](paneles-sim-v1/openapi.json) tienen contratos separados. No alteran la capacidad M2M 1.0.1. Su arranque y migración futura se explican en [integración solar](SOLAR.md).


## Implementación completa — 2026-11-09

El contrato compartido M2M 1.0.1 y todos sus esquemas/errores/redondeo permanecen intactos. Las operaciones con efectos de la web se formalizan por separado en [API privada 1.1.0](private-v1/README.md) y [eventos1.0.0](events/v1/README.md). La extensión [vendedor simulado por SKU](paneles-sim-v2/openapi.json)0.2.0-simulated mantiene v1 disponible; no reemplaza ningún contrato real del grupo proveedor.

La [publicación HTTPS actual](../PUBLICACION-CLINICA.md) usa un túnel desde la PC; el [proceso M2M en Render](../DESPLIEGUE-GRATUITO.md) está desplegado y verificado por separado, con nuevas claves. No se publican claves. Clínica mantiene costo/tarifa propios; no se agrega precio a su respuesta. Las decisiones de contrato no se cambian por ampliar la aplicación privada.

## Ampliación de la interfaz — 2026-11-09

La búsqueda pública de [paneles solares 1.0.0](panel-search-v1/openapi.json) y las operaciones privadas de archivo/eliminación [1.1.0](private-v1/openapi.json) tienen contratos separados. No alteran la capacidad M2M 1.0.1.


## Responsabilidad de la prueba del consumidor — 2026-11-09

Por indicación del usuario, Clínica realizará la prueba desde su propio backend. El proveedor conserva el contrato 1.0.1, ejemplos, errores, claves dedicadas y cuota pactada; no ejecuta ni afirma haber ejecutado esa validación en nombre de Clínica. Los ensayos del proveedor y su evidencia HTTPS se mantienen separados. Un futuro cambio de hostname cloud se comunicará con su estado de cold start, sin modificar campos, fórmulas ni redondeo.


La publicación cloud Render conserva el OpenAPI 1.0.1 byte a byte normalizando finales de línea; su base URL se configura externamente en el consumidor. No se modifica servers ni otro campo del contrato congelado para cambiar el hostname. La prueba desde Clínica es responsabilidad de su grupo; la sonda del proveedor no la sustituye.
