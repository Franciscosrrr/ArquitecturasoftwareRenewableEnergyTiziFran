# Infraestructura local y unidad cloud preparada

Compose inicia cuatro servicios de negocio, Gateway/React, dos instancias Solar, tres MySQL privados, MongoDB, RabbitMQ, Valkey, Meilisearch, Prometheus, Grafana, Loki y Jaeger. El simulador comercial no cuenta como servicio propio. Proyecto por defecto renewable-energy-entrega2 para conservar los volúmenes anteriores; cada base nueva recibe volumen/usuario propios. Solo8080/3000/9090/16686 se publican en 127.0.0.1.

Usar scripts/compose.ps1 o compose.sh: generan/incorporan secretos sin rotación y construyen secuencialmente. Límites por contenedor, cachés pequeñas y Go con 2 hilos permiten operar en el equipo de 16 GB. No son una configuración de capacidad productiva. Perfil verificar: pruebas Go con MySQL real, HTTP y navegador. Perfil mock: servidor histórico opcional, separado en 8081; no sustituye ninguna dependencia del entorno final.

Volúmenes catalogo-data, usuarios-data, consumo-data, solar-data, broker-data, search-data y observabilidad conservan datos. Valkey es descartable; los índices se reconstruyen desde MySQL para equipos y Mongo para paneles. No borrar `.env` conservando volúmenes inicializados. Detener con stop; no usar down -v. [Recursos](../docs/RECURSOS-LOCALES.md).

`cloud.Dockerfile`, `cloud-entrypoint.sh` y render.yaml publican únicamente la capacidad M2M sin estado. No despliegan toda esta infraestructura en Render Free ni usan archivos efímeros como bases durables. [Procedimiento y límites](../docs/DESPLIEGUE-GRATUITO.md).


Dos proyectos opcionales exponen HTTPS mediante Cloudflare desde la PC: compose.publicacion.yaml para M2M aislado y compose.web-publica.yaml para Nginx/túnel web. No alojan el código en servidores cloud; el origen sigue local. La web reutiliza el gateway y las bases, con cookie Secure en la entrada, límites de 32/64 MiB y sin puertos adicionales. [M2M](../docs/PUBLICACION-CLINICA.md) y [web](../docs/PUBLICACION-WEB.md). Render es el alojamiento cloud preparado aparte; no se declara publicado antes de completar el acceso y verificar su URL.
