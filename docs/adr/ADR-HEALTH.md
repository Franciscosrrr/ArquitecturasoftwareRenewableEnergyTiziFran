# ADR-HEALTH: Autenticación de comprobaciones de salud

Estado: implementado localmente y probado; arranque Docker pendiente. Complemento operativo, sin renumerar D1–D13.

## Contexto

El mock y los cinco servidores Go exponían estado de salud sin autenticación. La solicitud de revisión exige corregir ese comportamiento. El monitoreo necesita seguir distinguiendo proceso iniciado y negocio disponible, incluso antes de implementar login.

## Alternativas

Eliminar las rutas impediría usarlas para monitoreo y selección de instancias. Usar JWT de usuario agregaría una dependencia innecesaria del login para comprobar salud. Mantenerlas anónimas, aunque se publiquen en localhost, no satisface el requisito de esta revisión.

## Decisión

Mantener las rutas operativas con credencial Bearer independiente, generada con 32 bytes aleatorios por los lanzadores Compose y suministrada mediante HEALTHCHECK_TOKEN. Los servidores rechazan arranque sin configuración válida. Las solicitudes sin autorización reciben 401 sin estado. Se comparan digests de longitud fija en tiempo constante y se envían respuestas mínimas en español con Cache-Control: no-store.

Compose mantiene publicación en localhost y su comprobación del mock envía la credencial desde el entorno. Fuera del desarrollo local se usará acceso interno y transporte cifrado. El proxy público futuro no publicará las rutas operativas. Las credenciales de usuarios y de integración M2M no habilitan monitoreo.

## Consecuencias

La invocación directa de Compose o de un servidor exige definir HEALTHCHECK_TOKEN. Los lanzadores conservan un arranque automatizado sin archivo manual de secretos. Cambiar la credencial requiere actualizar los monitores y recrear/reiniciar los componentes afectados. En el desarrollo local se comparte una credencial operativa entre componentes; un despliegue posterior podrá separarlas por servicio y usar un gestor de secretos.

## Validación

Pruebas HTTP del mock, pruebas de los cinco handlers Go, arranque local de los cinco binarios, rechazo de credenciales ausentes/cortas y validación de Compose. Readiness de Go sigue devolviendo 503 con credencial válida porque los casos de negocio aún no existen. Resultados y límites en [auditoría](../SECURITY-AUDIT.md).

## Actualización 2026-11-08 - contexto y por qué

### Contexto

Los monitores necesitan estado técnico aunque el login no exista, pero las comprobaciones no deben filtrar estado a usuarios anónimos.

### Decisión, motivo y consecuencias

Conservar Bearer técnico independiente y liveness/readiness separados. Frente al /health abierto de las demos, es adecuado para este proyecto porque no reutiliza claves M2M ni depende de sesión humana. El costo es distribuir el secreto a monitores; las rutas no acreditan por sí solas alta disponibilidad.

### Referencias de clase y alcance

Clase 8.pdf pp. 8-9 y Clase 9.pdf pp. 11-12; implementación actual y requisito previo de salud autenticada.

El texto anterior se conserva como historia. Toda revisión posterior se añadirá en este mismo archivo.


## Actualización 2026-11-09

Contexto: se integra la aplicación completa y observabilidad sin convertir una simulación en un guardado implícito. Se conservan los motivos originales: invitado sin escrituras y salud autenticada. Identidad solo habilita persistencia explícita y datos propios; Prometheus usa la misma credencial técnica separada para /metrics. Estado y evidencia vigentes en [verificación](../VERIFICACION-FINAL.md). Este cambio se añade al mismo ADR, sin archivo sustituto.
