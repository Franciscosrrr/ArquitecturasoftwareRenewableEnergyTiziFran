"""Verifica una publicación M2M HTTPS con credenciales de entorno y evidencia sin secretos."""
import concurrent.futures,hashlib,json,math,os,re,ssl,sys,time
from datetime import datetime,timezone
from pathlib import Path
from urllib.error import HTTPError
from urllib.parse import urlsplit
from urllib.request import HTTPRedirectHandler,HTTPSHandler,Request,build_opener


class NoRedirect(HTTPRedirectHandler):
    """/**
     * Impide reenviar credenciales a otro destino mediante una redirección HTTP.
     *
     * @method NoRedirect
     * @returns {NoRedirect} Política de transporte para la sonda pública.
     */
    """
    def redirect_request(self,request,response,code,message,headers,new_url):
        """/**
         * Rechaza el cambio de destino de una solicitud de comprobación.
         *
         * @method redirect_request
         * @param {object} request - Solicitud original.
         * @param {object} response - Respuesta de redirección.
         * @param {int} code - Estado HTTP.
         * @param {str} message - Texto del estado.
         * @param {object} headers - Cabeceras recibidas.
         * @param {str} new_url - Destino que no se seguirá.
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        return None


def call(base,path,body=None,key=None,headers=None,pace=True):
    """/**
     * Ejecuta HTTPS con certificado verificado, presupuesto acotado y respuesta limitada.
     *
     * @method call
     * @param {str} base - Origen HTTPS autorizado y observado del servicio.
     * @param {str} path - Ruta de la comprobación.
     * @param {bytes} body - JSON opcional transmitido sin datos personales reales.
     * @param {str} key - Credencial M2M mantenida solo en memoria.
     * @param {dict} headers - Cabeceras técnicas adicionales.
     * @param {bool} pace - Respeta la tasa sostenida salvo la ráfaga negativa explícita.
     * @returns {tuple} Estado, cuerpo, cabeceras y duración sin secretos.
     */
    """
    if pace:time.sleep(1.05)
    outgoing={'Content-Type':'application/json',**(headers or {})}
    if key:outgoing['X-API-Key']=key
    opener=build_opener(NoRedirect(),HTTPSHandler(context=ssl.create_default_context()))
    started=time.perf_counter()
    try:response=opener.open(Request(base+path,data=body,headers=outgoing),timeout=15)
    except HTTPError as error:response=error
    with response:return response.status,response.read(1048577),response.headers,round((time.perf_counter()-started)*1000,3)


def verify(base):
    """/**
     * Comprueba contrato, cálculo, permisos, errores y límite sobre el origen cloud real.
     *
     * @method verify
     * @param {str} base - URL HTTPS asignada por la plataforma.
     * @returns {dict} Evidencia reproducible que no incluye claves.
     */
    """
    parsed=urlsplit(base)
    assert parsed.scheme=='https' and parsed.netloc and not parsed.username and not parsed.query and not parsed.fragment
    root=Path(__file__).resolve().parents[1]
    key=os.environ['M2M_API_KEY'];denied=os.environ['M2M_DENIED_KEY']
    body=(root/'docs/contracts/v1/examples/solicitud-valida.json').read_bytes()
    status,raw,_,_=call(base,'/openapi.json');assert status==200
    expected=(root/'docs/contracts/v1/openapi.json').read_bytes().replace(b'\r\n',b'\n')
    assert raw.replace(b'\r\n',b'\n')==expected
    assert json.loads(raw)['info']['version']=='1.0.1'
    observations=[];path='/api/v1/estimaciones-consumo'
    samples=[]
    for _ in range(8):
        status,raw,_,duration=call(base,path,body,key);assert status==200
        result=json.loads(raw);assert result['totalPeriodoKWh']==47.4 and result['promedioDiarioKWh']==1.58
        assert re.fullmatch('[0-9a-f]{32}',result['traceId'])
        samples.append(duration)
    invalid=(root/'docs/contracts/v1/examples/solicitud-invalida-dias.json').read_bytes()
    cases=[('sin clave',body,None,None,401,'NO_AUTENTICADO'),('sin permiso',body,denied,None,403,'SIN_PERMISO'),('días inválidos',invalid,key,None,422,'VALIDACION'),('JSON inválido',b'{',key,None,400,'FORMATO_INVALIDO'),('tipo inválido',body,key,{'Content-Type':'text/plain'},415,'TIPO_CONTENIDO_NO_SOPORTADO'),('cuerpo excesivo',body+b' '*262145,key,None,413,'CUERPO_DEMASIADO_GRANDE')]
    for label,payload,credential,headers,expected_status,code in cases:
        status,raw,_,_=call(base,path,payload,credential,headers)
        assert status==expected_status,(label,status)
        error=json.loads(raw);assert error['codigo']==code and isinstance(error['detalles'],list)
        observations.append({'caso':label,'http':status,'codigo':code})
    for resource in ['/api/v1/lugares','/api/v1/admin/usuarios','/health/ready','/metrics','/privado-openapi.json']:
        status,_,_,_=call(base,resource)
        assert status in (401,404),(resource,status)
        observations.append({'ruta':resource,'http':status})
    health=os.environ.get('HEALTHCHECK_TOKEN')
    if health:
        status,_,_,_=call(base,'/health/ready',headers={'Authorization':'Bearer '+health});assert status==200
        observations.append({'caso':'readiness técnica autorizada','http':status})
    # Ráfaga pequeña con la clave sin permiso; no consume el bucket de la clave principal.
    with concurrent.futures.ThreadPoolExecutor(max_workers=3) as pool:
        futures=[pool.submit(call,base,path,body,denied,None,False) for _ in range(17)]
        burst=[future.result() for future in futures]
    assert all(status in (403,429) for status,_,_,_ in burst)
    limited=[headers for status,_,headers,_ in burst if status==429]
    assert limited and all(headers.get('Retry-After')=='1' for headers in limited)
    p95=sorted(samples)[math.ceil(len(samples)*.95)-1]
    return {'observadoUTC':datetime.now(timezone.utc).isoformat(),'completado':True,'baseUrl':base,'endpoint':base+path,'openapi':base+'/openapi.json','contratoVersion':'1.0.1','contratoSHA256LF':hashlib.sha256(expected).hexdigest(),'certificadoTLSVerificado':True,'estimacionesCorrectas':len(samples),'totalPeriodoKWh':47.4,'latenciasCalientesMs':samples,'p95CalienteMs':p95,'dentroDe1200ms':p95<=1200,'dentroDe5s':p95<=5000,'coldStartMedido':False,'erroresYRutas':observations,'rafagaClaveSinPermiso':{'solicitudes':17,'concurrencia':3,'rechazadas429':len(limited),'retryAfterSegundos':1},'alcance':'Medición desde esta conexión con instancia activa; no garantiza cold start ni prueba desde Clínica.'}


if __name__=='__main__':
    origin=sys.argv[1].rstrip('/')
    evidence=verify(origin)
    destination=Path(os.environ.get('CLOUD_EVIDENCE','docs/evidence/render-cloud-2026-11-09/https.json'))
    destination.parent.mkdir(parents=True,exist_ok=True)
    destination.write_text(json.dumps(evidence,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
    print(json.dumps(evidence,ensure_ascii=False))
