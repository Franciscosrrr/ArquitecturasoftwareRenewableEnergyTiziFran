import React,{useEffect,useRef,useState} from 'react';

/**
 * Conserva la sesión en memoria y permite guardar y evaluar escenarios propios.
 *
 * @method Account
 * @param {Object} props - Borrador temporal y función para restaurarlo.
 * @returns {React.ReactElement} Cuenta, lugares, configuraciones, historial y administración.
 */
export default function Account({draft,onRestore}){
 const [session,setSession]=useState(null),[notice,setNotice]=useState(''),[busy,setBusy]=useState(false),[places,setPlaces]=useState([]),[configs,setConfigs]=useState([]),[history,setHistory]=useState([]),[studies,setStudies]=useState([]),[placeId,setPlaceId]=useState(''),[name,setName]=useState('Mi consumo'),[editing,setEditing]=useState(null),[coverage,setCoverage]=useState(1),[sun,setSun]=useState(4),[yieldFactor,setYield]=useState(.8);
 const access=useRef(''), actionKeys=useRef({});
 const [showArchived,setShowArchived]=useState(false),[noticeError,setNoticeError]=useState(false);
 /**
  * Adopta una sesión sin almacenar tokens en el navegador.
  *
  * @method adopt
  * @param {Object|null} value - Sesión emitida por Usuarios.
  * @returns {void} Actualiza la identidad visible y la credencial en memoria.
  */
 function adopt(value){access.current=value?.accessToken||'';setSession(value)}
 /**
  * Ejecuta una solicitud autenticada y renueva una vez una sesión vencida.
  *
  * @method api
  * @param {string} path - Ruta relativa del API.
  * @param {string} method - Método HTTP.
  * @param {Object|undefined} data - Cuerpo de la operación.
  * @param {Object} headers - Cabeceras adicionales.
  * @param {boolean} retry - Permite un único refresh.
  * @param {number} trafficRetry - Reintentos acotados que respetan Retry-After.
  * @returns {Promise<Object|null>} Respuesta validada o error visible.
  */
 async function api(path,method='GET',data,headers={},retry=true,trafficRetry=0){
  const response=await fetch('/api/v1'+path,{method,credentials:'same-origin',headers:{...(data===undefined?{}:{'Content-Type':'application/json'}),...(access.current?{Authorization:'Bearer '+access.current}:{}),...headers},...(data===undefined?{}:{body:JSON.stringify(data)})});
  if(response.status===429&&trafficRetry<2){const seconds=Number(response.headers.get('Retry-After')||1);if(Number.isFinite(seconds)&&seconds>0&&seconds<=10){await response.text();await new Promise(resolve=>setTimeout(resolve,seconds*1000));return api(path,method,data,headers,retry,trafficRetry+1)}}
  if(response.status===401&&retry&&!path.startsWith('/auth/')){const renewed=await api('/auth/refresh','POST',undefined,{},false);adopt(renewed);return api(path,method,data,headers,false)}
  const body=response.status===204?null:await response.json();if(!response.ok)throw Error(body?.codigo==='CONFLICTO'?'Estos datos cambiaron. Actualizá tus datos y volvé a intentar.':body?.mensaje||'No se pudo completar la operación.');return body;
 }
 /**
  * Carga recursos propios y conserva la selección de lugar cuando sigue vigente.
  *
  * @method refresh
  * @param {string} scope - Listas modificadas por la acción; evita recargar toda la cuenta.
  * @returns {Promise<void>} Actualiza guardados e historial.
  */
 async function refresh(scope='all'){
  if(scope==='all'||scope==='places'){const p=await api('/lugares');setPlaces(p);setPlaceId(old=>p.some(x=>x.id===old)?old:old?'':p.find(x=>!x.archivado)?.id||'')}
  if(scope==='all'||scope==='configs')setConfigs(await api('/configuraciones'));
  if(scope==='all'||scope==='results'){setHistory(await api('/evaluaciones'));setStudies(await api('/estudios'))}
 }

 /**
  * Muestra errores y evita dobles envíos del mismo formulario.
  *
  * @method run
  * @param {Function} operation - Operación elegida por el usuario.
  * @returns {Promise<void>} Finaliza el estado de espera incluso ante fallas.
  */
 async function run(operation){if(busy)return;setBusy(true);setNotice('');setNoticeError(false);try{await operation()}catch(error){setNoticeError(true);setNotice(error.message)}finally{setBusy(false)}}
 useEffect(()=>{let active=true;api('/auth/refresh','POST',undefined,{},false).then(value=>{if(active){adopt(value);refresh().catch(()=>{})}}).catch(()=>{});return()=>{active=false}},[]);
 useEffect(()=>{if(!session)return;let active=true;const timer=setInterval(()=>{api('/estudios').then(value=>{if(active)setStudies(value)}).catch(()=>{})},5000);return()=>{active=false;clearInterval(timer)}},[session]);
 /**
  * Registra o autentica las credenciales ingresadas.
  *
  * @method authenticate
  * @param {React.FormEvent} event - Formulario con email, contraseña y acción.
  * @returns {Promise<void>} Inicia sesión y carga datos del propietario.
  */
 async function authenticate(event){event.preventDefault();const form=new FormData(event.currentTarget),mode=event.nativeEvent.submitter?.value;await run(async()=>{const input={email:form.get('email'),password:form.get('password')};if(mode==='registro')await api('/auth/registro','POST',input);adopt(await api('/auth/login','POST',input));await refresh()})}
 /**
  * Crea un lugar con superficie opcional declarada por el usuario.
  *
  * @method createPlace
  * @param {React.FormEvent} event - Datos del lugar.
  * @returns {Promise<void>} Persiste el lugar y recarga la lista.
  */
 async function createPlace(event){event.preventDefault();const form=new FormData(event.currentTarget);await run(async()=>{const p=await api('/lugares','POST',{nombre:form.get('nombre'),ubicacion:form.get('ubicacion'),superficieM2:form.get('superficie')?Number(form.get('superficie')):null});await refresh('places');setPlaceId(p.id)})}
 /**
  * Guarda el borrador mediante cálculo y validación del servidor.
  *
  * @method save
  * @returns {Promise<void>} Crea una configuración o incrementa su versión.
  */
 async function save(){await run(async()=>{if(!draft.equipos.length)throw Error('Agregá equipos al escenario antes de guardar.');const input={nombre:name,lugarId:placeId,consumo:draft,...(editing?{version:editing.version}:{} )};const saved=await api('/configuraciones'+(editing?'/'+editing.id:''),editing?'PUT':'POST',input);setEditing(saved);await refresh('configs');setNotice('Consumo guardado. Podés calcular los paneles que necesitás.')})}
 /**
  * Reutiliza una clave mientras la entrada no cambia, incluso ante respuesta perdida.
  *
  * @method keyFor
  * @param {string} identity - Operación y entrada canónica.
  * @returns {string} Clave de idempotencia del intento.
  */
 function keyFor(identity){return actionKeys.current[identity]||(actionKeys.current[identity]=Array.from(crypto.getRandomValues(new Uint8Array(16)),byte=>byte.toString(16).padStart(2,'0')).join(''))}
 /**
  * Confirma un snapshot inmutable con parámetros solares explícitos.
  *
  * @method evaluate
  * @param {Object} config - Configuración guardada seleccionada.
  * @returns {Promise<void>} Muestra publicación pendiente o estudio procesado.
  */
 async function evaluate(config){await run(async()=>{const place=places.find(x=>x.id===config.lugarId);const input={version:config.version,versionLugar:place.version,solar:{horasSolPico:Number(sun),factorRendimiento:Number(yieldFactor),coberturaObjetivo:Number(coverage)}};await api('/configuraciones/'+config.id+'/evaluaciones','POST',input,{'Idempotency-Key':keyFor(config.id+JSON.stringify(input))});await refresh('results');setNotice('Estamos calculando tus paneles. El resultado aparecerá en el historial.')})}
 /**
  * Solicita otro estudio sin modificar la evaluación ni los resultados previos.
  *
  * @method newStudy
  * @param {Object} evaluation - Evaluación propia del historial.
  * @returns {Promise<void>} Encola una variante de cobertura.
  */
 async function newStudy(evaluation){await run(async()=>{const input={horasSolPico:Number(sun),factorRendimiento:Number(yieldFactor),coberturaObjetivo:Number(coverage),superficieM2:evaluation.snapshot.lugar.superficieM2};await api('/evaluaciones/'+evaluation.id+'/estudios','POST',input,{'Idempotency-Key':keyFor(evaluation.id+JSON.stringify(input))});await refresh('results');setNotice('Nuevo estudio solicitado; se conservan los anteriores.')})}
 /**
  * Cambia el archivo como estado independiente, conservando la versión del consumo o lugar.
  *
  * @method archive
  * @param {string} kind - Recurso privado seleccionado.
  * @param {Object} item - Consumo o lugar propio con versión vigente.
  * @returns {Promise<void>} Actualiza la organización y recarga la lista.
  */
 async function archive(kind,item){await run(async()=>{const archived=kind==='lugares'?item.archivado:item.archivada;await api('/'+kind+'/'+item.id+'/archivo','PATCH',{version:item.version,archivado:!archived});await refresh(kind==='lugares'?'places':'configs');setNotice(archived?'Restaurado. Su contenido no cambió.':'Archivado. Podés restaurarlo cuando quieras.')})}
 /**
  * Elimina un recurso propio después de explicar qué se conserva en el historial.
  *
  * @method remove
  * @param {string} kind - Lugares o configuraciones.
  * @param {Object} item - Recurso propio a quitar.
  * @returns {Promise<void>} Elimina de las listas sin alterar evaluaciones anteriores.
  */
 async function remove(kind,item){if(kind==='lugares'&&configs.some(c=>c.lugarId===item.id)){setNoticeError(true);setNotice('Este lugar tiene consumos guardados. Movelos a otro lugar o eliminalos antes.');return}if(!window.confirm('¿Eliminar «'+item.nombre+'»? Se quitará de tus guardados. Los resultados del historial se conservarán.'))return;await run(async()=>{await api('/'+kind+'/'+item.id,'DELETE',{version:item.version});if(kind==='configuraciones'&&editing?.id===item.id){setEditing(null);setName('Mi consumo')}await refresh(kind==='lugares'?'places':'configs');setNotice('Eliminado de tus guardados. El historial se conserva.')})}
 return <section className="account workspace"><h2>Mis consumos</h2>{notice&&<p role="status" className={noticeError?"notice error":"notice"}>{notice}</p>}<p>Guardá tus consumos y organizalos por lugar. También podés pedir un análisis de paneles solares.</p>{!session?<form onSubmit={authenticate}><div className="inputs"><label>Email<input name="email" type="email" autoComplete="username" required/></label><label>Contraseña (mínimo 12 caracteres)<input name="password" type="password" minLength={12} maxLength={512} autoComplete="current-password" required/></label></div><div className="actions"><button disabled={busy} value="login">Iniciar sesión</button><button disabled={busy} value="registro">Crear cuenta</button></div></form>:<><div className="actions"><span>{session.usuario.email}{session.usuario.rol==='ADMINISTRADOR'?' · Administrador':''}</span><button disabled={busy} onClick={()=>run(async()=>{await api('/auth/logout','POST');adopt(null);setPlaces([]);setConfigs([]);setHistory([]);setStudies([]);setEditing(null)})}>Cerrar sesión</button><button disabled={busy} onClick={()=>run(refresh)}>Actualizar mis datos</button></div><details><summary>Crear lugar</summary><form onSubmit={createPlace}><div className="inputs"><label>Nombre del lugar<input name="nombre" required maxLength={100}/></label><label>Ubicación<input name="ubicacion" maxLength={200}/></label><label>Superficie útil (m², opcional)<input name="superficie" type="number" min="0.000001" max="1000000" step="0.000001"/></label></div><button disabled={busy}>Crear lugar</button></form></details><div className="inputs"><label>Lugar<select aria-label="Lugar" value={placeId} onChange={e=>setPlaceId(e.target.value)}><option value="">Elegí un lugar</option>{places.filter(x=>!x.archivado||x.id===placeId).map(x=><option key={x.id} value={x.id} disabled={x.archivado}>{x.nombre}{x.archivado?' (Archivado)':''}</option>)}</select></label><label>Nombre del consumo<input value={name} required maxLength={100} onChange={e=>setName(e.target.value)}/></label></div><div className="actions"><button disabled={busy||!placeId||!places.some(p=>p.id===placeId&&!p.archivado)||!draft.equipos.length||!!(editing&&configs.find(c=>c.id===editing.id)?.archivada)} onClick={save}>{editing?'Guardar cambios':'Guardar consumo'}</button>{editing&&<button onClick={()=>setEditing(null)}>Guardar como otro consumo</button>}</div>{places.map(p=><details className="place-detail" key={p.id}><summary>{p.nombre} · {p.archivado?'Archivado':'Activo'}</summary><PlaceEditor place={p} busy={busy} save={value=>run(async()=>{await api('/lugares/'+p.id,'PUT',value);await refresh('places')})}/><div className="actions"><button disabled={busy} onClick={()=>archive('lugares',p)}>{p.archivado?'Restaurar lugar':'Archivar lugar'}</button><button className="danger-button" disabled={busy} onClick={()=>remove('lugares',p)}>Eliminar lugar</button></div></details>)}<details className="solar-options"><summary>Ajustar condiciones para calcular paneles</summary><p>Se usarán en el próximo análisis o comparación del historial.</p><div className="inputs"><label>Horas solares pico manuales<input type="number" min="0.5" max="8" step="0.1" value={sun} onChange={e=>setSun(e.target.value)}/></label><label>Rendimiento global<input type="number" min="0.1" max="1" step="0.01" value={yieldFactor} onChange={e=>setYield(e.target.value)}/></label><label>Cobertura objetivo<input type="number" min="0.01" max="1" step="0.01" value={coverage} onChange={e=>setCoverage(e.target.value)}/></label></div><p>Son condiciones manuales de ejemplo. Verificá las horas de sol y el rendimiento para tu lugar.</p></details><h3>Tus consumos guardados</h3><p>Calcular paneles guarda el resultado en el historial. Archivar aparta un consumo sin eliminarlo.</p><label className="inline-check"><input type="checkbox" checked={showArchived} onChange={e=>setShowArchived(e.target.checked)}/>Mostrar consumos archivados</label>{configs.filter(config=>showArchived||!config.archivada).map(config=><article className="saved" key={config.id}><h3>{config.nombre} {config.archivada&&'· Archivada'}</h3><div className="actions"><button disabled={busy||config.archivada} onClick={()=>{onRestore(config.consumo);setEditing(config);setName(config.nombre);setPlaceId(config.lugarId);setNoticeError(false);setNotice('Consumo abierto. Modificá los equipos y elegí Guardar cambios.');document.querySelector('form.workspace')?.scrollIntoView({behavior:'smooth',block:'start'})}}>Abrir para editar</button><button disabled={busy} onClick={()=>run(async()=>{await api('/configuraciones/'+config.id+'/duplicar','POST');await refresh('configs')})}>Duplicar</button><button disabled={busy||config.archivada||!places.some(p=>p.id===config.lugarId&&!p.archivado)} onClick={()=>evaluate(config)}>Calcular paneles</button><button disabled={busy} onClick={()=>archive('configuraciones',config)}>{config.archivada?'Restaurar':'Archivar'}</button><button className="danger-button" disabled={busy} onClick={()=>remove('configuraciones',config)}>Eliminar consumo</button></div><details><summary>Ver ficha del consumo</summary><p>Identificador: {config.id}</p><p>Versión de contenido: {config.version}</p><p>Lugar: {places.find(p=>p.id===config.lugarId)?.nombre}</p></details></article>)}<h3>Historial de resultados</h3>{history.length===0&&<p>Todavía no calculaste paneles para tus consumos guardados.</p>}{history.map(evaluation=><details key={evaluation.id}><summary>{configs.find(c=>c.id===evaluation.configuracionId)?.nombre||'Consumo eliminado'} · configuración v{evaluation.versionConfiguracion} {configs.some(c=>c.id===evaluation.configuracionId&&c.version!==evaluation.versionConfiguracion)?'· Versión anterior':''} · {evaluation.snapshot.consumo.totalPeriodoKWh} kWh</summary><p className="trace">Evaluación: {evaluation.id}</p><p>Lugar del cálculo: {evaluation.snapshot.lugar.nombre} · v{evaluation.snapshot.lugar.version}.</p><details><summary>Comparar con otras condiciones solares</summary><p>Usa el mismo consumo histórico con las condiciones elegidas en «Ajustar condiciones para calcular paneles». El resultado anterior se conserva.</p><button disabled={busy} onClick={()=>newStudy(evaluation)}>Calcular otra alternativa</button></details><details><summary>Ver fichas y parámetros congelados</summary><p>Estado técnico de publicación al consultar: {evaluation.publicacion}.</p><pre>{JSON.stringify(evaluation.snapshot,null,2)}</pre></details>{evaluation.estudioIds.map(id=>{const study=studies.find(s=>s.id===id);return <Study key={id} study={study} id={id}/>})}</details>)}{session.usuario.rol==='ADMINISTRADOR'&&<Admin api={api} run={run} busy={busy}/>}</>}</section>
}

