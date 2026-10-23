# Estructura operativa de entrega 2

`gateway`: HTTP público, proxy, interfaz compilada y contratos. `services/consumo`: application con aritmética/validación, HTTP e integration/catalog. `services/electrodomesticos`: HTTP, repository/mysql y schema.sql embebido. `platform`: servidor, salud, logs, trazas, cliente y limitador, sin modelos de dominio. `frontend`: React/Vite. `infra/backend.Dockerfile`: compilación y runner de tests Go. `tests`: HTTP real y Playwright e2e. `scripts`: arranque, verificación, demo y evidencias. `docs/evidence/entrega2`: resultados reales seleccionados.

Los módulos fijan dependencias en go.mod/go.sum; los módulos nuevos dependen del módulo técnico platform mediante replace local, sin importar los modelos privados de otros servicios. Docker construye desde la raíz del monorepo. Usuarios y Solar siguen como esqueletos compilables heredados, fuera de los servicios predeterminados de Compose.

Dependencias principales: Go 1.26.1, Gin 1.11.0, go-sql-driver/mysql 1.9.3, OpenTelemetry 1.38.0 y otelhttp 0.63.0, React 19.2.0, Vite 7.3.7, Playwright 1.64.1. No se necesita instalarlas en el host para usar Compose.
