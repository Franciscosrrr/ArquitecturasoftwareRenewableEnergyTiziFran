# Renewable Energy - Entrega 2

Sistema de Administración de Consumo Personalizado · Arquitectura de Software 2026 · UCC.

**Entrega 2: dominio propio operativo de punta a punta.** La cátedra aprobó el dominio y asignó proveedor/consumidor; el equipo aún no recibió su contrato ni documentación. Bonus anunciado: **tests e2e**.

| Integrante | Clave universitaria |
|---|---|
| Tiziano Nicolas | 2407889 |
| Francisco Skrie | 2423159 |

Repositorio: [ArquitecturasoftwareRenewableEnergyTiziFran](https://github.com/Franciscosrrr/ArquitecturasoftwareRenewableEnergyTiziFran). Grupo/comisión: número pendiente de registrar.

## Qué funciona

La calculadora React permite seleccionar fichas públicas, declarar hábitos y calcular energía sin registrarse. Gateway → Consumo → Electrodomésticos → MySQL resuelve la potencia real del catálogo y aplica las reglas. Los cálculos son temporales; recargar elimina el borrador. MySQL conserva únicamente el catálogo y sus semillas idempotentes.

La capacidad para otro grupo es **POST /api/v1/estimaciones-consumo**: recibe equipos externos, valida la entrada y calcula con aritmética decimal exacta, sin guardar datos privados. Conserva el contrato OpenAPI 1.0.1 y exige una clave M2M generada localmente. El mock de la entrega 1 es independiente y opcional.

## Arranque local

Requisito: Docker Desktop con motor Linux y Docker Compose v2. No hace falta instalar Go, Node, Python ni MySQL en el host. La primera ejecución descarga imágenes y dependencias.

Windows, desde la raíz del repositorio:

```powershell
.\scripts\compose.ps1 up -d --build
```

Linux/macOS:

```sh
sh scripts/compose.sh up -d --build
```

Abrir [calculadora](http://localhost:8080) y [Jaeger](http://localhost:16686). La interfaz y el visor se publican solo en localhost; MySQL y las APIs internas no exponen puertos al host. Los scripts generan `.env` aleatorio una sola vez. **No compartir ni subir ese archivo.** Conservarlo junto con el volumen MySQL para reiniciar con las mismas credenciales. No borrar `.env` mientras se conserva una base inicializada.

Para detener, usar el mismo script con `down`. Esto conserva el catálogo en su volumen. La calculadora no necesita usuarios ni sesiones; identidad, guardados e implementación solar siguen fuera del alcance de este hito.

## Pruebas y demostración

```powershell
.\scripts\verify.ps1
```

```sh
sh scripts/verify.sh
```

Los scripts construyen/inician el sistema y ejecutan tests Go, integración con MySQL real, pruebas HTTP de contrato y pruebas e2e en Chromium para escritorio y móvil. El navegador e2e se descarga en una imagen oficial, que puede tardar en la primera ejecución. El reporte e2e queda en `docs/evidence/entrega2/e2e/`.

Escenario de exposición: 30 días; cuatro lámparas de 10 W, 5 h/día y factor 1; una heladera de 150 W, 24 h/día y factor 0,3; un lavarropas de 0,75 kWh/ciclo con 12 ciclos. Resultado: **47,4 kWh** en el período y **1,58 kWh/día**. Son referencias didácticas, no fichas universales ni mediciones.

Prueba M2M reproducible sin imprimir su clave:

```powershell
.\scripts\demo-m2m.ps1
```

En Jaeger seleccionar el servicio `gateway` y buscar una operación de cálculo. El `traceId` mostrado en la calculadora permite abrir la misma traza. [Guion y evidencias](docs/DELIVERY-2.md).

## Documentación

- [Matriz de cumplimiento y guion de entrega 2](docs/DELIVERY-2.md).
- [Arquitectura implementada y evolución](docs/ARCHITECTURE.md).
- [Alcance y requisitos](SPEC.md).
- [ADR D1 a D13](docs/adr/README.md).
- [Capacidad compartida y errores](docs/contracts/README.md).
- [Contrato adicional de catálogo](docs/contracts/catalog-v1/openapi.json).
- [Observabilidad y recuperación](docs/OBSERVABILITY.md).
- [Bonus e2e](docs/BONUS-E2E.md).
- [Caída controlada](docs/POSTMORTEM.md).
- [Revisión histórica de entrega 1](docs/REVIEW.md).

## Alcance pendiente

Sin URL cloud todavía. La publicación pública de la capacidad se reserva para la presentación final y necesita acordar hosting y credenciales con el grupo consumidor. No se contrató ningún servicio pago. El costo local de licencias es cero; CPU, RAM, disco y electricidad pertenecen al equipo anfitrión.

Usuarios/roles, configuraciones guardadas, Solar, MongoDB, broker/outbox, Meilisearch, Valkey, balanceo entre réplicas y tablero de métricas siguen pendientes. Los ADR de esos temas distinguen decisión de diseño y evidencia de implementación. El dominio aprobado no equivale a una integración externa ya validada.

## Forma de trabajo

Un repositorio público, ramas cortas y PR revisados hacia main como versión evaluable. Se incluye CI de comprobación, que se ejecutará al subir los cambios. Esta preparación local no realiza publicaciones, commits ni push automáticamente.
