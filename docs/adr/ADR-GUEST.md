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
