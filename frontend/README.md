# Aplicación React

Estado 2026-11-09. Se compila con Vite y el gateway sirve sus archivos. El invitado selecciona equipos y hábitos, calcula consumo, compara alternativas y cotiza sin guardar datos de negocio. La búsqueda visible es de paneles solares, con texto, potencia, superficie, moneda/precio, orden y páginas. El catálogo de equipos continúa en el selector del consumo.

Las cuentas permiten lugares, consumos editables, duplicado, archivo/restauración sin nueva versión, eliminación lógica e historial. Calcular paneles confirma una evaluación durable y solicita el estudio asíncrono. Las alternativas avanzadas están plegadas en el historial; referencias y versiones se muestran en fichas/historial. La administración permite fichas, roles, índices y DLQ según permisos backend.

El borrador y el access token viven en memoria; no se usan localStorage, sessionStorage ni IndexedDB. La sesión se recupera con cookie refresh HttpOnly/SameSite Strict. La entrada HTTPS web agrega Secure; localhost conserva su funcionamiento. No se entrega una API key M2M al navegador. Las URLs de API son relativas al mismo origen.

El flujo reproducible usa Docker y no requiere Node en el host. Para desarrollo opcional: npm ci y npm run dev; Vite reenvía /api al gateway local. [Suite actual de doce casos](../docs/BONUS-E2E.md), [cambios de cliente](../docs/EXPERIENCIA-CLIENTE.md) y [publicación web](../docs/PUBLICACION-WEB.md).
