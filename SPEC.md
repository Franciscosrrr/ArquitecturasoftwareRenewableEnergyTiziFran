# Especificación de alcance y requisitos

Versión 1.2.0 · Entrega 1. Estado: especificación de diseño; criterios pendientes de implementación y prueba.

## 1. Objetivo y vocabulario

Estimar energía consumida por un lugar y recomendar paneles sin exigir cuenta. El modo invitado solo muestra resultados temporales. Iniciar sesión habilita conservar escenarios editables e historial propio.

| Concepto | Significado |
|---|---|
| Lugar | Casa, campo, comercio u otro espacio administrado por un usuario |
| Configuración de consumo | Escenario con nombre, período, equipos y hábitos; editable y versionado |
| Evaluación | Snapshot inmutable de una configuración y su lugar, con desglose y totales |
| Estudio solar | Trabajo con entradas congeladas, estado y comparación de alternativas |
| Snapshot | Copia explícita de datos y su versión utilizada para reproducir un resultado |
| W / kWh | Potencia instantánea / energía de un período; no son unidades intercambiables |
| HSP | Horas solares pico: irradiación equivalente; no horas de luz del día |

## 2. Actores y permisos

| Acción | Invitado | Usuario común | Administrador | Cliente de otro grupo |
|---|---|---|---|---|
| Abrir calculadora y consultar catálogos activos | Sí | Sí | Sí | Fuera de la capacidad M2M v1 |
| Calcular y ver consumo y recomendación temporal | Sí | Sí | Sí | Solo estimación de consumo con credencial M2M |
| Registrarse / iniciar sesión | Sí | Sí | Sí | No requerido |
| Perfil propio | No | Sí | Sí | No |
| Guardar/crear/editar/duplicar configuraciones propias | No | Sí | Sí, solo propias | No |
| Consultar historial y estudios propios | No | Sí | Sí, solo propios | No |
| Crear/editar/desactivar electrodomésticos y paneles | No | No | Sí | No |
| Administrar roles | No | No | Sí | No |
| Estimar por API de otro grupo con X-API-Key | No requerido para la web | Credencial M2M independiente | Credencial M2M independiente | Sí |

El permiso administrativo de catálogos no habilita lectura de consumos ajenos. La autorización se aplica en backend. El rol enviado por un registro público no se acepta.

## 3. Requisitos funcionales y aceptación

| ID | Requisito | Criterio de aceptación |
|---|---|---|
| RF01 | Registro | Un email normalizado único crea un usuario común; solicitudes concurrentes no crean duplicados |
| RF02 | Login, refresh y logout | Credenciales válidas habilitan guardados/perfil; la calculadora no exige login; logout revoca refresh; access token tiene vencimiento acotado |
| RF03 | Privacidad | Cambiar ids en una URL no permite acceder a lugares, guardados o estudios ajenos |
| RF04 | Lugares | Crear, listar, editar y archivar lugares propios con ubicación y superficie opcional |
| RF05 | Configuraciones guardadas | Guardar con nombre, listar en Mis consumos, abrir y restaurar todos los equipos y hábitos |
| RF06 | Edición y duplicación | Editar incrementa versión; duplicar crea un escenario independiente sin cambiar el original |
| RF07 | Catálogo precargado | El arranque inicial crea referencias genéricas con unidades y fuentes; posteriores arranques no duplican semillas |
| RF08 | Búsqueda | Texto, categoría, modo, potencia cuando corresponda, páginas y orden; cambios aparecen automáticamente |
| RF09 | Agregar equipos | Se valida ficha activa y cantidad/hábitos; el invitado mantiene datos en memoria y el guardado autenticado conserva snapshot |
| RF10 | Evaluación autenticada | Se guarda un snapshot coherente con desglose, total y promedio diario; edición futura no lo altera |
| RF11 | Idempotencia | Repetir evaluación con igual clave/entrada devuelve el mismo resultado; distinta entrada con igual clave produce conflicto |
| RF12 | Historial | Una configuración v2 mantiene evaluaciones de v1 consultables y marca resultados de versiones anteriores |
| RF13 | Alta administrativa de equipos | Solo ADMINISTRADOR crea/modifica/desactiva fichas y estas persisten en MySQL |
| RF14 | Alta administrativa de paneles | Solo ADMINISTRADOR crea/modifica/desactiva fichas y estas persisten en MongoDB |
| RF15 | Estudio solar persistente | Una evaluación autenticada confirmada dispara trabajo durable; el usuario ve estado y resultado por evaluación |
| RF16 | Comparación solar | Muestra modelo, cantidad, generación, cobertura, superficie, fuente y criterio de selección |
| RF17 | Restricciones solares | Sin superficie suficiente informa objetivo inviable y cobertura alcanzable; sin recurso confiable no inventa un resultado |
| RF18 | Cambiar cobertura | Solicitar otra cobertura crea otro estudio/configuración solar, conservando los anteriores |
| RF19 | Capacidad pública | El consumidor estima equipos externos según OpenAPI v1, sin consultar datos privados |
| RF20 | Integración externa | La respuesta del grupo proveedor influye en una decisión o resultado del flujo; no basta una llamada aislada |
| RF21 | Acceso invitado | Sin cookies ni credenciales se abre la calculadora, se consultan catálogos activos y se obtiene consumo y recomendación solar |
| RF22 | Sin guardado anónimo | El cálculo invitado no crea filas/documentos de consumo o estudios, outbox ni mensajes de negocio; recargar pierde el borrador |
| RF23 | Guardado explícito | Guardar solicita login; al volver conserva el borrador solo en memoria, y recién una acción Guardar confirmada valida y persiste bajo el usuario |
| RF24 | Protección de rutas | Solicitudes anónimas a guardados/historial/administración reciben 401; un rol insuficiente recibe 403 |

