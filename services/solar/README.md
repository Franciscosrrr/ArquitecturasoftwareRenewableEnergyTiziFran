# Servicio Solar

Estado 2026-11-09. Dominio/aplicación/puertos independientes de drivers y adaptadores HTTP, Mongo, proveedor y mensajería. Dos instancias ofrecen búsqueda de paneles, recomendaciones/cotizaciones temporales y estudios autenticados. El gateway las balancea; Solar consulta directamente a Consumo y al vendedor, sin usar el gateway como bus interno.

Mongo conserva paneles, estudios, deduplicación y claims con lease. El worker recibe ConsumoEvaluado.v1, congela entradas/fichas, calcula alternativas y consulta al vendedor por SKU; persiste antes del ACK. Reintentos durables, deduplicación y DLQ cubren fallas e incompatibilidades. No se compra ni reserva stock. HSP y referencias solares siguen siendo supuestos didácticos explícitos.

El índice paneles en Meilisearch sirve texto, potencia, superficie, moneda/precio, orden y páginas. Una marca Mongo pendiente y leases sincronizan versiones; la reconciliación periódica reconstruye el índice y el respaldo Mongo conserva búsqueda ante caída. La recomendación valida datos canónicos, no el índice derivado.

PANEL_PROVIDER_MODE distingue simulado/externo; el externo exige HTTPS y un adaptador compatible con el contrato real. El simulador comercial actual no acredita integración con el otro grupo. [Contrato comercial](../../docs/contracts/SOLAR.md), [búsqueda](../../docs/contracts/panel-search-v1/openapi.json), [pedido al vendedor](../../docs/SOLICITUD-PROVEEDOR.md), [D9](../../docs/adr/ADR-009.md).
