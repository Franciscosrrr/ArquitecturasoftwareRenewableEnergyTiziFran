# ADR complementario: precisión del diseño para entrega 1

Estado: adoptado para diseño. Complementa D1/D3/D5/D8; no sustituye una decisión anterior implementada.

## Contexto

La propuesta original identificaba estudios por evaluación y configuración de consumo, impidiendo variar cobertura. También asignaba a la llamada externa los mismos 8 s que a toda la recomendación temporal y dejaba genérica la comprobación de disponibilidad.

## Decisión

Identificar estudio por evaluación y versión de configuración solar; deduplicar solicitudes posteriores por propietario/operación/clave/hash en el documento del estudio. Reservar 6 s al proveedor dentro de 8 s de Solar temporal. Separar readiness por capacidades. Confirmar mensajes tras persistencia reconocida con journal. Reintentar invalidación de caché mediante outbox.

## Alternativas y consecuencias

Un único estudio por evaluación simplifica el índice, pero incumple cambiar cobertura manteniendo historial. Dar todo el presupuesto al proveedor deja sin tiempo al catálogo/respuesta. Readiness única facilita el balanceador pero deshabilita rutas que podrían funcionar. El diseño elegido exige más pruebas e índices explícitos; no promete alta disponibilidad de una base única.

## Validación pendiente

Pruebas concurrentes de dos coberturas y reentrega, pérdida de respuesta, expiración de lease, cancelación temporal, caída de broker sin bloquear invitado e invalidación después de una falla. El mock no prueba transacciones ni mensajería. Detalle normativo en ARCHITECTURE §13.

## Actualización 2026-11-08 - contexto y por qué

### Contexto

El diseño temprano mezclaba presupuesto del proveedor, evaluación/estudio y disponibilidad por capacidades; al crecer el flujo se necesitó precisar sus límites.

### Decisión, motivo y consecuencias

Conservar este antecedente sin convertirlo en otra decisión D1-D13. Sus presupuestos de 6/8 s corresponden al borrador histórico, no al vendedor actual: prevalecen D5/D10 canónicos con 1 s comercial y 4 s totales. Mantener historia evita atribuir al código un presupuesto inexistente.

### Referencias de clase y alcance

Clase 7.pdf pp. 12-13 y Clase 8.pdf pp. 1-5; requisitos de idempotencia y resiliencia del enunciado.

El texto anterior se conserva como historia. Toda revisión posterior se añadirá en este mismo archivo.
