# Capacidad M2M desplegada en Render Free

Estado 2026-11-09: **desplegada y verificada en cloud**, [https://renewable-consumo-v1.onrender.com](https://renewable-consumo-v1.onrender.com). El proceso Go corre en Render, independiente de la PC; se distingue de los túneles que siguen sirviendo el entorno local. [Despliegue](evidence/render-cloud-2026-11-09/despliegue.json), [sonda HTTPS](evidence/render-cloud-2026-11-09/https.json), [captura del panel](evidence/render-cloud-2026-11-09/render-activo.jpg).

## Alcance y configuración

POST https://renewable-consumo-v1.onrender.com/api/v1/estimaciones-consumo. OpenAPI: [https://renewable-consumo-v1.onrender.com/openapi.json](https://renewable-consumo-v1.onrender.com/openapi.json), contrato 1.0.1 intacto. Render Web Service Docker, plan Free, Oregon, 0.1 CPU/512 MiB, despliegue manual y auto-deploy Off. No se agregaron discos, bases ni planes pagos. El commit desplegado es 69850d800b4bbb0295d067b51387cd14324f25f3, en la [rama de despliegue](https://github.com/Franciscosrrr/ArquitecturasoftwareRenewableEnergyTiziFran/tree/deploy/estimaciones-consumo-20261109); main queda pendiente de revisión/integración.

infra/cloud.Dockerfile ejecuta Gateway y Consumo supervisados por tini en una instancia. Consumo escucha en loopback y Gateway en PORT; CAPABILITY_ONLY=true excluye cuentas, datos privados, Solar y contratos distintos del M2M. Es una unidad acotada sin persistencia, no el alojamiento de toda la arquitectura. El sistema completo sigue demostrando cuatro microservicios, bases privadas, broker, búsqueda y observabilidad en Compose. Se conserva Go y su cálculo decimal, sin portar el dominio.

Cuatro credenciales nuevas se generaron para Render: HEALTHCHECK_TOKEN, INTERNAL_TOKEN, M2M_API_KEY y M2M_DENIED_KEY. La revisión automática rechazó reutilizar las credenciales de la PC sin autorización específica para ese destino; se siguió la alternativa segura de generar otras. .runtime/render.env guarda el entorno privado localmente, excluido de Git y Docker. Las dos claves M2M de integración se entregan al usuario por separado; no son las del túnel Cloudflare. No se guarda la contraseña humana de Render en el proyecto.

## Verificación y corrección

Ocho estimaciones devolvieron 47.4 kWh y 1.58 kWh/día. TLS y hash de OpenAPI coinciden; 400/401/403/413/415/422, readiness autorizada y bloqueo de rutas privadas/técnicas comprobados. Una ráfaga de 17 llamadas con la clave sin permiso produjo seis 429 con Retry-After: 1. La cuota sigue siendo token bucket de 1 solicitud/s y ráfaga 10, no una ventana móvil estricta ni una cuota global de varias instancias.

El p95 caliente observado fue 730.585 ms con ocho muestras desde esta conexión, por debajo de 1.2 s/5 s. El percentil por rango más próximo coincide con el máximo de esas ocho llamadas; no es un SLA. No se midió cold start ni se ejecutó la integración desde Clínica.

La primera sonda detectó /privado-openapi.json accesible porque la imagen contenía todos los contratos. Se corrigió el registro de rutas en modo M2M y se añadió TestCapabilityOnlyContractSurface, con archivos realmente presentes, comprobando también el modo completo. [Primer intento](evidence/render-cloud-2026-11-09/primer-intento.txt), [Go aprobado](evidence/render-cloud-2026-11-09/go-gateway.txt) y sonda final satisfactoria. El OpenAPI M2M no cambió.

## Operación y activación en frío

El plan Free puede suspender la instancia tras 15 minutos sin tráfico; Render avisa que la reactivación puede demorar 50 segundos o más. Eso excede el presupuesto de Clínica en frío. Antes de presentar, preparar la instancia con GET /openapi.json y esperar el contrato real; después ejecutar el flujo. No se inventa un resultado ni se promete una latencia fría de cinco segundos. [Límites oficiales](https://render.com/docs/free).

Render aloja solo la capacidad M2M y sigue funcionando con la PC apagada. La web completa conserva su [túnel local](PUBLICACION-WEB.md), que necesita PC/Docker/red. El [túnel M2M anterior](PUBLICACION-CLINICA.md) sigue como alternativa, con sus propias claves y dependencia de la PC. No se apagó ninguno ni se cambiaron límites de WSL.

Para redeploy: revisar la rama/commit, conservar plan Free y elegir Manual Deploy en el panel. No configurar un health check anónimo sobre /health: requiere una credencial técnica; se dejó comprobación por puerto. Mantener auto-deploy Off y no cambiar contrato, errores o redondeo sin transición acordada.

Sonda reproducible: tests/cloud_publication.py recibe la URL HTTPS por argumento y lee M2M_API_KEY, M2M_DENIED_KEY y opcionalmente HEALTHCHECK_TOKEN del entorno; CLOUD_EVIDENCE elige el archivo de resultado. Usa una instancia ya activa, certificados verificados y no sigue redirecciones de credenciales. scripts/verificar-publicacion.ps1 sigue disponible para la comprobación básica.

## Coordinación y entrega

Clínica se encarga de probar desde su backend, por instrucción del usuario. Compartir el nuevo endpoint y las nuevas claves por privado, con el aviso de cold start. El costo energético lo calcula Clínica; la API no agrega precio. El servicio queda configurado sin fecha de baja, sujeto a las cuotas/condiciones Free y al acuerdo de continuidad de la evaluación.

El recordatorio solicitado corresponde ahora: revisar/fusionar la rama de despliegue en main y comprobar CI de la integración. No se fusionó automáticamente. El [estado anterior de preparación](evidence/preparacion-render-2026-11-09/estado.json) se conserva como historia; su bloqueo de login fue resuelto cuando el usuario creó la cuenta y autorizó el acceso.