/**
 * Permite editar y archivar un lugar utilizando control de versión.
 *
 * @method PlaceEditor
 * @param {Object} props - Lugar, estado de espera y función de guardado.
 * @returns {React.ReactElement} Formulario de lugar propio.
 */
function PlaceEditor({place,busy,save}){return <form onSubmit={e=>{e.preventDefault();const data=new FormData(e.currentTarget);save({...place,nombre:data.get('nombre'),ubicacion:data.get('ubicacion'),superficieM2:data.get('superficie')?Number(data.get('superficie')):null})}}><div className="inputs"><label>Nombre<input key={place.version} name="nombre" defaultValue={place.nombre} required maxLength={100}/></label><label>Ubicación<input name="ubicacion" defaultValue={place.ubicacion} maxLength={200}/></label><label>Superficie m²<input name="superficie" type="number" step="0.000001" min="0.000001" defaultValue={place.superficieM2??''}/></label></div><button disabled={busy}>Guardar lugar</button><details><summary>Ver ficha del lugar</summary><p>Identificador: {place.id}</p><p>Versión de contenido: {place.version}</p></details></form>}

/**
 * Presenta el estado durable y las alternativas de un estudio.
 *
 * @method Study
 * @param {Object} props - Estudio recuperado y su identificador.
 * @returns {React.ReactElement} Comparación y trazabilidad del cálculo.
 */
