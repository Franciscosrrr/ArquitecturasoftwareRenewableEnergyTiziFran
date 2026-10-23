# Consumo - Entrega 2

Go/Gin con application independiente del transporte. POST /api/v1/estimaciones-consumo conserva contrato M2M 1.0.1; POST /api/v1/publico/consumo admite simulación anónima con parámetros completos; POST /api/v1/publico/consumo-catalogo resuelve fichas vigentes por HTTP interno y aplica el mismo cálculo decimal.

No escribe consumos, evaluaciones ni eventos. No recibe credenciales MySQL. Timeouts y circuit breaker protegen la llamada al catálogo. HEALTHCHECK_TOKEN, INTERNAL_TOKEN, M2M_API_KEY y M2M_DENIED_KEY se suministran por entorno desde el .env local ignorado. Dockerfile se construye desde raíz usando infra/backend.Dockerfile y ARG SERVICE=services/consumo.
