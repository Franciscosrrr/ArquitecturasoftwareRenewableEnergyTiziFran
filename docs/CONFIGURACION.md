# Configuración y operación local

Revisión 11/11/2026. Docker Desktop debe tener el motor Linux disponible. El arranque único crea secretos locales, mantiene sus valores al reiniciar, construye en secuencia e inicia las dependencias:

```powershell
.\scripts\compose.ps1 up -d --build
```

```sh
sh scripts/compose.sh up -d --build
```

Los scripts agregan claves nuevas cuando faltan y conservan la identidad del broker. No requieren instalar bases, Go, Node o Python en Windows. `.env` y `.runtime` quedan excluidos de Git y del contexto de build; conservarlos con sus volúmenes. Si se pierde una credencial de una base ya inicializada, no se arregla regenerándola al azar. No utilizar `down -v`: elimina los datos.

## Accesos y cuentas

Aplicación: http://localhost:8080. Una persona puede calcular sin cuenta y registrarse desde Mis consumos para guardar sus datos. Administrador inicial: `admin@renewable.local`, salvo `ADMIN_EMAIL`; contraseña privada `ADMIN_PASSWORD` del entorno. Grafana usa `admin` y `GRAFANA_ADMIN_PASSWORD` en http://localhost:3000. Prometheus: http://localhost:9090. Jaeger: http://localhost:16686. Los puertos son loopback; las bases, RabbitMQ, Valkey, Meilisearch y Loki no publican puertos al host.

Las claves M2M de Clínica, los secretos de health/internal, las sesiones de usuario y la clave del proveedor cumplen funciones diferentes. No intercambiarlas ni incorporarlas al frontend. En Render se utilizan las credenciales emitidas para ese entorno, no las del simulador o la PC.

## Proveedor real y simulador

Un clon sin la configuración externa funciona con el simulador identificado. Para habilitar Comparador de Proveedores, configurar fuera de Git:

```dotenv
PANEL_PROVIDER_MODE=externo
PANEL_PROVIDER_URL=http://20.226.84.78
PANEL_PROVIDER_ALLOW_HTTP=true
PANEL_COMPARATOR_API_KEY=<clave privada recibida del grupo>
```

La excepción HTTP solo permite esa IP exacta. La clave viaja sin TLS en ese tramo; cuando el proveedor entregue HTTPS, cambiar URL y retirar la excepción. `PANEL_PROVIDER_API_KEY` pertenece al simulador local y debe conservarse. `PANEL_SIMULATOR_URL` apunta al servicio de respaldo interno. Para volver al modo simulado, seleccionar `PANEL_PROVIDER_MODE=simulado` y el destino interno documentado en Compose.

Después de cambiar el entorno, recrear las dos réplicas Solar con `docker compose up -d solar solar-2`. Un fallo técnico activa el respaldo completo identificado; una lista válida sin ofertas indica falta de stock. [Contrato y tratamiento de errores](contracts/comparador-proveedores-v1.1.0/README.md).

## Comandos operativos

| Acción | Comando desde la raíz |
|---|---|
| Detener conservando los datos | `scripts/compose.ps1 stop` |
| Reanudar | `scripts/compose.ps1 up -d` |
| Ver estados | `docker compose ps` |
| Ver logs de Solar | `docker compose logs --tail 100 solar solar-2` |
| Consultar límites y consumo actual | `docker stats --no-stream` |
| Verificar Go, HTTP e interfaz en secuencia | `scripts/verify.ps1` o `sh scripts/verify.sh` |
| Publicar la web autorizada | `scripts/publicar-web.ps1 iniciar` |
| Obtener la URL actual de la web | `scripts/publicar-web.ps1 estado` |
| Detener solo esa publicación | `scripts/publicar-web.ps1 detener` |

Las sondas de fallas detienen temporalmente dependencias del proyecto: ejecutarlas en una ventana de prueba, siguiendo su documentación. Las suites estándar no necesitan causar una caída en la VM ajena. Los límites de RAM/CPU y la compilación secuencial permanecen configurados; no se modifica WSL globalmente.

## Inventario de variables declaradas

Esta tabla relaciona nombres de configuración encontrados en el código y Compose. «Privada» significa que su valor no se publica. No todos los nombres pertenecen a `.env`: algunos se fijan internamente en Compose o son exclusivos de pruebas. Los secretos obligatorios del arranque se generan mediante los scripts; la credencial comercial debe recibirse por privado. Las declaraciones enlazadas indican el uso y los valores alternativos aplicables.

