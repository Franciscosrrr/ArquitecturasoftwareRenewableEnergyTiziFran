"""Comprobación optativa de la entrada web, sin activar túneles ni imprimir credenciales."""
import hashlib,json,os,re,time,uuid
from http.cookies import SimpleCookie
from pathlib import Path
from urllib.error import HTTPError
from urllib.parse import urlsplit
from urllib.request import Request,urlopen


def call(base,path,data=None,token='',cookie='',method=None,origin=None,extra=None):
    """/**
     * Consulta una ruta real de la cuenta sintética sin registrar credenciales.
     *
     * @method call
     * @param {str} base - Origen interno de prueba o URL HTTPS autorizada.
     * @param {str} path - Ruta de la aplicación.
     * @param {dict} data - JSON opcional de una cuenta sintética.
     * @param {str} token - Token temporal de esa cuenta.
     * @param {str} cookie - Cookie temporal de esa cuenta.
     * @param {str} method - Método HTTP opcional.
     * @param {str} origin - Origen opcional para comprobar protección de sesión.
     * @param {dict} extra - Cabeceras públicas de comprobación.
     * @returns {tuple} Estado, bytes y cabeceras, sin registrar secretos.
     */
    """
    time.sleep(.4)
    headers={'Content-Type':'application/json',**(extra or {})}
    if token:headers['Authorization']='Bearer '+token
    if cookie:headers['Cookie']=cookie
    if origin is not None:headers['Origin']=origin
    request=Request(base+path,data=None if data is None else json.dumps(data).encode(),method=method,headers=headers)
    try:response=urlopen(request,timeout=15)
    except HTTPError as error:response=error
    with response:return response.status,response.read(),response.headers


def refresh_cookie(headers,secure):
    """/**
     * Verifica los atributos de la cookie de sesión sin incluir su valor en evidencia.
     *
     * @method refresh_cookie
     * @param {object} headers - Cabeceras de una respuesta de identidad.
     * @param {bool} secure - Transporte requerido para ese acceso.
     * @returns {str} Cookie temporal para probar rotación y cierre.
     */
    """
    parsed=SimpleCookie();parsed.load(headers['Set-Cookie']);cookie=parsed['renewable_refresh']
    assert bool(cookie['secure'])==secure and cookie['httponly'] and cookie['samesite'].lower()=='strict'
    assert cookie['path']=='/api/v1/auth'
    return 'renewable_refresh='+cookie.value


def verify():
    """/**
     * Comprueba página, API y sesiones mediante la entrada web preparada.
     *
     * @method verify
     * @returns {dict} Evidencia sin claves ni datos personales reales.
     */
    """
    base=os.environ.get('WEB_EDGE_URL','http://web-edge:8080').rstrip('/')
    direct=os.environ.get('DIRECT_URL','http://gateway:8080').rstrip('/')
    assert urlsplit(base).scheme=='https' or urlsplit(base).hostname=='web-edge'
    status,html,_=call(base,'/');assert status==200
    status,original,_=call(direct,'/');assert status==200 and html==original
    asset=re.search(rb'src="(/assets/[^" ]+\.js)"',html).group(1).decode()
    status,js,_=call(base,asset);assert status==200
    status,original_js,_=call(direct,asset);assert status==200 and js==original_js
    status,panels,_=call(base,'/api/v1/publico/paneles/busqueda?potenciaMin=500&moneda=ARS');assert status==200 and json.loads(panels)['total']>0
    status,body,_=call(base,'/api/v1/publico/consumo-catalogo',{'diasPeriodo':30,'equipos':[{'catalogoId':'iluminacion','cantidad':4,'horasPorDia':5,'diasUso':30,'factorFuncionamiento':1}]});assert status==200 and json.loads(body)['totalPeriodoKWh']==6
    for path in ['/health/ready','/metrics','/internal/v1/identidad/validar','/.env']:
        assert call(base,path)[0]==404,path
    assert call(base,'/api/v1/lugares')[0]==401
    assert call(base,'/api/v1/lugares',extra={'X-Actor-ID':'ajeno','X-Actor-Role':'ADMINISTRADOR'})[0]==401
    credentials={'email':'web-edge-'+uuid.uuid4().hex+'@example.test','password':'credencial-sintetica-solo-verificacion'}
    origin=base
    status,_,_=call(base,'/api/v1/auth/registro',credentials,origin=origin);assert status==201
    status,session,headers=call(base,'/api/v1/auth/login',credentials,origin=origin);assert status==200
    access=json.loads(session)['accessToken'];cookie=refresh_cookie(headers,True)
    assert call(base,'/api/v1/auth/refresh',{},cookie=cookie,origin='https://otro-origen.invalid')[0]==403
    status,session,headers=call(base,'/api/v1/auth/refresh',{},cookie=cookie,origin=origin);assert status==200
    access=json.loads(session)['accessToken'];cookie=refresh_cookie(headers,True)
    place=None
    try:
        status,raw,_=call(base,'/api/v1/lugares',{'nombre':'Lugar de prueba de entrada web','ubicacion':'Manual'},token=access);assert status==201
        place=json.loads(raw)
        status,raw,_=call(direct,'/api/v1/lugares',token=access);assert status==200 and any(p['id']==place['id'] for p in json.loads(raw))
    finally:
        if place:assert call(base,'/api/v1/lugares/'+place['id'],{'version':place['version']},token=access,method='DELETE')[0]==204
    status,_,headers=call(base,'/api/v1/auth/logout',{},cookie=cookie,origin=origin);assert status==204
    refresh_cookie(headers,True)
    assert call(base,'/api/v1/auth/refresh',{},cookie=cookie,origin=origin)[0]==401
    status,_,headers=call(direct,'/api/v1/auth/login',credentials,origin=direct);assert status==200
    local_cookie=refresh_cookie(headers,False)
    assert call(direct,'/api/v1/auth/logout',{},cookie=local_cookie,origin=direct)[0]==204
    return {'completado':True,'baseProbada':base,'httpsPublicoProbado':urlsplit(base).scheme=='https','paginaYJavaScriptIdenticos':True,'htmlSHA256':hashlib.sha256(html).hexdigest(),'busquedaSolarYCalculoInvitado':True,'cuentaRegistroLoginRefreshLogout':True,'cookieSecureSoloEntradaWeb':True,'otroOrigenRechazado':True,'rutasTecnicasBloqueadas':True,'privadosSinSesionDenegados':True,'datosCompartidosConAccesoLocal':True,'lugarSinteticoEliminado':True}


if __name__=='__main__':
    evidence=verify()
    destination=Path(os.environ.get('WEB_EDGE_EVIDENCE','/tmp/web-edge.json'))
    destination.write_text(json.dumps(evidence,ensure_ascii=False,indent=2)+'\n',encoding='utf-8')
    print(json.dumps(evidence,ensure_ascii=False))
