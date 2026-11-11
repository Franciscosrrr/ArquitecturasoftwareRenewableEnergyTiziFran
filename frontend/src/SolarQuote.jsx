import React, {useEffect, useRef, useState} from 'react';
import CommercialOffer from './CommercialOffer';

/**
 * Cotiza paneles para el escenario calculado y conserva el consumo ante errores.
 *
 * @method SolarQuote
 * @param {Object} props - Propiedades de la sección solar.
 * @param {Object} props.consumo - Solicitud por catálogo que Solar hará validar.
 * @returns {React.ReactElement} Formulario y oferta comercial del vendedor.
 */
export default function SolarQuote({consumo}) {
  const [sun, setSun] = useState(4);
  const [yieldFactor, setYieldFactor] = useState(0.8);
  const [quote, setQuote] = useState(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const pending = useRef(null);
  useEffect(() => () => pending.current?.abort(), []);

  /**
   * Solicita una cotización con cancelación y presupuesto de cinco segundos.
   *
   * @method requestQuote
   * @param {React.FormEvent} event - Envío del formulario solar.
   * @returns {Promise<void>} Actualiza oferta o mensaje de error.
   */
  async function requestQuote(event) {
    event.preventDefault();
    pending.current?.abort();
    const controller = new AbortController();
    pending.current = controller;
    const timer = setTimeout(() => controller.abort(), 5000);
    setLoading(true);
    setError('');
    setQuote(null);
    try {
      const response = await fetch('/api/v1/publico/solar/cotizaciones', {
        method: 'POST', headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({consumo, horasSolPico: Number(sun), factorRendimiento: Number(yieldFactor)}),
        signal: controller.signal,
      });
      const body = await response.json();
      if (!response.ok) throw new Error(body.mensaje || 'No pudimos obtener la cotización.');
      setQuote(body);
    } catch (failure) {
      setError(failure.name === 'AbortError' ? 'La cotización tardó demasiado. Tu consumo sigue disponible; volvé a intentar.' : failure.message);
    } finally {
      clearTimeout(timer);
      setLoading(false);
    }
  }

  return <section className="solar" aria-label="Cotización solar">
    <h2>Planificá tus paneles</h2>
    <p>Usamos tu consumo y los supuestos indicados para consultar una oferta con stock. La cotización no reserva ni compra paneles.</p>
    <form onSubmit={requestQuote}>
      <div className="inputs">
        <label>Horas solares pico<input type="number" min="0.5" max="8" step="0.1" required value={sun} disabled={loading} onChange={e => {setSun(e.target.value); setQuote(null);}} /></label>
        <label>Factor de rendimiento<input type="number" min="0.1" max="1" step="0.01" required value={yieldFactor} disabled={loading} onChange={e => {setYieldFactor(e.target.value); setQuote(null);}} /></label>
      </div>
      <button className="primary" disabled={loading}>{loading ? 'Consultando al vendedor…' : 'Cotizar paneles'}</button>
    </form>
    {error && <p className="error" role="alert">{error}</p>}
    {quote && <div aria-live="polite">
      <CommercialOffer quote={quote.cotizacion}/>
      <p>Potencia objetivo: {quote.potenciaObjetivoW} W. No incluye instalación, inversor ni baterías; no reemplaza un estudio técnico.</p>
      <details><summary>Ver referencia del cálculo</summary><p className="trace">Referencia de consulta: {quote.traceId}</p></details>
    </div>}
  </section>;
}