| Variable | Privada | Declaración o uso |
|---|---|---|
| `ADDR` | No | [platform/runtime.go](../platform/runtime.go) |
| `ADMIN_EMAIL` | No | [compose.yaml](../compose.yaml), [services/usuarios/cmd/server/main.go](../services/usuarios/cmd/server/main.go) |
| `ADMIN_PASSWORD` | Sí | [compose.yaml](../compose.yaml), [services/usuarios/cmd/server/main.go](../services/usuarios/cmd/server/main.go) |
| `AMQP_URL` | No | [platform/bus.go](../platform/bus.go) |
| `CAPABILITY_ONLY` | No | [gateway/cmd/server/server.go](../gateway/cmd/server/server.go), [services/consumo/cmd/server/main.go](../services/consumo/cmd/server/main.go) |
| `CATALOG_URL` | No | [gateway/cmd/server/server.go](../gateway/cmd/server/server.go), [services/consumo/cmd/server/main.go](../services/consumo/cmd/server/main.go) |
| `CONSUMO_DB_PASSWORD` | Sí | [compose.yaml](../compose.yaml) |
| `CONSUMO_DB_ROOT_PASSWORD` | Sí | [compose.yaml](../compose.yaml) |
| `CONSUMO_URL` | No | [gateway/cmd/server/server.go](../gateway/cmd/server/server.go), [services/solar/cmd/server/main.go](../services/solar/cmd/server/main.go) |
| `COOKIE_SECURE` | No | [services/usuarios/cmd/server/main.go](../services/usuarios/cmd/server/main.go) |
| `DOCS_DIR` | No | [gateway/cmd/server/server.go](../gateway/cmd/server/server.go) |
| `GRAFANA_ADMIN_PASSWORD` | Sí | [compose.yaml](../compose.yaml) |
| `HEALTHCHECK_TOKEN` | Sí | [compose.yaml](../compose.yaml), [gateway/cmd/server/server.go](../gateway/cmd/server/server.go), [platform/cmd/healthcheck/main.go](../platform/cmd/healthcheck/main.go) |
| `IDENTITY_URL` | No | [gateway/cmd/server/server.go](../gateway/cmd/server/server.go) |
| `INSTANCE_ID` | No | [platform/loki.go](../platform/loki.go), [platform/metrics.go](../platform/metrics.go), [platform/runtime.go](../platform/runtime.go) |
| `INTERNAL_TOKEN` | Sí | [compose.yaml](../compose.yaml), [gateway/cmd/server/server.go](../gateway/cmd/server/server.go), [services/consumo/cmd/server/main.go](../services/consumo/cmd/server/main.go) |
| `JWT_SECRET` | Sí | [compose.yaml](../compose.yaml), [services/usuarios/cmd/server/main.go](../services/usuarios/cmd/server/main.go) |
| `LOKI_URL` | No | [platform/loki.go](../platform/loki.go) |
| `M2M_API_KEY` | Sí | [compose.yaml](../compose.yaml), [services/consumo/cmd/server/main.go](../services/consumo/cmd/server/main.go) |
| `M2M_DENIED_KEY` | No | [compose.yaml](../compose.yaml), [services/consumo/cmd/server/main.go](../services/consumo/cmd/server/main.go) |
| `MONGO_ROOT_PASSWORD` | Sí | [compose.yaml](../compose.yaml) |
| `MONGO_SOLAR_PASSWORD` | Sí | [compose.yaml](../compose.yaml) |
| `MONGO_URI` | No | [services/solar/internal/adapters/persistence/mongo.go](../services/solar/internal/adapters/persistence/mongo.go) |
| `MYSQL_ADDR` | No | [platform/sql.go](../platform/sql.go), [services/electrodomesticos/cmd/server/server.go](../services/electrodomesticos/cmd/server/server.go) |
| `MYSQL_DATABASE` | No | [platform/sql.go](../platform/sql.go), [services/electrodomesticos/cmd/server/server.go](../services/electrodomesticos/cmd/server/server.go) |
| `MYSQL_PASSWORD` | Sí | [compose.yaml](../compose.yaml), [platform/sql.go](../platform/sql.go), [services/electrodomesticos/cmd/server/server.go](../services/electrodomesticos/cmd/server/server.go) |
| `MYSQL_ROOT_PASSWORD` | Sí | [compose.yaml](../compose.yaml) |
| `MYSQL_TEST_ADDR` | No | [services/electrodomesticos/internal/repository/mysql_test.go](../services/electrodomesticos/internal/repository/mysql_test.go) |
| `MYSQL_TEST_PASSWORD` | Sí | [services/electrodomesticos/internal/repository/mysql_test.go](../services/electrodomesticos/internal/repository/mysql_test.go) |
| `MYSQL_USER` | No | [platform/sql.go](../platform/sql.go), [services/electrodomesticos/cmd/server/server.go](../services/electrodomesticos/cmd/server/server.go) |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | No | [platform/runtime.go](../platform/runtime.go) |
| `PANEL_COMPARATOR_API_KEY` | Sí | [compose.yaml](../compose.yaml), [services/solar/cmd/server/main.go](../services/solar/cmd/server/main.go) |
| `PANEL_COMPARATOR_OPENAPI` | No | [services/solar/internal/adapters/external/comparator_test.go](../services/solar/internal/adapters/external/comparator_test.go) |
| `PANEL_PROVIDER_ALLOW_HTTP` | No | [compose.yaml](../compose.yaml), [services/solar/cmd/server/main.go](../services/solar/cmd/server/main.go) |
| `PANEL_PROVIDER_API_KEY` | Sí | [compose.yaml](../compose.yaml), [services/solar/cmd/server/main.go](../services/solar/cmd/server/main.go), [services/solar/cmd/supplier/main.go](../services/solar/cmd/supplier/main.go) |
| `PANEL_PROVIDER_MODE` | No | [compose.yaml](../compose.yaml), [services/solar/cmd/server/main.go](../services/solar/cmd/server/main.go) |
| `PANEL_PROVIDER_URL` | No | [compose.yaml](../compose.yaml), [services/solar/cmd/server/main.go](../services/solar/cmd/server/main.go) |
| `PANEL_SIMULATOR_URL` | No | [services/solar/cmd/server/main.go](../services/solar/cmd/server/main.go) |
| `RABBITMQ_HOSTNAME` | No | [compose.yaml](../compose.yaml) |
| `RABBITMQ_PASSWORD` | Sí | [compose.yaml](../compose.yaml) |
| `SEARCH_API_KEY` | Sí | [compose.yaml](../compose.yaml), [services/electrodomesticos/internal/features/catalog.go](../services/electrodomesticos/internal/features/catalog.go), [services/solar/internal/adapters/search/client.go](../services/solar/internal/adapters/search/client.go) |
| `SEARCH_URL` | No | [services/electrodomesticos/internal/features/catalog.go](../services/electrodomesticos/internal/features/catalog.go), [services/solar/internal/adapters/search/client.go](../services/solar/internal/adapters/search/client.go) |
| `SOLAR_URLS` | No | [gateway/cmd/server/server.go](../gateway/cmd/server/server.go) |
| `USUARIOS_DB_PASSWORD` | Sí | [compose.yaml](../compose.yaml) |
| `USUARIOS_DB_ROOT_PASSWORD` | Sí | [compose.yaml](../compose.yaml) |
| `VALKEY_ADDR` | No | [services/electrodomesticos/internal/features/catalog.go](../services/electrodomesticos/internal/features/catalog.go) |
| `VALKEY_PASSWORD` | Sí | [compose.yaml](../compose.yaml), [services/electrodomesticos/internal/features/catalog.go](../services/electrodomesticos/internal/features/catalog.go) |
| `WEB_DIR` | No | [gateway/cmd/server/server.go](../gateway/cmd/server/server.go) |

[Publicación Render](DESPLIEGUE-GRATUITO.md), [entrada web y cookies HTTPS](PUBLICACION-WEB.md), [observabilidad](OBSERVABILITY.md) y [capacidad/costos](RECURSOS-LOCALES.md).


## Preparar las bases para presentar

Después de ejecutar pruebas, consultar [limpieza de entrega](LIMPIEZA-ENTREGA.md). La herramienta opcional retira restos generados y conserva las pruebas ejecutables; requiere Python 3 solo para ese mantenimiento. Los respaldos privados permanecen en .runtime y no se publican.
