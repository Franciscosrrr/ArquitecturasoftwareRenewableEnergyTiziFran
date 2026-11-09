# API Gateway Go

Estado 2026-11-09. Sirve React, contratos y rutas explícitas hacia Usuarios, Consumo, catálogo y Solar. Comprueba el token contra identidad y usa el rol vigente para las rutas privadas; elimina cabeceras de identidad aportadas por el cliente y envía actor/token técnico al servicio responsable. Los datos propios siguen protegidos por sus servicios. No expone el resolver interno del catálogo.

Balancea dos Solar por round-robin y salud activa. Aplica cuotas públicas/M2M y privadas por cuenta, timeouts y telemetría. M2M se limita por hash de API key y Consumo valida el permiso; no se confía en X-Forwarded-For enviado por Internet. El presupuesto M2M es 4.5 s y el de otras operaciones normalmente 8 s, con reindexación administrativa acotada aparte.

CAPABILITY_ONLY=true registra solo la capacidad M2M y su contrato/salud protegida para la unidad cloud, sin login, datos privados ni frontend. El gateway completo permanece en Compose. La entrada web Nginx/Cloudflare es otro proyecto y no sustituye este gateway ni su balanceo.

[Arquitectura](../docs/ARCHITECTURE.md), [contratos](../docs/contracts/README.md), [fallas verificadas](../docs/POSTMORTEM.md), [web HTTPS](../docs/PUBLICACION-WEB.md).
