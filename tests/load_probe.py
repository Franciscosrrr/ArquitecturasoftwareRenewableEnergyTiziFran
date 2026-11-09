"""Ensayo local breve: baja tasa y ráfaga con el limitador habilitado."""
import concurrent.futures
import json
import os
import statistics
import time
from collections import Counter
from datetime import datetime, timezone
from pathlib import Path
from urllib.request import Request, urlopen
from urllib.error import HTTPError

ROOT=Path(__file__).resolve().parents[1]
BASE=os.getenv('BASE_URL','http://127.0.0.1:8080')
BODY=(ROOT/'docs/contracts/v1/examples/solicitud-valida.json').read_bytes()
def one(_=None):
    """/**
     * Mide una solicitud completa incluyendo lectura de su respuesta.
     *
     * @method one
     * @param {Any} _ - Parámetro _ de la operación.
     * @returns {Any} Resultado verificado de la operación.
     */
    """
    req=Request(BASE+'/api/v1/publico/consumo',data=BODY,headers={'Content-Type':'application/json'})
    start=time.perf_counter()
    try:
        with urlopen(req,timeout=10) as res:
            res.read();status=res.status
    except HTTPError as err:
        err.read();status=err.code
    return {'status':status,'ms':round((time.perf_counter()-start)*1000,3)}
def summary(values):
    """/**
     * Resume latencias, throughput, estados y origen de resultados.
     *
     * @method summary
     * @param {Any} values - Parámetro values de la operación.
     * @returns {Any} Resultado verificado de la operación.
     */
    """
    latencies=sorted(v['ms'] for v in values)
    return {'requests':len(values),'status':dict(Counter(str(v['status']) for v in values)), 'p50_ms':statistics.median(latencies),'p95_ms':latencies[max(0,int(len(latencies)*.95)-1)],'samples':values}
if __name__=='__main__':
    low=[]
    for _ in range(12):
        start=time.perf_counter();low.append(one());time.sleep(max(0,2-(time.perf_counter()-start)))
    with concurrent.futures.ThreadPoolExecutor(max_workers=5) as pool:
        burst=list(pool.map(one,range(40)))
    result={'at_utc':datetime.now(timezone.utc).isoformat(),'environment':'Docker Desktop local; gateway, Consumo, Electrodomésticos, MySQL y Jaeger en un host; sin límite de recursos fijado','endpoint':'POST /api/v1/publico/consumo','low_rate_0_5_rps':summary(low),'burst_40_concurrency_5':summary(burst),'limitations':['Muestra de 12 solicitudes calientes; no es ensayo de larga duración.','La ráfaga verifica política de tráfico: 1 req/s y burst 10.','No mide throughput máximo de negocio ni saturación de CPU/DB.','M2M usa las mismas reglas y otra identidad de limitador; este ensayo no estima capacidad cloud.']}
    target=ROOT/'docs/evidence/entrega2/carga.json';target.parent.mkdir(parents=True,exist_ok=True);target.write_text(json.dumps(result,ensure_ascii=False,indent=2),encoding='utf-8')
    print(json.dumps({k:v for k,v in result.items() if k not in ('limitations','environment')},ensure_ascii=False))
