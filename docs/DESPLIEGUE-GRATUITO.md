# Publicación HTTPS y alternativa Render Free

Estado actual 2026-11-09: la API M2M y la web tienen HTTPS mediante Cloudflare desde la PC autorizada. El enunciado pide además la capacidad propia ejecutándose en la nube: el proxy TLS no cumple por sí solo ese alojamiento. Se seleccionó Render Free para conservar la implementación Go/Docker existente. La configuración y el paquete M2M están preparados, pero el panel requiere iniciar sesión y el usuario no puede crear/acceder a su cuenta ahora. **No hay despliegue ni URL Render verificados.** [Estado comprobado](evidence/preparacion-render-2026-11-09/estado.json).

Las publicaciones actuales de la PC siguen operativas e independientes: [M2M](PUBLICACION-CLINICA.md) y [web](PUBLICACION-WEB.md). No se reiniciaron ni se rotaron sus claves por esta preparación. La prueba desde el backend consumidor queda a cargo de Clínica.

## Qué se publica

`render.yaml` define un único Web Service **Free** con `infra/cloud.Dockerfile`. Expone exclusivamente la capacidad M2M `/api/v1/estimaciones-consumo` y el OpenAPI congelado 1.0.1. Ejecuta Gateway y Consumo en dos procesos supervisados por tini dentro de una instancia gratuita. Consumo escucha solo en loopback. No utiliza bases efímeras como persistencia de usuarios ni simula el despliegue de la arquitectura completa.

Esta unidad de despliegue es una excepción documentada a la separación por contenedor del entorno completo: permite compartir los 750 instance-hours gratuitos mensuales sin bases ni cuotas adicionales para una capacidad que es una función pura. La demostración de microservicios, bases privadas, broker y observabilidad se realiza mediante Compose. Un Render Free no alcanza para alojar de forma durable todo ese entorno. La recomendación general de Render es separar las aplicaciones; esta excepción acotada conserva procesos y contratos separados y supervisa la caída de cualquiera de los dos.

## Publicación cuando esté disponible la cuenta

1. Subir estos cambios al repositorio público después de revisarlos. En Render, crear una cuenta y un **Blueprint** conectado al repositorio; elegir `render.yaml`.
2. Verificar que figura **Free**, sin discos ni bases pagas y sin upgrades automáticos. Desplegar. El nombre puede ajustarse si ya está ocupado.
3. Copiar del dashboard la URL HTTPS realmente asignada. No deducirla del nombre propuesto. La configuración no fija un health check HTTP porque `/health/*` está protegido: Render usa la comprobación de puerto por defecto.
4. En Environment, recuperar `M2M_API_KEY`, `M2M_DENIED_KEY` y `HEALTHCHECK_TOKEN` sin pegarlos en Git, capturas o logs. Enviar las dos claves M2M al grupo Clínica por un canal privado. La primera representa el permiso `estimar-consumo`; la segunda es válida pero sin ese permiso y produce 403.
5. Desde otra red, ejecutar `scripts/verificar-publicacion.ps1 -BaseUrl https://URL-REAL` con las claves en variables de entorno. El script comprueba HTTPS, OpenAPI, 200, 401, 403 y el presupuesto de una solicitud caliente. Registrar URL, fecha y resultado en la evidencia del equipo.
6. Comunicar a Clínica URL, disponibilidad y límite sostenido 60 solicitudes/minuto, ráfaga 10. No cambiar `/v1`, errores ni redondeo. El costo en pesos lo calcula Clínica.
7. Mantener el servicio y revisar las cuotas durante la evaluación. Desactivar despliegues automáticos evita cambiar el contrato inadvertidamente, pero las correcciones requieren redeploy explícito.

## Activación en frío y presupuesto del consumidor

Render Free se suspende después de 15 minutos sin tráfico y puede tardar aproximadamente un minuto en activarse. Eso supera tanto el presupuesto interactivo observado de Clínica (1.2 s) como sus 5 s en segundo plano. Coordinar una llamada de preparación antes de presentar y después medir la llamada caliente. No se garantiza un SLA que el plan gratuito no ofrece. Si la cátedra exige 5 s también en frío, esta opción no satisface ese criterio y hará falta una infraestructura ya disponible que no se suspenda.

Fuentes oficiales consultadas: [límites Free](https://render.com/docs/free), [Blueprint](https://render.com/docs/blueprint-spec), [Docker](https://render.com/docs/docker), [recomendación de separación](https://render.com/docs/faq). No se contratan servicios ni se crean cuentas desde el repositorio.


## Preparación realizada y recordatorio

Se preparó una copia aislada en una rama local deploy/estimaciones-consumo-20261109 con el código/configuración necesario para construir M2M y contrato congelado. No se creó una rama remota ni se hizo commit/push de la entrega completa. infra/cloud.Dockerfile todavía no está en main remoto; antes de crear el Blueprint hay que publicar el código fuente o una imagen accesible. Puede usarse una rama de despliegue y dejar la integración final en main para la revisión posterior.

El paquete Despliegue-M2M-Render.zip entregado al usuario contiene 62 archivos, sin secretos ni dependencias instaladas, y una guía específica. El Blueprint genera sus secretos; esos valores no son automáticamente los de la publicación de la PC. Compartir nuevas credenciales solo por privado una vez verificado el despliegue. No enviar claves mediante este documento.

Al completar el despliegue: recordar revisión de los cambios completos, integración en main y ejecución de CI. Es un recordatorio ligado al cierre de este trabajo, sin alarma de fecha arbitraria. No se considera completado por preparar archivos o abrir la pantalla de login.
