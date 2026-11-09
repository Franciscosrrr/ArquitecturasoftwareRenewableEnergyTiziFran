# Capacidad M2M desplegada en Render Free

API: https://renewable-consumo-v1.onrender.com/api/v1/estimaciones-consumo. OpenAPI: https://renewable-consumo-v1.onrender.com/openapi.json. Servicio Free/Oregon, 512 MiB/0.1 CPU, sin extras pagos y con despliegue automático desactivado. El código desplegado es 69850d800b4bbb0295d067b51387cd14324f25f3; su CI aprobó: https://github.com/Franciscosrrr/ArquitecturasoftwareRenewableEnergyTiziFran/actions/runs/42717335793.

Esta rama contiene el proyecto completo verificado para revisión e integración posterior en main. Render construye únicamente infra/cloud.Dockerfile: Gateway y Consumo Go supervisados, CAPABILITY_ONLY=true, sin SQL, broker, frontend ni datos privados. Esta capacidad sigue operativa con la PC apagada; la web completa continúa dependiendo de la PC y su túnel Cloudflare.

Se conserva exactamente el contrato 1.0.1. Se verificaron HTTPS, hash, cálculos, permisos, errores, protección de rutas privadas y 429. La evidencia está en docs/evidence/render-cloud-2026-11-09/ y la operación en docs/DESPLIEGUE-GRATUITO.md. Las cuatro variables secretas se generaron para Render; no están en Git. Las claves del túnel anterior no se reutilizan.

Free puede suspenderse por inactividad y exceder 1.2 s/5 s al reactivarse. El frío no fue medido; el p95 caliente fue 730.585 ms en ocho muestras. Clínica realizará su prueba desde su backend. Revisar e integrar esta rama en main y verificar la CI de esa integración; no se fusionó main.
