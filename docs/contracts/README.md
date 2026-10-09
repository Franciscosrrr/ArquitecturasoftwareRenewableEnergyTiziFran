# Capacidad pública: estimación de consumo

Proyecto: ArquitecturasoftwareRenewableEnergy. Proveedor: servicio Consumo, publicado a través del gateway. Contrato 1.0.1; ruta mayor /v1. Estado: contrato con mock local ejecutable; servicio productivo y URL cloud pendientes.

## Contrato formal

[OpenAPI 3.0.3 en JSON](v1/openapi.json), formato estándar procesable. Las reglas cruzadas y el redondeo descritos aquí son parte normativa del contrato. La especificación se puede importar en herramientas compatibles; no depende de instalar una herramienta para leerla.

## Utilidad y límites

Otro grupo envía equipos y hábitos para utilizar el consumo calculado en una cotización, planificación u otro flujo importante. La API acepta parámetros externos, sin exigir ids del catálogo ni acceso a lugares privados. No devuelve paneles ni guarda configuraciones/evaluaciones. La autenticación de integración es independiente del login de la web.

## Operación y autenticación

POST /api/v1/estimaciones-consumo. Content-Type: application/json. Header X-API-Key: credencial M2M que el equipo proveedor entregará por un canal privado. X-Request-Id es opcional para correlación, con hasta 64 letras, números, guion o guion bajo; no es clave de idempotencia.

Mock local: http://localhost:8080 después de ejecutar Compose; no es una URL productiva. URL HTTPS cloud: pendiente de publicar. No se incluyen claves reales. Un cliente sin clave válida recibe 401; autenticado sin permiso estimar-consumo recibe 403. El gateway aplicará inicialmente 60 solicitudes/minuto por cliente con ráfaga de hasta 10, ajustable mediante acuerdo documentado. Máximo cuerpo: 256 KiB; máximo 100 equipos.

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

Presupuesto propuesto con instancia activa: 5 s; ante fallo de conexión o 503, una repetición con jitter dentro del presupuesto. No repetir 4xx de validación/permiso. Si se acuerda Render Free, la primera activación puede exceder ese presupuesto: acordar un sondeo de disponibilidad separado antes del flujo o mostrar indisponibilidad temporal y reintento explícito. No se simula un éxito para ocultar cold start.

## Compatibilidad y verificación

Cambios de significado, unidades, fórmula, campos obligatorios o errores incompatibles requieren /v2 y transición acordada. La semántica de versionAlgoritmo y las reglas del cálculo no se cambian silenciosamente dentro de v1. Se conserva v1 hasta que el consumidor pueda migrar. Los esquemas actuales rechazan campos desconocidos. Cualquier campo nuevo de respuesta requiere un contrato acordado y la actualización de los consumidores antes de enviarlo; una extensión no acordada no se considera compatible con estos esquemas.

El proveedor verificará implementación contra OpenAPI, ejemplos y casos límite; el grupo consumidor tendrá tests que detecten incompatibilidades. La [guía de mock](MOCK.md) explica arranque, credenciales de prueba y escenarios ejecutables. La asignación del consumidor y su caso concreto todavía no fueron comunicados.

## Distinción respecto del modo invitado web

Este contrato es la integración M2M de otro grupo y conserva X-API-Key. La calculadora web no exige registro ni dicha clave: usa rutas anónimas separadas de consumo y recomendación solar, documentadas en [GUEST-MODE.md](../GUEST-MODE.md) y [OpenAPI invitado](guest-v1/openapi.json). No se elimina seguridad M2M ni se expone una credencial en React al habilitar visitas sin cuenta.

## Revisión compatible 1.0.1

Aclara que el mock local ya es ejecutable y que la implementación productiva sigue pendiente. El contrato invitado retira una cota de salida de superficie que rechazaba resultados matemáticos de solicitudes válidas; no cambian campos, rutas, fórmulas ni unidades. Los errores de validación del mock ahora señalan el campo observado.
