import React,{useEffect,useState} from 'react';
import {createRoot} from 'react-dom/client';
import './style.css';
import SolarQuote from './SolarQuote.jsx';
import Account from './Account.jsx';
import Comparison from './Comparison.jsx';
import CatalogSearch from './CatalogSearch.jsx';

/**
 * Formatea energía con hasta seis decimales para la pantalla.
 *
 * @method format
 * @param {number} n - Valor de energía.
 * @returns {string} Texto numérico localizado.
 */
const format=n=>new Intl.NumberFormat('es-AR',{maximumFractionDigits:6}).format(n);
/**
 * Administra el escenario temporal de equipos y su estimación.
 *
 * @method App
 * @returns {React.ReactElement} Calculadora y flujo de cotización solar.
 */
function App(){
 const[catalog,setCatalog]=useState([]),[selected,setSelected]=useState([]),[days,setDays]=useState(30),[result,setResult]=useState(null),[error,setError]=useState(''),[loading,setLoading]=useState(false),[loadingCatalog,setLoadingCatalog]=useState(true);
 /**
 * Carga las fichas activas y permite reintentar la lectura.
 *
 * @method load
 * @returns {Promise<void>} Actualiza catálogo y estado de carga.
 */
async function load(){setLoadingCatalog(true);setResult(null);setError('');try{const r=await fetch('/api/v1/publico/electrodomesticos?tamano=100');if(!r.ok)throw Error('No pudimos cargar el catálogo. Podés volver a intentar.');setCatalog((await r.json()).items)}catch(e){setError(e.message)}finally{setLoadingCatalog(false)}}
 useEffect(()=>{load()},[]);
 /**
 * Agrega una ficha al escenario sin duplicar equipos.
 *
 * @method add
 * @param {string} id - Identificador de catálogo.
 * @returns {void} Actualiza el borrador e invalida el resultado.
 */
function add(id){const f=catalog.find(x=>x.id===id);if(!f||selected.some(x=>x.catalogoId===id))return;setSelected([...selected,{catalogoId:id,cantidad:1,...(f.modo==='POTENCIA'?{horasPorDia:5,diasUso:Number(days),factorFuncionamiento:Number(f.factorFuncionamiento)}:{ciclosPeriodo:12})}]);setResult(null)}
 /**
 * Actualiza un hábito y elimina la estimación anterior.
 *
 * @method update
 * @param {string} id - Ficha seleccionada.
 * @param {string} key - Hábito a editar.
 * @param {string} value - Valor ingresado.
 * @returns {void} Actualiza los datos temporales.
 */
function update(id,key,value){setSelected(selected.map(x=>x.catalogoId===id?{...x,[key]:value===''?'':Number(value)}:x));setResult(null)}
 /**
 * Solicita a Consumo una estimación con potencias del catálogo.
 *
 * @method calculate
 * @param {React.FormEvent} e - Envío del formulario.
 * @returns {Promise<void>} Muestra desglose o error de validación.
 */
async function calculate(e){e.preventDefault();setLoading(true);setError('');setResult(null);try{const r=await fetch('/api/v1/publico/consumo-catalogo',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({diasPeriodo:Number(days),equipos:selected})});const body=await r.json();if(!r.ok)throw Error(body.mensaje+(body.detalles?.[0]?' '+body.detalles[0].campo+': '+body.detalles[0].motivo:''));setResult(body)}catch(e){setError(e.message)}finally{setLoading(false)}}
 return <><header><a href="/" className="brand"><span className="brandmark">↗</span> Renewable Energy</a><span className="tag">Calculadora de consumo</span></header><main><section className="intro"><p className="eyebrow">ENTENDÉ TU ENERGÍA</p><h1>Tu consumo,<br/><em>más claro.</em></h1><p>Elegí tus equipos y contanos cómo los usás.<br/>Calculamos una estimación para tu espacio.</p></section><aside className="note"><strong>Probá sin registrarte.</strong><p>Podés calcular sin cuenta. Si querés guardar tus consumos, iniciá sesión más abajo. Los valores de los equipos son referencias: verificá su ficha técnica.</p></aside><form onSubmit={calculate} className="workspace"><fieldset disabled={loading}><div className="section-title"><div><span className="step">01</span><h2>Armá tu escenario</h2></div><label>Período en días<input aria-label="Período en días" type="number" min="1" max="366" step="1" required value={days} onChange={e=>{setDays(e.target.value);setResult(null)}}/></label></div><div className="picker"><label htmlFor="equipo">Agregar un equipo</label><select id="equipo" value="" onChange={e=>add(e.target.value)} disabled={loadingCatalog}><option value="">{loadingCatalog?'Cargando catálogo…':'Seleccioná del catálogo'}</option>{catalog.filter(f=>!selected.some(x=>x.catalogoId===f.id)).map(f=><option key={f.id} value={f.id}>{f.nombre}</option>)}</select><button type="button" className="text-button" onClick={load} disabled={loadingCatalog}>Actualizar catálogo</button></div>{selected.length===0&&<div className="empty">Agregá tu primer equipo para empezar.</div>}<div className="equipment">{selected.map(item=>{const f=catalog.find(x=>x.id===item.catalogoId);return <article key={item.catalogoId}><div className="equipment-heading"><div><small>{f?.categoria}</small><h3>{f?.nombre||'Equipo del consumo guardado'}</h3><p>{f?.modo==='POTENCIA'?`${format(f.potenciaW)} W`:`${format(f?.energiaPorCicloKWh)} kWh por ciclo`}</p></div><button type="button" className="remove" aria-label={`Quitar ${f?.nombre}`} onClick={()=>{setSelected(selected.filter(x=>x.catalogoId!==item.catalogoId));setResult(null)}}>Quitar</button></div><div className="inputs"><Field label="Cantidad" value={item.cantidad} min={1} max={1000} step={1} onChange={v=>update(item.catalogoId,'cantidad',v)}/>{f?.modo==='POTENCIA'?<><Field label="Horas por día" value={item.horasPorDia} min={0} max={24} onChange={v=>update(item.catalogoId,'horasPorDia',v)}/><Field label="Días de uso" value={item.diasUso} min={0} max={Number(days)} step={1} onChange={v=>update(item.catalogoId,'diasUso',v)}/><Field label="Factor de funcionamiento" value={item.factorFuncionamiento} min={0.000001} max={1} onChange={v=>update(item.catalogoId,'factorFuncionamiento',v)}/></>:<Field label="Ciclos por unidad en el período" value={item.ciclosPeriodo} min={0} max={10000} step={1} onChange={v=>update(item.catalogoId,'ciclosPeriodo',v)}/>}</div>{f?.modo==='POTENCIA'&&<small>Factor 1 si la potencia ya representa un promedio. Se aplica una sola vez.</small>}</article>})}</div><div aria-live="polite">{error&&<p className="error" role="alert">{error}</p>}</div><button className="primary" disabled={loading||selected.length===0||loadingCatalog}>{loading?'Calculando…':'Calcular consumo'} <span>↗</span></button></fieldset></form>{result&&<section className="results" aria-live="polite"><p className="eyebrow">TU ESTIMACIÓN</p><div className="totals"><div><strong>{format(result.totalPeriodoKWh)}</strong><span>kWh en {result.diasPeriodo} días</span></div><div><strong>{format(result.promedioDiarioKWh)}</strong><span>kWh por día, en promedio</span></div></div><table><caption>Desglose por equipo</caption><thead><tr><th>Equipo</th><th>Consumo del período</th></tr></thead><tbody>{result.desglose.map(x=><tr key={x.id}><td>{catalog.find(f=>f.id===x.id)?.nombre||'Equipo del consumo'}</td><td>{format(x.energiaPeriodoKWh)} kWh</td></tr>)}</tbody></table><p>Estimación basada en tus hábitos; no reemplaza una medición real.</p><details><summary>Ver ficha del cálculo</summary><p>Algoritmo: {result.versionAlgoritmo}</p><p className="trace">Traza: {result.traceId}</p></details><Comparison daily={result.promedioDiarioKWh}/><SolarQuote consumo={{diasPeriodo:Number(days),equipos:selected}} /></section>}<CatalogSearch/><Account draft={{diasPeriodo:Number(days),equipos:selected}} onRestore={value=>{setDays(value.diasPeriodo);setSelected(value.equipos);setResult(null)}}/><footer>Renewable Energy · Planificá tu energía<br/>Las ofertas indican si provienen del proveedor o del respaldo simulado. Las HSP son supuestos manuales explícitos.</footer></main></>;
}
/**
 * Presenta un campo numérico con límites del dominio.
 *
 * @method Field
 * @param {Object} props - Etiqueta, valor, límites y callback de edición.
 * @returns {React.ReactElement} Control numérico accesible.
 */
function Field({label,value,onChange,min,max,step='0.000001'}){return <label>{label}<input type="number" required min={min} max={max} step={step} value={value} onChange={e=>onChange(e.target.value)}/></label>}
createRoot(document.getElementById('root')).render(<App/>);
