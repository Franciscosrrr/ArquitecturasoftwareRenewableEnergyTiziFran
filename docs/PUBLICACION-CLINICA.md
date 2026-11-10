# Publicación M2M para Clínica

## Publicación cloud principal — 2026-11-09

Render Free está desplegado y verificado: POST https://renewable-consumo-v1.onrender.com/api/v1/estimaciones-consumo; [OpenAPI 1.0.1](https://renewable-consumo-v1.onrender.com/openapi.json), sin modificaciones. [Operación y pruebas](DESPLIEGUE-GRATUITO.md). Usa claves nuevas específicas, entregadas al usuario en CLINICA-CREDENCIALES-PRIVADAS.txt para compartir por privado. La instancia Free puede superar el presupuesto al reactivarse; preparar antes de la presentación. La prueba desde el backend de Clínica queda a cargo de ese grupo.

## Publicación local anterior: detenida

Estado actual 2026-11-09: **detenida por solicitud del usuario después de entregar el ZIP de Render a Clínica**. Sus dos contenedores y claves locales se conservan para recuperación; el enlace anterior no se presenta como operativo. Los datos y pruebas siguientes registran la publicación histórica mediante Quick Tunnel. No se creó cuenta ni dominio Cloudflare ni se contrató un plan.

- URL base: https://emails-bingo-preference-packets.trycloudflare.com
- Operación: POST https://emails-bingo-preference-packets.trycloudflare.com/api/v1/estimaciones-consumo
- Contrato: [https://emails-bingo-preference-packets.trycloudflare.com/openapi.json](https://emails-bingo-preference-packets.trycloudflare.com/openapi.json), versión 1.0.1 sin cambios.
- Datos públicos de la comprobación: [https.json](evidence/publicacion-cloudflare-2026-11-09/https.json).

## Alcance y recursos

compose.publicacion.yaml utiliza un proyecto separado renewable-clinica-publica: api-publica (Gateway y Consumo Go supervisados, sin SQL) y tunnel. La API no registra usuarios, lugares, administración ni capacidades privadas; solo tiene montado el contrato M2M. Se comprobaron 401 en rutas privadas, salud y métricas y 404 al contrato privado. Puerto 18081 solo en loopback. No se publican bases, broker, dashboard ni secretos del sistema completo.

La imagen Go ya probada se reutiliza sin reescribir el cálculo ni cambiar el OpenAPI. Cloudflare actúa como proxy TLS: el código corre en la PC, no en Workers ni en un servidor de Render. El contenedor API tiene límite 128 MiB/1 CPU y cloudflared 64 MiB/0.5 CPU. La primera muestra sumó unos 25 MiB entre ambos; el límite no es consumo reservado ni un SLA. La imagen de cloudflared queda fijada por digest. Cada servicio reinicia con unless-stopped cuando Docker está activo.

## Operación en PowerShell

Desde el repositorio:

```powershell
.\scripts\publicar-clinica.ps1 estado
.\scripts\publicar-clinica.ps1 iniciar
.\scripts\publicar-clinica.ps1 detener
```

El script iniciar genera cuatro credenciales dedicadas solo si no existe .runtime/clinica.env; no imprime las claves ni reutiliza las del sistema completo. Recupera la última URL del log en .runtime/clinica-url.json. Si la imagen no existe, la construye secuencialmente con el Dockerfile preparado. No borra volúmenes ni modifica configuración global de WSL, Windows, suspensión o router.

El sistema completo y esta publicación son independientes. Se puede detener el entorno de demostración con scripts/compose.ps1 stop para liberar sus recursos y dejar solo la API pública; para recuperar la aplicación completa usar scripts/compose.ps1 up -d --no-build. No se detuvo automáticamente el entorno que el usuario ya tenía activo.

## Condiciones para Clínica

Mientras la PC esté encendida y sin suspender, Docker esté activo, el túnel esté conectado y la red funcione, este origen puede responder. El servicio Go no tiene la suspensión por inactividad del plan Render Free. No se garantiza latencia desde otra red ni disponibilidad permanente de Quick Tunnels. Al reiniciar/recrear el proceso cloudflared puede asignarse otra URL: ejecutar estado, comprobarla y comunicar la nueva a Clínica. No recrear el túnel durante la evaluación sin coordinarlo.

**La URL no es permanente ni tiene garantía de uptime.** Para un hostname fijo hace falta una cuenta Cloudflare y un dominio configurado; no existen todavía. La fecha final de evaluación/disponibilidad continúa pendiente de acuerdo. Este resultado no acredita por sí solo el compromiso de disponibilidad hasta finalizarla.

Se verificaron certificado HTTPS, hash del contrato, 9 estimaciones correctas, 401/403 y rutas privadas inaccesibles. Ocho llamadas medidas tuvieron p95 por rango más próximo de 154.631 ms (con 8 muestras corresponde al máximo), todas por debajo de 1.2 s y 5 s desde esta conexión al edge público. Cardiología dio 0.381 kWh. Falta prueba desde el backend real de Clínica y observación durante el período acordado. No se ejecutó una prueba de saturación contra Cloudflare.

## Entrega de credenciales

El usuario recibió un archivo separado CLINICA-CLOUDFLARE-CREDENCIALES-PRIVADAS.txt en outputs con únicamente las dos claves que necesita Clínica. Compartirlo por privado. No versionar ni agregarlo al ZIP compartible. .runtime está ignorado por Git. Render usa claves nuevas específicas de su publicación; las antiguas solo corresponden al túnel local detenido.

El contrato 1.0.1, permisos, redondeo, ausencia de precio y cuota sostenida 60/min con ráfaga 10 se conservan. Render Free está desplegado y es la publicación principal de la capacidad. El proveedor de paneles real sigue pendiente, por separado.

Referencias: [Quick Tunnels](https://developers.cloudflare.com/tunnel/get-started/quick-tunnels/), [Tunnel con hostname](https://developers.cloudflare.com/tunnel/get-started/), [Render Free](https://render.com/docs/free).
