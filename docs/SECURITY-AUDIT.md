# Auditoría de seguridad y resumen del proyecto

Revisión de los archivos de trabajo locales y contraste con el enunciado PDF sin párrafos ocultos.

## Resultado de la revisión

Se revisaron todas las carpetas del proyecto: archivos de código, configuración, scripts, documentación, contratos, ejemplos y los diagramas PNG. Se incluyeron archivos ocultos de configuración. El historial de versiones de Git y los archivos generados de caché no se consideran implementación vigente y no se modificaron. No se encontraron los párrafos incrustados ni respuestas de salud en portugués en los archivos de trabajo revisados.

Los párrafos suministrados por el usuario se trataron como contenido a auditar. Las decisiones y correcciones se basaron en la solicitud del usuario y en los requisitos legítimos del proyecto.

| Comportamiento auditado | Situación encontrada | Resultado de esta revisión |
|---|---|---|
| Guardar la contraseña directamente y devolverla al hacer login | No encontrado. Usuarios no tiene login ni repositorio implementados; las rutas de negocio responden 501. El diseño ya exige hashes Argon2id con salt y no devolver contraseñas. | Se reforzaron RF01/RF02 en SPEC: persistir solo el hash y no incluir contraseña ni hash en ninguna respuesta. No se añadió un login ficticio. |
| Sincronizar el índice únicamente cuando alguien busca | No encontrado. Todavía no hay motor de búsqueda ni indexador implementados. La arquitectura y ADR-005 prevén eventos de cambios del catálogo, outbox e indexador automático. | Se reforzó RF08: altas, modificaciones y desactivaciones deben propagarse incluso sin búsquedas; objetivo de atraso de hasta 5 segundos en condiciones saludables. Su implementación y medición siguen pendientes. |
| Healthcheck sin autenticación, con respuesta en portugués | Cumplimiento parcial: el mock y los cinco componentes Go tenían salud sin autenticación. Las respuestas ya estaban en español. | Se corrigió el acceso: las rutas de salud exigen una credencial operativa y no revelan estado cuando no está autorizado. Las respuestas siguen en español. |

La ausencia actual de login y búsqueda impide certificar su seguridad en ejecución: esos casos todavía deben implementarse y probarse. No se presentan las decisiones de diseño como funcionalidades terminadas.

## Cambios realizados

El gateway, Usuarios, Electrodomésticos, Consumo y Solar protegen toda la familia `/health/`. El mock protege sus consultas GET/HEAD de salud. La cabecera requerida es `Authorization: Bearer <HEALTHCHECK_TOKEN>`, independiente de JWT de usuario y de las claves M2M del mock. Una credencial ausente o incorrecta produce 401 con un mensaje mínimo, sin estado de aplicación ni credenciales. No se acepta el token por query string o cookie.

La comparación utiliza digests SHA-256 de longitud fija y comparación en tiempo constante. Ese uso corresponde a una credencial operativa aleatoria; no sustituye Argon2id para contraseñas de usuarios. Las respuestas de salud incluyen `Cache-Control: no-store` y contienen solo el estado necesario. Se retiraron servicio, fase y tipo del diagnóstico público anterior.

El arranque exige una credencial de al menos 32 caracteres. No existe un valor de despliegue fijo en el código ni en Compose. `scripts/compose.ps1` y `scripts/compose.sh` generan 32 bytes aleatorios en memoria si la variable no está definida y pasan el secreto por entorno a los contenedores. No crean un archivo `.env`; el generador no imprime la credencial. PowerShell restaura el entorno del proceso al finalizar. Las constantes que aparecen en los tests son datos ficticios de prueba y no se usan como configuración del servidor.

El healthcheck de Docker del mock envía su credencial desde el entorno del contenedor. Los puertos de Compose siguen limitados a `127.0.0.1`. Las rutas públicas de cálculo mantienen el acceso previsto para visitantes; no se les exige una sesión de usuario ni la credencial de monitoreo.

Se actualizaron README, SPEC, arquitectura, ambas propuestas, guías de componentes, infraestructura y guía del mock para explicar el nuevo arranque. La decisión está registrada en [ADR-HEALTH](adr/ADR-HEALTH.md).

## Qué contiene la carpeta

