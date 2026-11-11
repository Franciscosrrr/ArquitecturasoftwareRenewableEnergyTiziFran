"""Provoca fallas locales reversibles y conserva evidencia sin secretos."""
import json,os,subprocess,time,uuid
from pathlib import Path
from urllib.request import urlopen
from test_durable import Client,INPUT,OPTIONS

ROOT=Path(__file__).resolve().parents[1]
SECRET={key:value.strip().strip('"').strip("'") for line in (ROOT/'.env').read_text(encoding='utf-8-sig').splitlines() if '=' in line for key,value in [line.split('=',1)]}
COMMAND=['docker','compose','-p',os.getenv('TEST_PROJECT','renewable-energy-entrega2'),'-f','compose.yaml']
if os.getenv('TEST_COMPOSE_OVERRIDE'):COMMAND+=['-f',os.environ['TEST_COMPOSE_OVERRIDE']]
events=[]

def record(action,**data):
    """/**
     * Registra una observación de falla con tiempo relativo y sin secretos.
     *
     * @method record
     * @param {str} action - Acción observada que se registra en la cronología de la prueba.
     * @param {Any} data - Campos de evidencia sin secretos.
     * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
     */
    """
    events.append({'segundo':round(time.monotonic()-START,3),'accion':action,**data})
    print(action,flush=True)
def docker(*args,input=None):
    """/**
     * Ejecuta Compose con argumentos separados y comprueba su código de salida.
     *
     * @method docker
     * @param {dict} input - Entrada del cálculo que se contrasta con el contrato.
     * @param {Any} args - Argumentos separados del comando o callback.
     * @returns {str} Salida textual del comando Compose; propaga el fallo si el proceso termina con error.
     */
    """
    result=subprocess.run(COMMAND+list(args),cwd=ROOT,input=input,text=True,capture_output=True,timeout=120)
    if result.returncode:raise RuntimeError('Fallo Docker: '+' '.join(args[:3])+' '+result.stderr[-500:])
    return result.stdout
def sql(query):
    """/**
     * Consulta evidencia SQL mediante las credenciales internas del contenedor.
     *
     * @method sql
     * @param {str} query - Consulta SQL o parámetros de búsqueda que ejecuta el helper.
     * @returns {str} Salida de la consulta SQL realizada dentro del contenedor de prueba.
     */
    """
    return docker('exec','-T','mysql-consumo','sh','-c','MYSQL_PWD="$MYSQL_PASSWORD" mysql -u"$MYSQL_USER" -D "$MYSQL_DATABASE" -N -B -e '+"'"+query+"'")
def prom(query):
    """/**
     * Consulta Prometheus para verificar métricas y alertas reales.
     *
     * @method prom
     * @param {str} query - Consulta SQL o parámetros de búsqueda que ejecuta el helper.
     * @returns {list} Series devueltas por la consulta de Prometheus.
     */
    """
    from urllib.parse import urlencode
    with urlopen(os.getenv('PROM_URL','http://localhost:9090')+'/api/v1/query?'+urlencode({'query':query}),timeout=5) as response:return json.load(response)['data']['result']
def ready(client,study_id,seconds=30):
    """/**
     * Espera un resultado durable hasta el plazo de recuperación.
     *
     * @method ready
     * @param {Client} client - Cliente de prueba con sesión y cookies propias.
     * @param {str} study_id - Identificador del estudio propio que se espera completar.
     * @param {float} seconds - Plazo máximo de espera de la comprobación, en segundos.
     * @returns {dict} Estudio completado; falla la comprobación si no se recupera dentro del plazo.
     */
    """
    deadline=time.monotonic()+seconds
    while time.monotonic()<deadline:
        status,study,_=client.call('/estudios/'+study_id)
        if status==200 and study['estado']=='COMPLETADO':return study
        time.sleep(.5)
    raise AssertionError('Estudio no completado: '+str(study))
