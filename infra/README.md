# Infraestructura de entrega 1

El Compose de la raíz inicia por defecto solo el mock. El perfil estructura agrega procesos Go iniciales; verificar ejecuta pruebas. Los Dockerfiles están junto a cada componente. No se lanzan bases/broker vacíos aparentando un flujo integrado. La infraestructura completa y sus dependencias de negocio siguen descritas en ARCHITECTURE.

Usar scripts/compose.ps1 en PowerShell o scripts/compose.sh en Linux/macOS para generar HEALTHCHECK_TOKEN automáticamente antes de invocar Compose. Todos los componentes HTTP requieren esta credencial para salud; no se configura ningún valor fijo en el repositorio.
