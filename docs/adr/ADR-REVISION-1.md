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