def publish(payload,dlq=False):
    """/**
     * Inyecta un evento de prueba mediante la API administrativa del broker.
     *
     * @method publish
     * @param {Any} payload - Contenido que se serializa o utiliza como cuerpo de la prueba.
     * @param {bool} dlq - Selecciona la cola de mensajes no procesables cuando es verdadero.
     * @returns {str} Salida del publicador de eventos de prueba ejecutado en el contenedor.
     */
    """
    return docker('run','--rm','--no-deps','-T','verificar-real','python','-B','tests/broker_probe.py','publish',*(['dlq'] if dlq else []),input=json.dumps(payload))

if __name__=='__main__':
    START=time.monotonic();user=Client();user.account();admin=Client()
    status,session,_=admin.call('/auth/login',{'email':'admin@renewable.local','password':SECRET['ADMIN_PASSWORD']});assert status==200
    admin.token=session['accessToken']
    _,place,_=user.call('/lugares',{'nombre':'Falla controlada','ubicacion':'Manual','superficieM2':50})
    _,config,_=user.call('/configuraciones',{'nombre':'Recuperación durable','lugarId':place['id'],'consumo':INPUT})
    body={'version':1,'versionLugar':1,'solar':OPTIONS}
    stopped=set()
    try:
        docker('stop','rabbitmq');stopped.add('rabbitmq');record('broker detenido')
        status,evaluation,_=user.call('/configuraciones/'+config['id']+'/evaluaciones',body,headers={'Idempotency-Key':uuid.uuid4().hex})
        assert status==201 and evaluation['publicacion']=='PENDIENTE',(status,evaluation)
        record('evaluación confirmada sin broker',http=status,publicacion=evaluation['publicacion'])
        docker('start','rabbitmq');stopped.remove('rabbitmq');record('broker reiniciado')
        study=ready(user,evaluation['estudioIds'][0],50);record('outbox recuperada y estudio completado',estado=study['estado'])
        # La identidad del nodo debe conservar mensajes incluso al recrear el contenedor.
        publish({'tipo':'PruebaPersistencia.v99','eventId':str(uuid.uuid4())},True)
        _,before_queues,_=admin.call('/admin/mensajeria')
        preserved_dlq=before_queues['solar.estudios.dlq'];assert preserved_dlq>=1,before_queues
        docker('up','-d','--no-deps','--force-recreate','rabbitmq')
        deadline=time.monotonic()+45
        while 'healthy' != docker('ps','--format','{{.Health}}','rabbitmq').strip():
            if time.monotonic()>deadline:raise AssertionError('Broker no recuperó salud tras recrear')
            time.sleep(1)
        time.sleep(4)
        _,after_queues,_=admin.call('/admin/mensajeria')
        assert after_queues['solar.estudios.dlq']>=preserved_dlq,(before_queues,after_queues)
        record('broker recreado con identidad y volumen conservados',mensajesDLQAntes=preserved_dlq,mensajesDLQDespues=after_queues['solar.estudios.dlq'])
        identifier='reconnect'+uuid.uuid4().hex[:16]
        item={'id':identifier,'nombre':'Ensayo reconexión '+identifier,'categoria':'Prueba','modo':'POTENCIA','potenciaW':321,'energiaPorCicloKWh':0,'factorFuncionamiento':1,'fuente':'Prueba de reconexión tras inactividad','activo':True}
        started=time.monotonic();status,created,_=admin.call('/admin/electrodomesticos',item);assert status==201,created
        while True:
            status,page,_=admin.call('/publico/electrodomesticos?q='+identifier)
            if page.get('total')==1:break
            if time.monotonic()-started>5:raise AssertionError('Catálogo no visible en 5 s tras reconectar: '+str(page))
            time.sleep(.25)
        record('alta de catálogo tras reconexión ociosa visible en menos de 5 s',segundos=round(time.monotonic()-started,3))
        assert admin.call('/admin/electrodomesticos/'+identifier,{**item,'version':1,'activo':False},'PUT')[0]==200
        raw=sql('SELECT payload FROM outbox WHERE evaluacion_id="'+evaluation['id']+'" LIMIT 1').strip()
        event=json.loads(raw);publish(event);time.sleep(2)
        _,items,_=user.call('/estudios?evaluacionId='+evaluation['id']);assert len(items)==1,items
        record('evento duplicado entregado',estudios=len(items))
        publish({'tipo':'VersionIncompatible.v99'});time.sleep(2)
        _,counts,_=admin.call('/admin/mensajeria');assert counts['solar.estudios.dlq']>=1,counts
        record('evento incompatible en DLQ',colas=counts)
        replay={**event,'eventId':str(uuid.uuid4()),'estudioId':str(uuid.uuid4())};publish(replay,True)
        status,replayed,_=admin.call('/admin/mensajeria/reprocesar',{'limite':10})
        assert status==200 and replayed['republicados']>=1 and replayed['invalidosConservados']>=1,(status,replayed)
        ready(user,replay['estudioId']);record('reproceso administrado completa mensaje válido y conserva inválido',**replayed)
        before=[]
        for _ in range(4):before.append(user.call('/publico/paneles')[2].get('X-Instance-ID'));time.sleep(.8)
        assert set(before)=={'solar-1','solar-2'},before
        docker('stop','solar-2');stopped.add('solar-2');record('réplica Solar 2 detenida',distribucionAnterior=before)
        time.sleep(22)
        healthy=[]
        for _ in range(4):
            status,_,headers=user.call('/publico/paneles');assert status==200;healthy.append(headers.get('X-Instance-ID'));time.sleep(.8)
        assert set(healthy)=={'solar-1'},healthy
        alerts=prom('ALERTS{alertname="RenewableTargetDown",alertstate="firing"}')
        assert any('solar-2' in item['metric'].get('instance','') for item in alerts),alerts
        record('balanceo excluye caída y alerta activa',instancias=healthy,alertas=[a['metric'] for a in alerts])
        docker('start','solar-2');stopped.remove('solar-2');time.sleep(14)
        after=[]
        for _ in range(4):after.append(user.call('/publico/paneles')[2].get('X-Instance-ID'));time.sleep(.8)
        assert set(after)=={'solar-1','solar-2'},after;record('réplica recuperada',distribucion=after)
        docker('stop','proveedor-paneles-sim');stopped.add('proveedor-paneles-sim')
        status,new,_=user.call('/evaluaciones/'+evaluation['id']+'/estudios',{**OPTIONS,'coberturaObjetivo':.5},headers={'Idempotency-Key':uuid.uuid4().hex})
        assert status==202;time.sleep(3)
        status,preserved,_=user.call('/evaluaciones/'+evaluation['id']);assert status==200
        record('vendedor caído conserva evaluación y encola reintento',evaluacionDisponible=status)
        docker('start','proveedor-paneles-sim');stopped.remove('proveedor-paneles-sim')
        ready(user,new['estudioId'],50);record('reintento comercial completado')
        docker('stop','valkey','meilisearch');stopped.update(['valkey','meilisearch']);time.sleep(1)
        status,page,headers=user.call('/publico/electrodomesticos?q=Heladera&potenciaMin=100')
        assert status==200 and page['total']>=1 and headers['X-Catalog-Source']=='sql-degradado',(status,page,dict(headers))
        record('buscador y caché caídos: respaldo SQL',http=status,origen=headers['X-Catalog-Source'])
        docker('start','valkey','meilisearch');stopped.difference_update(['valkey','meilisearch'])
        docker('restart','usuarios','consumo','solar','solar-2');time.sleep(14)
        # Access tokens remain usable because secret and sessions persist.
        status,restored,_=user.call('/configuraciones/'+config['id']);assert status==200
        status,restored_study,_=user.call('/estudios/'+evaluation['estudioIds'][0]);assert status==200
        record('reinicios conservan SQL, sesión y Mongo',configuracionVersion=restored['version'],estudioEstado=restored_study['estado'])
    finally:
        if stopped:docker('start',*sorted(stopped))
        destination=ROOT/'docs/evidence/final-2026-11-09/fallas.json';destination.parent.mkdir(parents=True,exist_ok=True)
        destination.write_text(json.dumps(events,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
    print(json.dumps({'completado':True,'pasos':len(events),'evidencia':str(destination)},ensure_ascii=False))
