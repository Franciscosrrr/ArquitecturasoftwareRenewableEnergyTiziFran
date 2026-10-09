# ArquitecturasoftwareRenewableEnergy

Sistema de Administración de Consumo Personalizado

**Trabajo Práctico Integrador · Arquitectura de Software · Entrega 1: diseño, estructura y mock**

Versión documental: 1.2.0.

| Dato académico | Estado |
|---|---|
| Proyecto | ArquitecturasoftwareRenewableEnergy |
| Grupo / comisión | Número no informado |
| Repositorio | [ArquitecturasoftwareRenewableEnergyTiziFran](https://github.com/Franciscosrrr/ArquitecturasoftwareRenewableEnergyTiziFran) |
| Aprobación docente del dominio | Pendiente de confirmar |
| Grupo proveedor / consumidor | Asignación docente pendiente |

| Integrante | Clave universitaria |
|---|---|
| Tiziano Nicolas | 2407889 |
| Francisco Skrie | 2423159 |

## Presentación

El sistema permite estimar el consumo energético de una casa, campo, comercio u otro lugar a partir de sus equipos y hábitos de uso. Cualquier visitante podrá calcular consumo y ver una recomendación solar sin registrarse. El resultado del invitado será temporal y no se guardará. Los usuarios que inicien sesión podrán guardar configuraciones, seleccionarlas, editarlas y recalcularlas. A partir de una evaluación se recomendarán un modelo y una cantidad de paneles solares, considerando cobertura objetivo, recurso solar y superficie disponible.

La operación central combina reglas energéticas, validaciones, restricciones físicas, estados y procesamiento distribuido. El proyecto no se limita a altas, bajas y consultas.

## Arquitectura elegida

| Servicio | Estilo interno | BD propia |
|---|---|---|
| Usuarios | En capas | MySQL usuarios_db |
| Electrodomésticos | En capas | MySQL electrodomesticos_db |
| Consumo | En capas | MySQL consumo_db |
| Solar | Hexagonal | MongoDB solar_db |

Los servicios y el gateway se desarrollarán en Go con Gin; la interfaz será React con JavaScript. Los tres servicios energéticos conservan dos arquitecturas en capas y una hexagonal. Usuarios incorpora identidad como cuarto servicio en capas. Ningún servicio accede a la base de otro: se integran mediante API y eventos, con snapshots versionados.

## Flujo principal

1. Entrar directamente a la calculadora, sin registro ni login.
2. Seleccionar electrodomésticos del catálogo y declarar cantidad y uso.
3. Calcular el consumo por equipo, el total y el promedio diario.
4. Indicar los datos solares y ver una recomendación temporal de modelo y cantidad de paneles.
5. Modificar los datos y recalcular mientras la página esté abierta.
6. Si se desea conservar el escenario, registrarse o iniciar sesión y pulsar **Guardar** de forma explícita.
7. Desde **Mis consumos**, abrir y editar configuraciones propias y consultar evaluaciones y estudios persistentes.

El modo invitado no crea lugares, configuraciones, evaluaciones, estudios ni eventos de negocio. Recargar o cerrar la página elimina sus datos temporales. El modo autenticado mantiene el flujo durable de evaluación y estudio solar asíncrono. Ver [modo invitado](docs/GUEST-MODE.md).

El login se exige para guardar, recuperar información privada y administrar. Solo administradores podrán crear, modificar o desactivar electrodomésticos y paneles del catálogo. El registro público crea usuarios comunes.

## Estado de esta entrega

Versión documental 1.2.0. Incluye documentación, diagramas, contratos OpenAPI y ejemplos, un mock HTTP ejecutable y estructura compilable de cuatro servicios Go y gateway. Los casos de negocio reales, persistencia, frontend React, broker y despliegue cloud siguen pendientes.

El mock es una herramienta local en Python estándar, aislada del backend Go previsto. No requiere Python en el host si se usa Docker. Las estructuras Go usan net/http para el arranque mínimo; Gin y los drivers se agregarán al implementar cada caso de uso. No se presenta la estructura como una implementación completa de capas o hexagonal.

Ver [verificación](docs/REVIEW.md), [mock](docs/contracts/MOCK.md) y [estructura](docs/STRUCTURE.md).

## Ejecución local de la primera entrega

Desde la raíz, con Docker Desktop iniciado y Compose v2:

Windows (PowerShell):

```powershell
.\scripts\compose.ps1 up --build
```

Linux/macOS:

```sh
sh scripts/compose.sh up --build
```

El script genera una credencial aleatoria HEALTHCHECK_TOKEN de 32 bytes en memoria y la pasa a los contenedores. No crea .env, no la imprime y no usa una clave fija. Si ya está definida, la conserva; para invocar Docker Compose directamente debe suministrarse esa variable.

Inicia el mock en http://localhost:8080. No inicia todavía la aplicación completa ni una interfaz web. Contrato M2M: http://localhost:8080/openapi.json; contrato invitado: http://localhost:8080/guest-openapi.json. En navegador se pueden abrir estos contratos; las operaciones de negocio requieren POST.

Prueba desde PowerShell, Linux o macOS (en Windows usar `curl.exe`):

```sh
curl -i http://localhost:8080/api/v1/estimaciones-consumo -H "Content-Type: application/json" -H "X-API-Key: mock-consumidor" --data-binary @docs/contracts/v1/examples/solicitud-valida.json
```

Resultado esperado: HTTP 200, total 47,4 kWh y promedio 1,58 kWh/día. Las claves `mock-consumidor` y `mock-sin-permiso` son datos públicos de simulación sin acceso a sistemas reales.

```sh
sh scripts/compose.sh --profile verificar run --build --rm verificar
sh scripts/compose.sh --profile estructura up --build
```

En PowerShell, reemplazar `sh scripts/compose.sh` por `.\scripts\compose.ps1`. El primer comando ejecuta pruebas HTTP aisladas dentro del contenedor. El segundo inicia también los esqueletos: Usuarios 8101, Electrodomésticos 8102, Consumo 8103, Solar 8104 y gateway 8105. `/health/live` y `/health/ready` exigen `Authorization: Bearer <HEALTHCHECK_TOKEN>`; sin credencial válida responden 401 sin revelar estado. Con credencial válida devuelven 200 y 503 respectivamente; negocio devuelve 501 porque no está implementado. Las respuestas de salud permanecen en español y no se almacenan en caché. El gateway inicial todavía no enruta al mock ni implementa balanceo.

Para detener: `.\scripts\compose.ps1 --profile estructura down` en Windows o `sh scripts/compose.sh --profile estructura down` en Linux/macOS. El mock no tiene volúmenes ni guarda información de negocio. Arranque completo con bases, seeds, colas e interfaz: pendiente de implementación. La validación Docker de esta revisión cubre configuración Compose; el build y arranque en Docker quedan pendientes porque el motor no estaba activo.

## Capacidad para otro grupo

Consumo publicará **POST /api/v1/estimaciones-consumo** para estimar energía de equipos y hábitos sin acceder a guardados privados. El contrato v1.0.1, errores, autenticación M2M, ejemplos y alcance del mock están en [docs/contracts/README.md](docs/contracts/README.md). URL cloud: pendiente de publicación; no existe aún un entorno productivo; el mock local sí está incluido.

Todos los componentes locales serán gratuitos. La publicación elegirá un plan gratuito sin medio de pago ni mejoras pagas; la propuesta Render Free requiere validar activación en frío y cuotas. No se contratarán servicios pagos.

## Documentación

- [Alcance, requisitos y aceptación](SPEC.md).
- [Arquitectura y diagramas](docs/ARCHITECTURE.md).
- [Estructura y dependencias previstas](docs/STRUCTURE.md).
- [Decisiones arquitectónicas](docs/adr/README.md).
- [Contrato público y ejemplos](docs/contracts/README.md).
- [Matriz de la primera entrega y pendientes](docs/DELIVERY-1.md).
- [Revisión documental realizada](docs/REVIEW.md).
- [Coherencia del proyecto y correcciones](docs/CONSISTENCY-REVIEW.md).
- [Auditoría de seguridad y resumen del proyecto](docs/SECURITY-AUDIT.md).
- [Modo invitado y guardado opcional](docs/GUEST-MODE.md).
- [Fuentes](docs/REFERENCES.md).
- [Propuesta revisada](docs/PROPUESTA-REVISADA.md).
- [Cambios recomendados y relación con clases](docs/REVISION-PROPUESTA.md).

## Forma de trabajo

Se utilizará un repositorio público único, main como rama evaluable y ramas cortas por tarea. Las modificaciones se integrarán mediante PR revisados. La CI futura verificará pruebas, contratos y build de contenedores. Las decisiones y el alcance se actualizarán durante el desarrollo; un ADR reemplazado conservará su historia.
