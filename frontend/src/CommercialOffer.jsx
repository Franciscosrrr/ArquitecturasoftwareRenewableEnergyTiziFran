import React from 'react';

/**
 * Presenta la procedencia y el importe comercial sin confundir simulación con disponibilidad real.
 *
 * @method CommercialOffer
 * @param {Object} props - Datos de presentación de la oferta.
 * @param {Object} props.quote - Oferta adaptada por el backend, sin credenciales.
 * @returns {React.ReactElement|null} Oferta y advertencias, o vacío sin cotización.
 */
export default function CommercialOffer({quote}) {
  if (!quote) return null;
  const simulated = quote.origen !== 'externo';
  const currency = new Intl.NumberFormat('es-AR', {style: 'currency', currency: quote.moneda});
  return <div className="commercial-offer" aria-label="Oferta de paneles">
    {simulated ? <p role="status"><strong>Oferta simulada: stock y precios didácticos.</strong> {quote.motivoSimulacion && 'El proveedor real no está disponible. Usamos paneles simulados como respaldo.'}</p> : <p><strong>Oferta del proveedor: {quote.proveedorNombre}.</strong></p>}
    <p>{quote.cantidad} {quote.cantidad === 1 ? 'panel' : 'paneles'} de {quote.potenciaPanelW} W · Stock informado: {quote.stockDisponible}</p>
    <p><strong>Total de la oferta: {currency.format(quote.totalCentavos / 100)}</strong></p>
    {!simulated && <><p>Incluye flete: {currency.format((quote.fleteCentavos || 0) / 100)}. {quote.descuentoAplicado ? `Descuento aplicado: ${quote.descuentoPorcentaje}%.` : 'Sin descuento aplicado.'}</p><p>Precio y stock al momento de la consulta; la oferta no reserva inventario ni congela el precio.</p></>}
    <p>No incluye instalación, inversor ni baterías. Potencia y superficie se toman de nuestras fichas locales de referencia.</p>
    <details><summary>Ver ficha de la oferta</summary>{quote.cotizacionId && <p>Identificador de simulación: {quote.cotizacionId}</p>}{quote.skuProveedor && <p>SKU comercial: {quote.skuProveedor}</p>}{quote.referenciaProveedor && <p>Referencia de consulta al proveedor: {quote.referenciaProveedor}</p>}{quote.consultadoEn && <p>Consultada: {quote.consultadoEn}</p>}</details>
  </div>;
}