function Study({study,id}){const result=study?.resultado;return <article className="saved"><h3>Estudio {id.slice(0,8)} · {{PENDIENTE:'En espera',EN_PROCESAMIENTO:'Calculando',COMPLETADO:'Listo',SIN_ALTERNATIVA:'Sin alternativa viable',ERROR:'Pendiente de reintento'}[study?.estado]||'En espera'}</h3>{study?.error&&<p>{study.error}</p>}{result&&<><p>{result.nota}</p><p>Criterio: {result.criterio}. HSP manual: {result.supuestos.horasSolPico}; rendimiento: {result.supuestos.factorRendimiento}; cobertura objetivo: {result.supuestos.coberturaObjetivo}.</p><div className="table-scroll"><table><thead><tr><th>Modelo</th><th>Paneles</th><th>kWh/día</th><th>Cobertura</th><th>m²</th><th>Cumple</th></tr></thead><tbody>{result.alternativas.map(a=><tr key={a.panel.id}><td>{a.panel.nombre} · v{a.panel.version}<small>{a.panel.fuente}</small></td><td>{a.cantidad}</td><td>{a.generacionDiariaKWh}</td><td>{(Number(a.coberturaAlcanzada)*100).toFixed(2)}%</td><td>{a.superficieM2}</td><td>{a.cumpleObjetivo?'Sí':'No'}</td></tr>)}</tbody></table></div>{result.cotizacion&&<p>Oferta del vendedor simulado: {result.cotizacion.cantidad} paneles · {result.cotizacion.totalCentavos/100} {result.cotizacion.moneda}. No reserva stock.</p>}<p className="trace">Traza: {study.traceId}</p></>}</article>}

