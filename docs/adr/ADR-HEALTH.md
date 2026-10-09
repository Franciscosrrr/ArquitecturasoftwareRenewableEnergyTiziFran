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