## 4. Reglas de consumo

Para potencia: **E_período_kWh = potenciaW × cantidad × horasPorDia × diasUso × factorFuncionamiento / 1000**.

Para ciclos: **E_período_kWh = energiaPorCicloKWh × cantidad × ciclosPeriodo**. ciclosPeriodo se refiere a cada unidad del equipo, por eso se multiplica por cantidad.

**E_total = suma de equipos; promedioDiario = E_total / diasPeriodo**. Se estima un período convencional de 30 días por defecto. Un mes calendario usa sus días reales si esa modalidad se selecciona.

El factor representa la proporción de demanda durante el tiempo declarado. Una potencia ya expresada como media no recibe otra reducción por el mismo ciclo de funcionamiento. Las fuentes del catálogo y las hipótesis se muestran en el resultado.

Reglas mínimas: cantidad entera positiva; potencia/energía por ciclo positivas; horas entre 0 y 24; días de uso entre 0 y días del período; factor mayor que 0 y hasta 1; ciclos no negativos; identificadores de entrada únicos. No se suman W como si fueran kWh.

La evaluación toma versiones esperadas de lugar/configuración; ante conflicto de edición debe actualizarse la información. Cambiar una ficha del catálogo no actualiza guardados silenciosamente: se permite una actualización explícita, con nueva versión y aviso de impacto.

Los valores de API y límites de carga de la capacidad externa se especifican de manera exacta en [contratos](docs/contracts/README.md). El algoritmo v1 conserva precisión decimal durante el cálculo y redondea salidas a seis decimales con mitad hacia arriba; no redondea cada entrada antes de sumar.

## 5. Reglas solares

**Generación diaria/panel = potenciaW / 1000 × HSP × rendimientoGlobal**.

**Demanda objetivo = promedioDiario × coberturaObjetivo**; **cantidad requerida = techo(demandaObjetivo / generaciónPorPanel)**.

Si hay superficie: **cantidad que cabe = piso(superficieUtil / superficieEfectivaPanel)**. La superficie efectiva registra la reserva por disposición; no verifica geometría ni resistencia estructural.

La cobertura admitida inicialmente es mayor que 0 y hasta 100 %. Se descartan fichas inactivas o inválidas. Se comparan candidatos por cumplimiento de energía/superficie; si todos los factibles tienen precios comparables en igual moneda y referencia temporal, gana menor costo de paneles, luego menor superficie e id. Si faltan precios comparables, gana menor superficie requerida, luego menor cantidad e id. Se informa el criterio aplicado.

Si nadie cumple superficie, se muestra la mayor cobertura alcanzable sin presentarla como cumplimiento del objetivo. Consumo cero produce cero paneles. Sin candidatos se completa con resultado sin alternativa. Sin ubicación/recurso confiable se requiere información adicional o se informa un error recuperable; un recurso manual se etiqueta con su fuente.

