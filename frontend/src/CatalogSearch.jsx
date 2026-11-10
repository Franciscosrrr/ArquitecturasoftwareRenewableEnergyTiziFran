import React,{useState} from 'react';
/**
 * Explora paneles solares con filtros y paginación del servidor.
 *
 * @method CatalogSearch
 * @returns {React.ReactElement} Catálogo solar con referencias técnicas dentro de cada ficha.
 */
export default function CatalogSearch(){
 const [page,setPage]=useState(null),[error,setError]=useState(''),[busy,setBusy]=useState(false),[filters,setFilters]=useState({q:'',moneda:'ARS',orden:'nombre',potenciaMin:'',potenciaMax:'',superficieMax:'',precioMax:''});
 /**
  * Recupera una página con criterios validados por Solar.
  *
  * @method search
  * @param {number} number - Página solicitada.
  * @returns {Promise<void>} Presenta resultados o un error recuperable.
  */
 async function search(number=1){setBusy(true);setError('');try{const params=new URLSearchParams({pagina:number,tamano:6});for(const [key,value] of Object.entries(filters)){if(value!=='')params.set(key,value)}const response=await fetch('/api/v1/publico/paneles/busqueda?'+params);const body=await response.json();if(!response.ok)throw Error(body.mensaje||'No pudimos buscar los paneles.');setPage(body)}catch(error){setError(error.message)}finally{setBusy(false)}}
 /**
  * Cambia un filtro e invalida resultados pertenecientes a otra consulta.
  *
  * @method change
  * @param {string} key - Campo del filtro.
  * @param {string} value - Valor ingresado.
  * @returns {void} Actualiza criterios y limpia la página visible.
  */
 function change(key,value){setFilters(current=>({...current,[key]:value}));setPage(null)}
 return <section className="workspace panel-catalog" aria-label="Catálogo de paneles solares"><h2>Encontrá tus paneles solares</h2><p>Explorá modelos por potencia, espacio y precio. Los valores son referencias de ejemplo.</p><details className="search"><summary>Buscar y filtrar paneles solares</summary><form onSubmit={event=>{event.preventDefault();search()}}><div className="inputs"><label>Modelo o descripción<input maxLength={100} value={filters.q} onChange={e=>change('q',e.target.value)}/></label><label>Moneda<select value={filters.moneda} onChange={e=>change('moneda',e.target.value)}><option value="">Todas</option><option value="ARS">Pesos argentinos</option><option value="USD">Dólares</option></select></label><label>Potencia mínima (W)<input type="number" min="0" max="2000" step="1" value={filters.potenciaMin} onChange={e=>change('potenciaMin',e.target.value)}/></label><label>Potencia máxima (W)<input type="number" min="0" max="2000" step="1" value={filters.potenciaMax} onChange={e=>change('potenciaMax',e.target.value)}/></label><label>Superficie máxima por panel (m²)<input type="number" min="0" max="100" step="0.001" value={filters.superficieMax} onChange={e=>change('superficieMax',e.target.value)}/></label><label>Precio máximo por panel<input type="number" min="0" max="10000000" step="0.01" value={filters.precioMax} onChange={e=>change('precioMax',e.target.value)}/></label><label>Ordenar por<select value={filters.orden} onChange={e=>change('orden',e.target.value)}><option value="nombre">Nombre</option><option value="potencia">Menor potencia</option><option value="-potencia">Mayor potencia</option><option value="precio">Menor precio</option></select></label></div><button disabled={busy}>{busy?'Buscando…':'Buscar paneles'}</button></form>{error&&<p className="error" role="alert">{error}</p>}{page&&<><p role="status">{page.total} {page.total===1?'modelo encontrado':'modelos encontrados'} · página {page.pagina}</p>{page.total===0&&<p>No encontramos paneles con esos filtros. Probá ampliar el rango.</p>}<div className="panel-grid">{page.items.map(panel=><article className="saved panel-card" key={panel.id}><h3>{panel.nombre}</h3><p>{panel.potenciaW} W · {panel.superficieEfectivaMilliM2/1000} m² por panel</p><strong>{panel.precioCentavos>0?new Intl.NumberFormat('es-AR',{style:'currency',currency:panel.moneda}).format(panel.precioCentavos/100):'Precio a consultar'}</strong><details><summary>Ver ficha del panel</summary><p>Identificador: {panel.id}</p><p>Versión de ficha: {panel.version}</p><p>{panel.fuente}</p></details></article>)}</div><div className="actions"><button disabled={busy||page.pagina===1} onClick={()=>search(page.pagina-1)}>Anterior</button><button disabled={busy||page.pagina*page.tamano>=page.total} onClick={()=>search(page.pagina+1)}>Siguiente</button></div></>}</details></section>
}
