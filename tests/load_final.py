"""Carga reproducible de catálogo y capacidad M2M con métricas de caché reales."""
import concurrent.futures
import json
import os
import statistics
import time
from collections import Counter
from urllib.request import Request,urlopen
from urllib.error import HTTPError
from test_durable import Client,INPUT,OPTIONS

def one(url,headers=None,body=None):
    """/**
     * Mide una solicitud completa incluyendo lectura de su respuesta.
     *
     * @method one
     * @param {Any} url - Parámetro url de la operación.
     * @param {Any} headers - Parámetro headers de la operación.
     * @param {Any} body - Parámetro body de la operación.
     * @returns {Any} Resultado verificado de la operación.
     */
    """
    started=time.perf_counter()
    request=Request(url,headers=headers or {},data=None if body is None else json.dumps(body).encode())
    try:response=urlopen(request,timeout=10)
    except HTTPError as error:response=error
    response.read()
    return (time.perf_counter()-started)*1000,response.status,response.headers.get('X-Catalog-Source','')

def summary(values,elapsed,concurrency=8):
    """/**
     * Resume latencias, throughput, estados y origen de resultados.
     *
     * @method summary
     * @param {Any} values - Parámetro values de la operación.
     * @param {Any} elapsed - Parámetro elapsed de la operación.
     * @param {Any} concurrency - Concurrencia efectiva del escenario.
     * @returns {Any} Resultado verificado de la operación.
     */
    """
    ordered=sorted(value[0] for value in values)
    return {'solicitudes':len(values),'concurrencia':concurrency,'duracionSegundos':round(elapsed,3),'reqPorSegundo':round(len(values)/elapsed,2),'p50Ms':round(statistics.median(ordered),3),'p95Ms':round(ordered[max(0,int(len(ordered)*.95)-1)],3),'maxMs':round(max(ordered),3),'estados':dict(Counter(str(v[1]) for v in values)),'origenes':dict(Counter(v[2] for v in values))}

def benchmark(cache):
    """/**
     * Compara una consulta caliente con y sin caché usando el mismo backend.
     *
     * @method benchmark
     * @param {Any} cache - Parámetro cache de la operación.
     * @returns {Any} Resultado verificado de la operación.
     */
    """
    catalog=os.getenv('CATALOG_BASE','http://electrodomesticos:8080')
    url=catalog+'/api/v1/publico/electrodomesticos?tamano=20&orden=nombre&q=referencia'
    headers={} if cache else {'Authorization':'Bearer '+os.environ['INTERNAL_TOKEN'],'X-Cache-Mode':'bypass'}
    one(url,headers)
    started=time.perf_counter()
    with concurrent.futures.ThreadPoolExecutor(max_workers=8) as pool:values=list(pool.map(lambda _:one(url,headers),range(400)))
    return summary(values,time.perf_counter()-started)

def asynchronous_flow():
    """/**
     * Mide confirmaciones reales y visibilidad de estudios con diez configuraciones independientes.
     *
     * @method asynchronous_flow
     * @returns {dict} Latencias HTTP y cota observada de visibilidad con polling de un segundo.
     */
    """
    import uuid
    client=Client();client.account()
    status,place,_=client.call('/lugares',{'nombre':'Carga de estudios','ubicacion':'Recurso manual','superficieM2':100});assert status==201
    durations=[];created={};visible={};started=time.perf_counter()
    for index in range(10):
        status,config,_=client.call('/configuraciones',{'nombre':'Carga '+str(index),'lugarId':place['id'],'consumo':INPUT});assert status==201,(status,config)
        time.sleep(.25);before=time.perf_counter()
        status,evaluation,_=client.call('/configuraciones/'+config['id']+'/evaluaciones',{'version':1,'versionLugar':1,'solar':OPTIONS},headers={'Idempotency-Key':uuid.uuid4().hex},pace=False)
        elapsed=(time.perf_counter()-before)*1000
        assert status==201,(status,evaluation)
        durations.append((max(elapsed,0),status,''));created[evaluation['estudioIds'][0]]=time.perf_counter()
        _,studies,_=client.call('/estudios')
        for study in studies:
            if study['id'] in created and study['id'] not in visible and study['estado']=='COMPLETADO':visible[study['id']]=time.perf_counter()-created[study['id']]
    deadline=time.monotonic()+30
    while len(visible)<len(created) and time.monotonic()<deadline:
        _,studies,_=client.call('/estudios')
        for study in studies:
            if study['id'] in created and study['id'] not in visible and study['estado']=='COMPLETADO':visible[study['id']]=time.perf_counter()-created[study['id']]
        time.sleep(1)
    ordered=sorted(visible.values());assert len(ordered)==10,{'completados':len(ordered)}
    return {'confirmaciones':summary(durations,time.perf_counter()-started,1),'estudiosCompletados':len(ordered),'p95VisibilidadSegundos':round(ordered[int(len(ordered)*.95)-1],3),'maxVisibilidadSegundos':round(max(ordered),3),'dentroDe15s':sum(value<=15 for value in ordered),'nota':'Confirmaciones secuenciales con pacing0.25s excluido de su duración HTTP; el throughput incluye preparar escenarios y observar estudios. Visibilidad observada por consulta, sin inferir el instante exacto de finalización.'}

if __name__=='__main__':
    result={'nota':'Misma consulta y backend caliente; bypass autorizado solo por red interna. Sin latencia artificial. No estima capacidad cloud.','sinCache':benchmark(False),'conCache':benchmark(True)}
    base=os.getenv('BASE_URL','http://gateway:8080')
    body={'diasPeriodo':30,'equipos':[{'id':'carga','modo':'POTENCIA','potenciaW':100,'cantidad':1,'horasPorDia':5,'diasUso':30,'factorFuncionamiento':1}]}
    values=[];started=time.perf_counter()
    for _ in range(15):
        values.append(one(base+'/api/v1/estimaciones-consumo',{'Content-Type':'application/json','X-API-Key':os.environ['M2M_API_KEY']},body));time.sleep(1.02)
    result['m2mCuotaPermitida']=summary(values,time.perf_counter()-started,1)
    result['flujoDurable']=asynchronous_flow()
    print(json.dumps(result,ensure_ascii=False,indent=2))
