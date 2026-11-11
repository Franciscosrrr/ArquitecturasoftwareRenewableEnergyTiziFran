# Servicio Solar

Estado 2026-11-11. Dominio/aplicación/puertos independientes de drivers y adaptadores HTTP, Mongo, proveedor y mensajería. Dos instancias ofrecen búsqueda de paneles, recomendaciones/cotizaciones temporales y estudios autenticados. El gateway las balancea; Solar consulta directamente a Consumo y al vendedor, sin usar el gateway como bus interno.

Mongo conserva paneles, estudios, deduplicación y claims con lease. El worker recibe ConsumoEvaluado.v1, congela entradas/fichas, calcula alternativas y consulta al vendedor por SKU; persiste antes del ACK. Reintentos durables, deduplicación y DLQ cubren fallas e incompatibilidades. No se compra ni reserva stock. HSP y referencias solares siguen siendo supuestos didácticos explícitos.

El índice paneles en Meilisearch sirve texto, potencia, superficie, moneda/precio, orden y páginas. Una marca Mongo pendiente y leases sincronizan versiones; la reconciliación periódica reconstruye el índice y el respaldo Mongo conserva búsqueda ante caída. La recomendación valida datos canónicos, no el índice derivado.

PANEL_PROVIDER_MODE distingue simulado/externo. Comparador de Proveedores 1.1.0 está integrado mediante GET por SKU/cantidad y X-API-Key privada. El total ARS conserva flete y descuento. Las fallas técnicas activan el respaldo simulado para todo el estudio; stock vacío no activa ese respaldo. Su HTTP académico requiere una excepción explícita para la IP acordada. Precio/stock comerciales se consultan sin sobrescribir fichas técnicas locales. [Contrato comercial](../../docs/contracts/SOLAR.md), [búsqueda](../../docs/contracts/panel-search-v1/openapi.json), [pedido al vendedor](../../docs/SOLICITUD-PROVEEDOR.md), [D9](../../docs/adr/ADR-009.md).


[Contrato real y configuración](../../docs/contracts/comparador-proveedores-v1.1.0/README.md), [pruebas real/respaldo](../../docs/evidence/proveedor-real-2026-11-11/README.md). La búsqueda aplica precio/orden/paginación después de enriquecer las coincidencias técnicas. Los estudios guardan la procedencia aunque el proveedor se recupere después.
