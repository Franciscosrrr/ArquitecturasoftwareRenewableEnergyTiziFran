# Infraestructura - Entrega 2

compose.yaml predeterminado inicia MySQL, Jaeger, Electrodomésticos, Consumo y gateway/React. Los servicios de prueba están en el perfil verificar; el mock histórico en perfil mock, puerto 8081. El proyecto Compose se llama renewable-energy-entrega2.

Arranque y secretos: scripts/compose.ps1 o compose.sh. Volumen catalogo-data conserva las fichas. Solo 8080 y 16686 se publican en 127.0.0.1. No ejecutar down -v para detener normalmente: destruiría los datos locales. Las pruebas controladas operan únicamente este proyecto Compose.
