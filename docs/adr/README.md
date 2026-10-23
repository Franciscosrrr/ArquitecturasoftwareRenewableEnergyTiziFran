# Decisiones arquitectónicas - Entrega 2

| Decisión | Registro vigente | Estado del hito |
|---|---|---|
| D1 Límites | [ADR-001](ADR-001.md) | Validados: solo Electrodomésticos accede a su MySQL; Consumo usa HTTP |
| D2 Arquitectura interna | [ADR-002-E2](ADR-002-E2.md) | Capas operativas; hexagonal Solar pendiente |
| D3 Persistencia | [ADR-003-E2](ADR-003-E2.md) | MySQL integrado, SQL explícito; demás BD pendientes |
| D4 Consistencia | Alcance inicial en D3/D5 | ADR específico de evaluación durable pendiente para la final |
| D5 Comunicación | [ADR-005-E2](ADR-005-E2.md) | HTTP por lote, timeout, circuito; mensajería pendiente |
| D6 Búsqueda | [ADR-006](ADR-006.md) | SQL operativo; Meilisearch futuro |
| D7 Caché | [ADR-007](ADR-007.md) | Diseño Valkey, todavía sin implementación |
| D8 Capacidad propia | [ADR-008](ADR-008.md) | Contrato M2M 1.0.1 conservado y ejecutado por Go |
| D9 Proveedor | [ADR-009](ADR-009.md) | Asignado, contrato/documentación no recibidos |
| D10 Resiliencia | [ADR-010](ADR-010.md) | Timeout/circuito y ensayo local |
| D11 Observabilidad | [ADR-011](ADR-011.md) | Primera versión de logs/trazas; métricas futuras |
| D12 Balanceo | [ADR-012](ADR-012.md) | Diseño justificado; réplicas futuras |
| D13 Capacidad/costos | [ADR-013](ADR-013.md) | Ensayo local y costo contratado cero |

Los ADR anteriores se conservan. ADR-002, ADR-003 y ADR-005 quedan reemplazados para el alcance indicado por sus versiones E2. [ADR-GUEST](ADR-GUEST.md), [ADR-HEALTH](ADR-HEALTH.md) y [ADR-REVISION-1](ADR-REVISION-1.md) son antecedentes complementarios; sus presupuestos propuestos se leen junto con la implementación E2 vigente. La autenticación de salud continúa vigente.
