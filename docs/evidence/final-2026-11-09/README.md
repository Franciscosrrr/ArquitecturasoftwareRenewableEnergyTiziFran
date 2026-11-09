# Evidencias de implementación final local

Esta carpeta reúne resultados ejecutados contra Go, MySQL, MongoDB, RabbitMQ, Meilisearch y Valkey reales. Las verificaciones anteriores se conservan en sus carpetas históricas. No contiene `.env`, tokens de sesión ni claves M2M. Los resultados de la primera revisión de recursos se explican en RECURSOS-LOCALES; el consumo final debe leerse junto con la carga y sus condiciones.

Los archivos se incorporan después de ejecutar cada comprobación; un comando documentado por sí solo no acredita que haya pasado. La URL HTTPS pública permanece pendiente de cuenta/despliegue del equipo. El vendedor simulado no demuestra un acuerdo con el grupo real.

Cierre: 17 HTTP aprobados después de corregir reconexión e independencia de fixtures; Go normal/race aprobados; 16 observaciones de recuperación, incluida persistencia de DLQ al recrear el broker y alta visible en menos de 5 s. Los intentos fallidos anteriores se conservan identificados. solar-reglas-final.txt registra la frontera de selección y 89.4% de cobertura de ese dominio.