Si el proveedor ya devuelve generación neta, no se aplican otra vez sus pérdidas. Cada estudio persistente conserva parámetros, fichas y versiones, fuente/fecha, orientación e inclinación asumidas. La recomendación invitada incluye esa información en la respuesta temporal, sin escribirla en una base. La cobertura del período no equivale a autonomía horaria ni alimentación durante cortes.

El modo invitado utiliza las mismas reglas de consumo y selección solar. Un valor declarado para simular no es un alta de catálogo; los paneles candidatos siempre se obtienen del catálogo activo de Solar. Para la promoción del borrador se vuelven a validar fichas activas y permisos. No se acepta un resultado calculado por el navegador como evaluación confirmada.

## 6. Estados

| Entidad | Estados |
|---|---|
| Lugar | Activo / archivado |
| Configuración | Editable, versionada; archivada si corresponde |
| Evaluación | Confirmada e inmutable; publicación técnica pendiente / enviada / intervención |
| Estudio | Pendiente → en procesamiento → completado / sin alternativa / error |

Estos estados durables corresponden al modo autenticado. El invitado solo tiene estados visuales en memoria: editando → calculando → resultado / sin alternativa / error; no tiene id de evaluación ni endpoint de historial.

Reintentar error vuelve a pendiente con auditoría. Un respaldo solar válido se etiqueta como degradado. Tras guardar una evaluación, el usuario puede ver “Recomendación pendiente”; no debe interpretarse ausencia temporal del estudio como pérdida del consumo.

## 7. Requisitos no funcionales

| ID | Compromiso | Evidencia futura |
|---|---|---|
| RNF01 | Cuatro servicios y gateway; capas/hexagonal distinguibles | Despliegues y revisión de dependencias |
| RNF02 | Base privada por servicio; MySQL + MongoDB | Credenciales y pruebas sin acceso cruzado |
| RNF03 | Arranque docker compose up --build sin instalación extra | Máquina limpia con Docker/Compose |
| RNF04 | Persistencia tras reinicios | Reabrir usuarios/guardados con volúmenes preservados |
| RNF05 | REST con deadlines y RabbitMQ durable/idempotente | Integración, duplicados y recuperación |
| RNF06 | Consistencia de evaluación y outbox local | Transacciones, rollback, concurrencia y respuesta perdida |
| RNF07 | Índice automático, objetivo de atraso ≤5 s en estado saludable | Medir commit a visibilidad, incluida tarea de indexación |
| RNF08 | Caché relevante con TTL e invalidación | Hit ratio y comparación de carga con/sin caché |
| RNF09 | Dos Solar con health checks y balanceo | Solicitudes por instancia y caída provocada |
| RNF10 | Circuit breaker, reintentos limitados y DLQ | Ensayo y POSTMORTEM posterior |
| RNF11 | Logs, métricas, trazas, tablero y alertas | Recorrido completo correlacionado |
| RNF12 | Unitarias, integración real, contrato y carga | Informes y ejecución reproducible |
| RNF13 | Repositorio público y documentación vigente | main evaluable, PR y CI |
| RNF14 | Componentes/hosting exclusivamente gratuitos | Ediciones verificadas y cuotas; ningún upgrade pago |
| RNF15 | Privacidad invitada y control de abuso | Verificar ausencia de escrituras/eventos, ausencia de cuerpos sensibles en logs y límites de las rutas anónimas |

Objetivos iniciales, todavía no medidos: p95 de catálogo <500 ms; p95 de confirmación <1 s con hasta 100 equipos; 95 % de estudios visibles <15 s con proveedor saludable. La activación en frío cloud se informa separadamente y no se oculta en estas métricas. La cobertura unitaria propuesta es 80 % de ramas del dominio y todos los invariantes críticos; el PDF no fija un porcentaje preciso y debe validarse con la cátedra.

## 8. Exclusiones y pendientes

No se incluye IoT, facturación/pagos, venta, instalación, baterías, diseño eléctrico/estructural, normativa de conexión, autonomía, imágenes adjuntas ni importación masiva. “Subir un equipo/panel” significa alta de ficha por formulario administrativo.

Pendientes externos: aprobación docente, número de grupo/comisión, proveedor/consumidor asignados y contrato, fuente solar para ubicaciones reales, dataset con fuentes, reglas de precio/moneda y validación del hosting gratuito. Los resultados de carga/fallas y la integración no se declaran ya realizados.
