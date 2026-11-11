# Renewable Energy — Arquitectura de Software 2026

> Estado 11/11/2026: [API M2M en Render](https://renewable-consumo-v1.onrender.com/openapi.json), contrato 1.0.1 verificado, y [web completa por Cloudflare](https://boring-eclipse-copying-did.trycloudflare.com). La API Render no depende de la PC; la web sí. [Operación, pruebas y cold start](docs/DESPLIEGUE-GRATUITO.md).

Calculadora de consumo y estudios solares con acceso invitado, guardados propios y administración. Tiziano Nicolas (2407889) y Francisco Skrie (2423159), UCC. [Repositorio](https://github.com/Franciscosrrr/ArquitecturasoftwareRenewableEnergyTiziFran). Comisión **A1**. JTP: **Nicolás Rodríguez**.

El entorno completo implementa Usuarios, Consumo, Electrodomésticos y Solar, Gateway Go, MySQL privados, MongoDB, RabbitMQ, Meilisearch, Valkey, dos réplicas Solar y observabilidad. La capacidad de Clínica mantiene OpenAPI **1.0.1** sin precio. El vendedor solar real es **Comparador de Proveedores 1.1.0**, integrado por SKU y cantidad; ante una falla se utilizan paneles simulados identificados. [Contrato, configuración y límites](docs/contracts/comparador-proveedores-v1.1.0/README.md). La web está publicada por HTTPS mediante su túnel desde la PC. La capacidad M2M está desplegada en Render Free, independiente de la PC y con claves propias. La publicación M2M local anterior quedó detenida y el entorno final-dev obsoleto se retiró, conservando datos. [Estado de despliegue](docs/DESPLIEGUE-GRATUITO.md).

## Arranque único

Docker Desktop con motor Linux y Compose v2. No se requiere Go, Node, Python ni bases instalados en el host. Desde la raíz:

```powershell
.\scripts\compose.ps1 up -d --build
```

```sh
sh scripts/compose.sh up -d --build
```

Los scripts generan secretos aleatorios una sola vez en `.env`, incorporan claves nuevas sin rotar las anteriores y preparan la credencial de métricas excluida de Git. La primera descarga/compilación puede tardar. Construyen cada servicio por separado para contener el consumo; las siguientes aprovechan caché. Conservar `.env` junto con los volúmenes y no publicarlo.

| Acceso local | Dirección |
|---|---|
| Aplicación completa | http://localhost:8080 |
| Grafana | http://localhost:3000 |
| Prometheus | http://localhost:9090 |
| Jaeger | http://localhost:16686 |

Puertos solo en loopback. Usuario inicial de la aplicación: `admin@renewable.local`, contraseña ADMIN_PASSWORD de `.env`; Grafana usa `admin` y GRAFANA_ADMIN_PASSWORD. Recuperarlas localmente sin enviarlas a Git ni capturas. El resto de las cuentas se registra desde Mis consumos. Las métricas y /health requieren HEALTHCHECK_TOKEN; las bases, broker, índice y caché no publican puertos al host.

Para detener sin perder datos: mismo script con `stop`. Volver a iniciar con `up -d`. No utilizar `down -v`: elimina los datos. [Recursos y límites locales](docs/RECURSOS-LOCALES.md).

## Recorrido de negocio

Como invitado, elegir fichas activas y hábitos; calcular consumo, comparar alternativas con superficie/HSP manuales y cotizar. Nada de ese recorrido escribe consumos, estudios ni eventos de negocio. Recargar pierde el borrador.

Para conservarlo, crear cuenta/iniciar sesión, crear un lugar y pulsar Guardar consumo. La validación y el cálculo se repiten en el servidor. Calcular paneles confirma la evaluación: congela versiones de configuración y lugar, guarda idempotencia/outbox en la misma transacción y encola un estudio. El historial muestra el procesamiento y resultado recuperado de MongoDB; editar después conserva el snapshot anterior. Comparar con otras condiciones solares permite calcular otra alternativa sobre la evaluación histórica. Archivar/restaurar organiza guardados sin crear versiones de contenido; eliminar quita guardados conservando sus resultados históricos. Solo el propietario puede ver sus datos, incluso si otro usuario es administrador.

La administración permite editar/desactivar equipos/paneles, gestionar roles, reconstruir el índice e inspeccionar/reprocesar DLQ. Actualizaciones usan versión esperada. Una oferta sin stock puede cambiar el panel seleccionado; la cotización no compra ni reserva. Las HSP son un supuesto manual identificado, no una fuente meteorológica real ni garantía de autonomía.

Ejemplo: 30 días, cuatro lámparas de 10 W×5 h/día; una heladera de 150 W×24 h/día×factor 0.3; lavarropas de 0.75 kWh×12 ciclos. Total **47.4 kWh**, promedio **1.58 kWh/día**. Son referencias didácticas que deben sustituirse por fichas reales al usar el sistema fuera de la evaluación.

## Verificación

```powershell
.\scripts\verify.ps1
```

```sh
sh scripts/verify.sh
```

Ejecuta Go, integración real y Chromium escritorio/móvil en secuencia. [Resultados y condiciones](docs/VERIFICACION-FINAL.md), [evidencia actual](docs/evidence/proveedor-real-2026-11-11/README.md), [postmortem](docs/POSTMORTEM.md) y [carga](docs/CARGA-FINAL.md). La sonda histórica de fallas del 09/11 presupone modo simulado y el destino interno del simulador; no debe interpretarse como ensayo del proveedor real. Para ejecutarla en ese escenario desde el host se requiere Python: `python tests/fault_final.py`; detiene temporalmente dependencias de este proyecto y las recupera incluso si falla una aserción. No es parte del arranque normal.

## Documentación vigente

- [Requisitos y aceptación](SPEC.md), [matriz final](docs/VERIFICACION-FINAL.md).
- [Arquitectura y diagramas](docs/ARCHITECTURE.md), [D1–D13 canónicos](docs/adr/README.md), [los nueve prácticos y sus diferencias justificadas](docs/CLASES-PRACTICAS.md).
- [Contrato M2M 1.0.1](docs/contracts/README.md), [cuenta/guardados/administración](docs/contracts/private-v1/README.md), [eventos](docs/contracts/events/v1/README.md).
- [Despliegue M2M en Render y distinción respecto del túnel](docs/DESPLIEGUE-GRATUITO.md), [observabilidad](docs/OBSERVABILITY.md), [recursos](docs/RECURSOS-LOCALES.md).

Las entregas 1/2 y sus evidencias se conservan como historia. Clínica utiliza la URL Render entregada y sus claves privadas; su integración desde el backend consumidor debe comprobarla ese grupo. La publicación M2M local antigua está detenida. El contrato 1.0.1 permanece congelado y Render Free puede exceder 1.2 s/5 s al reactivarse.

Documentación completa: [índice y vigencia](docs/README.md), [configuración](docs/CONFIGURACION.md), [guía de comentarios](docs/DOCUMENTACION-CODIGO.md) y [cierre de entrega](docs/ENTREGA.md). La nueva integración solar se configura en el backend; un clon sin su clave usa el simulador y lo identifica. Nunca publicar `.env` ni `.runtime`.

## Mejora de experiencia de cliente — 2026-11-09

[Archivo sin versiones artificiales, eliminación de guardados y búsqueda solar](docs/EXPERIENCIA-CLIENTE.md). Pruebas posteriores: 19 HTTP y 12 de navegador aprobadas. Se conservan los resultados anteriores como antecedentes. Clínica mantiene OpenAPI 1.0.1.


## Página local por Cloudflare

[Web pública verificada](https://boring-eclipse-copying-did.trycloudflare.com), independiente de la URL M2M de Clínica. El usuario autorizó la publicación completa. La entrada HTTPS mantiene las mismas cuentas y datos del entorno local. El proveedor real y el respaldo fueron verificados en 14 y 4 casos de navegador respectivamente, además de la sonda HTTPS actual. [Operación y límites](docs/PUBLICACION-WEB.md): scripts/publicar-web.ps1 iniciar|estado|detener. La PC/Docker/red deben permanecer activos y el hostname temporal puede cambiar al reiniciar el túnel. La URL enlazada es la observada el 11/11; consultar el script para obtener la actual.