/**
 * Administra catálogos y roles con permisos verificados por backend.
 *
 * @method Admin
 * @param {Object} props - Cliente autenticado, manejador de acciones y estado.
 * @returns {React.ReactElement} Formularios administrativos y operaciones del índice y DLQ.
 */
function Admin({api,run,busy}){
 const [kind,setKind]=useState('electrodomesticos'),[items,setItems]=useState([]),[selected,setSelected]=useState(null),[users,setUsers]=useState([]),[info,setInfo]=useState('');
 const equipment={id:'',nombre:'',categoria:'General',modo:'POTENCIA',potenciaW:100,energiaPorCicloKWh:0,factorFuncionamiento:1,fuente:'Ficha declarada por administrador',activo:true};
 const panel={id:'',nombre:'',potenciaW:450,superficieEfectivaMilliM2:2500,precioCentavos:25000000,moneda:'ARS',fuente:'Ficha declarada por administrador',activo:true};
 /**
  * Carga el catálogo administrativo elegido sin ocultar fichas desactivadas.
  *
  * @method load
  * @param {string} value - Catálogo a administrar.
  * @returns {Promise<void>} Actualiza la lista y limpia la selección previa.
  */
 async function load(value=kind){setKind(value);setSelected(null);setItems(await api('/admin/'+value))}
 /**
  * Valida el formulario con el contrato del servidor y persiste su versión.
  *
  * @method submit
  * @param {React.FormEvent} event - Datos de la ficha.
  * @returns {Promise<void>} Guarda y vuelve a cargar el catálogo.
  */
 async function submit(event){event.preventDefault();const data=new FormData(event.currentTarget),input={};const template=selected|| (kind==='paneles'?panel:equipment);for(const key of Object.keys(template)){if(key==='version')input.version=template.version;else if(key==='activo')input.activo=data.get(key)==='on';else input[key]=typeof template[key]==='number'?Number(data.get(key)):data.get(key)}await run(async()=>{await api('/admin/'+kind+(selected?'/'+selected.id:''),selected?'PUT':'POST',input);await load()})}
 const current=selected||(kind==='paneles'?panel:equipment);
 return <details className="admin"><summary>Administración</summary><div className="actions"><button disabled={busy} onClick={()=>run(()=>load('electrodomesticos'))}>Electrodomésticos</button><button disabled={busy} onClick={()=>run(()=>load('paneles'))}>Paneles</button><button disabled={busy} onClick={()=>run(async()=>setUsers(await api('/admin/usuarios')))}>Usuarios y roles</button><button disabled={busy} onClick={()=>run(async()=>setInfo(JSON.stringify(await api(kind==='paneles'?'/admin/paneles/reindexar':'/admin/catalogo/reindexar','POST'))))}>Reconstruir índice</button><button disabled={busy} onClick={()=>run(async()=>setInfo(JSON.stringify(await api('/admin/mensajeria'))))}>Consultar colas</button><button disabled={busy} onClick={()=>run(async()=>setInfo(JSON.stringify(await api('/admin/mensajeria/reprocesar','POST',{limite:10}))))}>Reprocesar hasta 10 mensajes DLQ</button></div><p>{info}</p><h3>Catálogo: {kind}</h3><select value={selected?.id||''} onChange={e=>setSelected(items.find(x=>x.id===e.target.value)||null)}><option value="">Nueva ficha</option>{items.map(item=><option key={item.id} value={item.id}>{item.nombre} · v{item.version} {item.activo?'':'· Inactiva'}</option>)}</select><form key={kind+':'+(selected?.id||'new')+':'+(selected?.version||0)} onSubmit={submit}><div className="inputs">{Object.entries(current).filter(([key])=>key!=='version').map(([key,value])=><label key={key}>{key}{typeof value==='boolean'?<input name={key} type="checkbox" defaultChecked={value}/>:<input name={key} required={key!=='fuente'} type={typeof value==='number'?'number':'text'} step="any" defaultValue={value} readOnly={key==='id'&&!!selected}/>}</label>)}</div><button disabled={busy}>Guardar ficha</button></form>{users.map(user=><div className="actions" key={user.id}><span>{user.email}</span><select aria-label={'Rol de '+user.email} value={user.rol} disabled={busy} onChange={e=>{const rol=e.target.value;run(async()=>{await api('/admin/usuarios/'+user.id+'/rol','PUT',{rol});setUsers(await api('/admin/usuarios'))})}}><option>USUARIO</option><option>ADMINISTRADOR</option></select></div>)}</details>
}