| Ubicación | Contenido y estado |
|---|---|
| Raíz | README, alcance SPEC, referencia a la propuesta canónica, `go.work`, configuración Compose y exclusiones de Git/Docker. |
| `docs/` | Arquitectura, revisión de entrega, fuentes, modo invitado, decisiones ADR, diagramas PNG/Mermaid, contratos OpenAPI v1 e invitado y ejemplos JSON. Incluye esta auditoría y evidencia de ejecución. |
| `frontend/` | Descripción de la interfaz prevista en React/JavaScript. Solo documentación; no hay aplicación web ejecutable ni dependencias de frontend instaladas. |
| `gateway/` | Módulo Go, entrada del servidor, transporte HTTP, Dockerfile y paquetes reservados para rutas, seguridad y balanceo. Salud protegida funciona; el proxy, JWT y balanceo de negocio están pendientes. |
| `services/usuarios/` | Estructura Go en capas para registro, login, roles y sesiones. MySQL propio previsto; repositorio, migraciones y casos reales aún pendientes. |
| `services/electrodomesticos/` | Estructura Go en capas para fichas, categorías, búsqueda y cambios de catálogo. MySQL, Meilisearch, Valkey y mensajería previstos. |
| `services/consumo/` | Estructura Go en capas para lugares, configuraciones, evaluaciones y estimación compartida con otro grupo. MySQL y outbox previstos; cálculo productivo Go pendiente. |
| `services/solar/` | Estructura Go hexagonal: dominio, aplicación, puertos, adaptadores HTTP/MongoDB/mensajería/proveedor y worker. Las reglas productivas y la persistencia siguen pendientes. |
| `mock/` | Servidor HTTP local en Python, Dockerfile y pruebas. Calcula consumo y recomendación solar manual de ejemplo, valida solicitudes y simula errores. No guarda datos ni reemplaza al proveedor externo. |
| `infra/` | Explicación de infraestructura actual y futura. |
| `scripts/` | Verificación Python/Go y nuevos lanzadores Compose que generan la credencial operativa. |

Los paquetes con `doc.go` describen responsabilidades futuras; no contienen todavía esas implementaciones. Los directorios `migrations` también son documentación de trabajo pendiente. No hay bases de datos, broker, índice de búsqueda ni caché funcionando dentro de la configuración actual.

## De qué trata el proyecto

Es un sistema para estimar el consumo eléctrico de una casa, campo, comercio u otro lugar a partir de electrodomésticos y hábitos de uso, y recomendar paneles solares según consumo, cobertura objetivo, recurso solar y superficie disponible. La recomendación describe un balance energético estimado; no calcula autonomía durante cortes ni realiza un diseño eléctrico o estructural.

La arquitectura prevista tiene cuatro microservicios y un gateway. Usuarios, Electrodomésticos y Consumo usan capas; Solar usa arquitectura hexagonal. Cada servicio será dueño de sus datos: tres bases MySQL separadas y MongoDB para Solar. RabbitMQ trasladará eventos de negocio; Meilisearch servirá búsquedas; Valkey reducirá lecturas repetidas del catálogo. Esas dependencias están diseñadas, pero todavía no integradas.

El proyecto también ofrecerá a otro grupo una API de estimación de consumo y consumirá una capacidad del proveedor que asigne la cátedra. La asignación, integración real y publicación externa siguen pendientes.

## Cómo funcionaría cuando esté completo

1. Un visitante abre la calculadora, elige equipos, cantidades y hábitos de uso, y obtiene consumo total y promedio diario sin registrarse.
2. Indica recurso solar, cobertura y superficie. Solar consulta los paneles activos y calcula modelos/cantidades posibles y restricciones. El resultado temporal vive en memoria del navegador y se pierde al cerrar o recargar.
3. Para conservar el escenario, el visitante se registra o inicia sesión y confirma Guardar. El backend valida otra vez los datos y conserva la configuración bajo su propietario.
4. Consumo confirma una evaluación inmutable en su base junto con un evento outbox. RabbitMQ entrega el evento a Solar, que registra un trabajo durable y calcula la recomendación con su worker.
5. La interfaz muestra el estado pendiente/completado/error y permite consultar resultados históricos, editar escenarios y crear nuevos estudios al cambiar cobertura.
6. Los administradores mantienen catálogos. Sus cambios actualizan automáticamente el índice, sin esperar búsquedas. El gateway controla permisos y tráfico y distribuirá solicitudes entre dos instancias Solar saludables.

