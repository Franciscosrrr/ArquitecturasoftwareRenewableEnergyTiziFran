"""Caída reversible del MySQL de este proyecto Compose y verificación de aislamiento."""
import json
import subprocess
import time
from datetime import datetime, timezone, timedelta
from pathlib import Path
from urllib.request import Request, urlopen
from urllib.error import HTTPError, URLError

ROOT=Path(__file__).resolve().parents[1]
BASE='http://127.0.0.1:8080'
CATALOG={'diasPeriodo':30,'equipos':[{'catalogoId':'iluminacion','cantidad':4,'horasPorDia':5,'diasUso':30,'factorFuncionamiento':1}]}
secret={line.split('=',1)[0]:line.split('=',1)[1] for line in (ROOT/'.env').read_text().splitlines() if '=' in line}
timeline=[]
def record(event,**details):
    """/**
     * Registra una observación de falla con tiempo relativo y sin secretos.
     *
     * @method record
     * @param {Any} event - Parámetro event de la operación.
     * @param {Any} details - Parámetro de la operación.
     * @returns {Any} Resultado verificado de la operación.
     */
    """
    timeline.append({'time':datetime.now(timezone(timedelta(hours=-3))).isoformat(),'event':event,**details})
def docker(*args):
    """/**
     * Ejecuta Compose con argumentos separados y comprueba su código de salida.
     *
     * @method docker
     * @param {Any} args - Argumentos separados del comando o callback.
     * @returns {Any} Resultado verificado de la operación.
     */
    """
    result=subprocess.run(['docker','compose',*args],cwd=ROOT,capture_output=True,text=True)
    if result.returncode:raise RuntimeError('Falló la operación Compose '+args[0])
    return result.stdout.strip()
def call(path,body,headers=None):
    """/**
     * Ejecuta HTTP real y devuelve estado, JSON y cabeceras sin ocultar errores.
     *
     * @method call
     * @param {Any} path - Parámetro path de la operación.
     * @param {Any} body - Parámetro body de la operación.
     * @param {Any} headers - Parámetro headers de la operación.
     * @returns {Any} Resultado verificado de la operación.
     */
    """
    start=time.perf_counter();req=Request(BASE+path,data=json.dumps(body).encode(),headers={'Content-Type':'application/json',**(headers or {})})
    try:response=urlopen(req,timeout=10)
    except HTTPError as err:response=err
    result=json.loads(response.read());return {'status':response.status,'ms':round((time.perf_counter()-start)*1000,3),'traceId':result.get('traceId')}
def count():
    """/**
     * Valida y ejecuta count según el contrato de prueba.
     *
     * @method count
     * @returns {Any} Resultado verificado de la operación.
     */
    """
    return int(docker('exec','-T','mysql','sh','-c','MYSQL_PWD="$MYSQL_PASSWORD" mysql -u"$MYSQL_USER" -N -D "$MYSQL_DATABASE" -e "SELECT COUNT(*) FROM electrodomesticos"'))
def wait_catalog():
    """/**
     * Valida y ejecuta wait catalog según el contrato de prueba.
     *
     * @method wait_catalog
     * @returns {Any} Resultado verificado de la operación.
     */
    """
    for _ in range(20):
        try:
            result=call('/api/v1/publico/consumo-catalogo',CATALOG)
            if result['status']==200:return result
        except (URLError,TimeoutError):pass
        time.sleep(2)
    raise RuntimeError('El catálogo no se recuperó dentro del presupuesto de ensayo')
if __name__=='__main__':
    initial=wait_catalog();before=count();record('baseline',response=initial,rows=before)
    try:
        docker('stop','mysql');record('mysql_stopped')
        failures=[]
        for _ in range(4):
            time.sleep(1.1);failures.append(call('/api/v1/publico/consumo-catalogo',CATALOG))
        if any(result['status']!=503 for result in failures):raise AssertionError(failures)
        record('catalog_unavailable',responses=failures)
        raw=json.loads((ROOT/'docs/contracts/v1/examples/solicitud-valida.json').read_text())
        independent=call('/api/v1/estimaciones-consumo',raw,{'X-API-Key':secret['M2M_API_KEY']})
        if independent['status']!=200:raise AssertionError(independent)
        record('m2m_remains_available',response=independent)
    finally:
        docker('start','mysql');record('mysql_start_requested')
    recovered=wait_catalog();record('catalog_recovered',response=recovered)
    docker('restart','electrodomesticos');record('catalog_restart_requested')
    wait_catalog();after=count();record('catalog_restart_idempotent',rows=after)
    if before!=after:raise AssertionError('Las semillas se duplicaron')
    out={'timezone':'America/Buenos_Aires (UTC-03:00)','scope':'Solo mysql y electrodomesticos del proyecto renewable-energy-entrega2; no se borraron volúmenes.','timeline':timeline,'conclusion':'Cuatro errores 503 controlados; capacidad M2M aislada responde 200; recuperación tras volver MySQL; semillas sin duplicados.'}
    target=ROOT/'docs/evidence/entrega2/fallo-controlado.json';target.write_text(json.dumps(out,ensure_ascii=False,indent=2),encoding='utf-8')
    print(json.dumps(out,ensure_ascii=False))
