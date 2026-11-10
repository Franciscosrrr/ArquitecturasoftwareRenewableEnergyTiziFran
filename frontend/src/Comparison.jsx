import React,{useState} from 'react';
/**
 * Compara alternativas solares temporales con superficie y fuente explícitas.
 *
 * @method Comparison
 * @param {Object} props - Promedio diario calculado por Consumo.
 * @returns {React.ReactElement} Formulario y alternativas de dimensionamiento sin guardado.
 */
export default function Comparison({daily}){
 const [result,setResult]=useState(null),[error,setError]=useState(''),[busy,setBusy]=useState(false);
 /**
  * Solicita un dimensionamiento anónimo al catálogo activo de Solar.
  *
  * @method compare
  * @param {React.FormEvent} event - Supuestos de la comparación.
  * @returns {Promise<void>} Muestra cumplimiento, cobertura y fuentes.
  */
 async function compare(event){event.preventDefault();setBusy(true);setError('');const form=new FormData(event.currentTarget);try{const response=await fetch('/api/v1/publico/recomendaciones-solares',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({promedioDiarioKWh:Number(daily),coberturaObjetivo:Number(form.get('cobertura')),rendimientoGlobal:Number(form.get('rendimiento')),superficieUtilM2:form.get('superficie')?Number(form.get('superficie')):undefined,recursoSolar:{tipo:'MANUAL',hsp:Number(form.get('hsp')),fuente:form.get('fuente')}})});const body=await response.json();if(!response.ok)throw Error(body.mensaje);setResult(body)}catch(error){setError(error.message)}finally{setBusy(false)}}
 return <div className="solar"><h3>Comparar paneles y superficie</h3><form onSubmit={compare}><div className="inputs"><label>Horas solares pico manuales<input name="hsp" type="number" min="0.5" max="8" step="0.1" defaultValue={4} required/></label><label>Fuente del supuesto<input name="fuente" maxLength={200} defaultValue="Supuesto manual didáctico del usuario" required/></label><label>Rendimiento global<input name="rendimiento" type="number" min="0.1" max="1" step="0.01" defaultValue={.8} required/></label><label>Cobertura objetivo<input name="cobertura" type="number" min="0.01" max="1" step="0.01" defaultValue={1} required/></label><label>Superficie útil m² (opcional)<input name="superficie" type="number" min="0.000001" max="1000000" step="0.000001"/></label></div><button disabled={busy}>{busy?'Comparando…':'Comparar alternativas'}</button></form>{error&&<p role="alert" className="error">{error}</p>}{result&&<><p>{result.estado} · {result.criterioSeleccion}</p><p>{result.fuenteRecurso}</p><div className="table-scroll"><table><thead><tr><th>Modelo</th><th>Paneles</th><th>kWh/día</th><th>Cobertura</th><th>m²</th><th>Cumple</th></tr></thead><tbody>{result.alternativas.map(a=><tr key={a.panelId}><td>{a.modelo}</td><td>{a.cantidad}</td><td>{a.generacionDiariaKWh}</td><td>{(Number(a.coberturaAlcanzada)*100).toFixed(2)}%</td><td>{a.superficieRequeridaM2}</td><td>{a.cumpleObjetivo?'Sí':'No'}</td></tr>)}</tbody></table></div>{result.supuestos.map(s=><p key={s}>{s}</p>)}<details><summary>Ver ficha del análisis</summary>{result.alternativas.map(a=><p key={a.panelId}>{a.modelo}: ficha {a.panelId}, versión {a.versionFicha}</p>)}<p className="trace">Referencia de consulta: {result.traceId}</p></details></>}</div>
}
