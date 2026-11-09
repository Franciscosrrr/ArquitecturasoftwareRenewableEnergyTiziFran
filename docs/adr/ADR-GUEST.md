# ADR-GUEST — Calculadora anónima sin persistencia

Versión: 1.0.0. Estado: **Propuesto para implementación**. Complementa D1/D8; no es una nueva decisión numerada del enunciado. Autores: Tiziano Nicolas (2407889) y Francisco Skrie (2423159).

## Contexto

El visitante debe usar la página y conocer consumo y paneles sin registrarse. Solo una cuenta puede conservar configuraciones e historial. El procesamiento solar durable previsto creará documentos, por lo que no corresponde usarlo para el invitado.

## Decisión

Mantener cuatro servicios y sus arquitecturas. Abrir consultas de catálogos activos y agregar rutas de cálculo/recomendación temporal, síncronas y sin escrituras ni eventos de negocio. Compartir reglas del dominio con el flujo autenticado, con puertos/casos de uso separados para impedir persistencia accidental. El borrador vive en memoria del navegador. El guardado requiere sesión y acción explícita; se revalida y recalcula.

## Alternativas

Exigir login para calcular contradice el alcance. Crear cuentas anónimas o estudios persistentes con TTL almacena información del visitante sin necesidad. Calcular todo en React duplica reglas y dificulta usar el catálogo solar y validar resultados. Se eligen cálculos temporales en backend.

## Consecuencias

Recargar pierde el borrador. La recomendación anónima debe terminar dentro del deadline HTTP o devolver error; no tiene recuperación por id ni garantía durable. Se añaden límites de abuso sin introducir servicios pagos. La capacidad M2M conserva su clave y sus rutas. Consultar catálogos puede usar sus índices/cachés compartidos; nunca se cachea el consumo/resultados personales invitados.

## Validación prevista

Navegador limpio sin credenciales obtiene consumo y paneles; no cambian tablas/documentos de negocio ni aparecen eventos. Guardados/admin anónimos reciben 401. Un login por sí solo no persiste. Tras Guardar explícito existe información solo bajo el usuario. Ver [especificación del modo](../GUEST-MODE.md).

## Actualización 2026-11-08 - contexto y por qué

### Contexto

El visitante necesita entender consumo y paneles antes de crear una cuenta; el borrador temporal no necesita identidad ni una fila de base.

### Decisión, motivo y consecuencias

Conservar casos de uso sin escrituras/eventos y guardar solo por una acción autenticada futura. Es mejor para este flujo porque evita persistir datos que el usuario no pidió guardar y mantiene usable la calculadora mientras Usuarios sigue pendiente. El costo aceptado es perder el borrador al recargar.

### Referencias de clase y alcance

Clases 1 y 2 sobre límites y contratos; alcance explícito del equipo. No implica que todo sistema deba ser anónimo.

El texto anterior se conserva como historia. Toda revisión posterior se añadirá en este mismo archivo.


## Actualización 2026-11-09

Contexto: se integra la aplicación completa y observabilidad sin convertir una simulación en un guardado implícito. Se conservan los motivos originales: invitado sin escrituras y salud autenticada. Identidad solo habilita persistencia explícita y datos propios; Prometheus usa la misma credencial técnica separada para /metrics. Estado y evidencia vigentes en [verificación](../VERIFICACION-FINAL.md). Este cambio se añade al mismo ADR, sin archivo sustituto.


## Actualización 2026-11-09 — lenguaje de cliente y trazabilidad opcional

**Contexto:** el cliente no necesita UUID ni versiones en avisos, selectores o títulos de guardados. Sí pidió conservarlos en fichas e historial.

**Decisión y por qué:** se ocultan referencias técnicas fuera de esos detalles. Confirmar evaluación se presenta como Calcular paneles; la comparación adicional queda bajo Comparar con otras condiciones solares en el historial. Archivar sigue siendo opcional y reversible; Eliminar es otra acción. Se conserva la confirmación transaccional necesaria para generar el análisis asíncrono, con su historial inmutable. La interfaz recarga solo las listas modificadas y respeta Retry-After con reintentos acotados ante 429; no se desactiva la cuota para hacer pasar las pruebas.

**Verificación:** doce casos de navegador reales en escritorio/móvil aprobaron los flujos; el primer intento detectó exceso de recargas y se conserva identificado. La calculadora invitada sigue sin guardar datos y el contrato externo de Clínica permanece intacto.