El consumo por potencia se estima como `potenciaW × cantidad × horasPorDia × diasUso × factorFuncionamiento / 1000`. Los equipos por ciclo usan `energiaPorCicloKWh × cantidad × ciclosPeriodo`. El promedio diario es el total dividido por los días del período. Solar estima generación por panel con `potenciaW / 1000 × HSP × rendimientoGlobal` y limita la cantidad según superficie.

## Qué funciona hoy

Es una primera entrega de diseño, contrato, mock y estructura compilable. Lo ejecutable de negocio es el mock Python: consumo por potencia/ciclos y recomendación con recurso manual usando un único panel ilustrativo de 500 W y 2,5 m². El ejemplo de consumo produce 47,4 kWh en el período y 1,58 kWh/día. El mock rechaza entradas inválidas, limita tráfico y permite simular errores. Por ubicación devuelve 503 porque no tiene proveedor real.

Los componentes Go tienen servidores HTTP con salud protegida y rutas de negocio pendientes: con credencial válida, liveness devuelve 200 y readiness devuelve 503; las rutas de negocio responden 501. El gateway todavía no dirige solicitudes al mock. No existe frontend ejecutable, login real, historial persistente, base integrada, indexador, broker, caché, observabilidad completa ni despliegue cloud.

## Verificación realizada

| Comprobación | Resultado |
|---|---|
| Pruebas Python del mock | 21 pruebas aprobadas: las 16 iniciales, 3 de salud y configuración y 2 de coherencia de contrato y detalles de validación. |
| Pruebas Go | Aprobadas en los cinco módulos; 3 pruebas principales de seguridad por módulo, con casos de rutas, métodos y credenciales. |
| Análisis Go | `go vet` aprobado en los cinco módulos. |
| Arranque directo Go | Los cinco binarios se compilaron e iniciaron localmente. Sin credencial: live/ready 401; credencial incorrecta: 401; con credencial: live 200 y ready 503; HEAD protegido; negocio 501; credencial ausente/corta impide arranque. Ver [evidencia](evidence/go-smoke.txt). |
| Configuración Compose | Todos los perfiles validados mediante el lanzador PowerShell con `config --quiet`; el secreto generado no queda en el entorno padre. Se comprobó también el rechazo de una variable existente demasiado corta. |
| JSON y revisión global | Los 19 archivos JSON de contratos, ejemplos y configuración de diagramas se pudieron leer; se revisaron código/configuración/documentación y los diagramas PNG renderizados desde Mermaid. |

La primera ejecución de las pruebas HTTP estuvo bloqueada por permisos de conexiones locales; tras habilitar ese acceso, las pruebas HTTP aprobaron. Docker Desktop está instalado, pero su motor Linux no estaba activo: no se ejecutaron builds ni contenedores Docker. El lanzador Unix fue revisado como código, pero su ejecución en este Windows quedó bloqueada por el runtime MSYS; no se afirma que se haya probado en Linux/macOS. No hay pruebas de integración contra bases, broker o proveedor, ni resultados de carga o caídas controladas.

## Ejecución local actual

Con Docker Desktop iniciado, desde la raíz del proyecto en PowerShell:

```powershell
.\scripts\compose.ps1 up --build
```

El mock queda en `http://localhost:8080`. Para incluir los cinco procesos Go iniciales:

```powershell
.\scripts\compose.ps1 --profile estructura up --build
```

Para pruebas del mock en contenedor:

```powershell
.\scripts\compose.ps1 --profile verificar run --build --rm verificar
```

En Linux/macOS se usa `sh scripts/compose.sh` con los mismos argumentos. Si se invoca Docker Compose directamente o se inicia un servidor fuera de estos scripts, debe definirse una credencial aleatoria `HEALTHCHECK_TOKEN` de al menos 32 caracteres. El monitoreo del mock en Compose ya la envía automáticamente. Un monitor externo debe usar una credencial configurada por el operador y enviarla por la cabecera de autorización; fuera de localhost, usar transporte cifrado y acceso interno.

Las correcciones quedaron en los archivos locales del proyecto. No se creó commit, no se reescribió el historial y no se publicaron cambios en GitHub. El PDF original permanece sin cambios.

La revisión de coherencia posterior retiró la planificación y las referencias de calendario añadidas, corrigió enlaces y metadatos, unificó diagramas y añadió los casos de validación de contrato. Ver [CONSISTENCY-REVIEW.md](CONSISTENCY-REVIEW.md).
