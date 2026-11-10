"""Pruebas HTTP de identidad, propiedad, snapshots, outbox, Mongo e índice reales."""
import concurrent.futures
import http.cookiejar
import json
import os
import time
import unittest
import uuid
from urllib.request import Request, build_opener, HTTPCookieProcessor
from urllib.error import HTTPError

BASE = os.getenv('BASE_URL', 'http://localhost:8080')
INPUT = {'diasPeriodo':30,'equipos':[{'catalogoId':'iluminacion','cantidad':4,'horasPorDia':5,'diasUso':30,'factorFuncionamiento':1}]}
OPTIONS = {'horasSolPico':4,'factorRendimiento':0.8,'coberturaObjetivo':1}

class Client:
    """/**
     * Agrupa verificaciones del flujo Client.
     *
     * @method Client
     * @returns {Client} Resultado verificado de la operación.
     */
    """
    def __init__(self):
        """/**
         * Inicializa el cliente de prueba con cookies y credencial aisladas.
         *
         * @method __init__
         * @returns {Any} Resultado verificado de la operación.
         */
        """
        self.jar=http.cookiejar.CookieJar()
        self.opener=build_opener(HTTPCookieProcessor(self.jar))
        self.token=''

    def call(self,path,data=None,method=None,headers=None,pace=True):
        """/**
         * Ejecuta HTTP real y devuelve estado, JSON y cabeceras sin ocultar errores.
         *
         * @method call
         * @param {Any} path - Parámetro path de la operación.
         * @param {Any} data - Parámetro data de la operación.
         * @param {Any} method - Parámetro method de la operación.
         * @param {Any} headers - Parámetro headers de la operación.
         * @param {Any} pace - Aplica pacing de cuota antes de enviar.
         * @returns {Any} Resultado verificado de la operación.
         */
        """
        if pace:time.sleep(.25)
        req=Request(BASE+'/api/v1'+path,method=method or ('POST' if data is not None else 'GET'),data=None if data is None else json.dumps(data).encode(),headers={'Content-Type':'application/json',**({'Authorization':'Bearer '+self.token} if self.token else {}),**(headers or {})})
        try: response=self.opener.open(req,timeout=12)
        except HTTPError as error: response=error
        raw=response.read()
        return response.status,json.loads(raw) if raw else None,response.headers

    def account(self):
        """/**
         * Crea una identidad de prueba única e inicia su sesión.
         *
         * @method account
         * @returns {Any} Resultado verificado de la operación.
         */
        """
        email='prueba-'+uuid.uuid4().hex+'@example.test'
        credentials={'email':email,'password':'una-clave-segura-para-pruebas'}
        status,_,_=self.call('/auth/registro',credentials)
        assert status==201,status
        status,session,_=self.call('/auth/login',credentials)
        assert status==200,(status,session)
        self.token=session['accessToken']
        return credentials,session

class DurableFlow(unittest.TestCase):
    """/**
     * Agrupa verificaciones del flujo DurableFlow.
     *
     * @method DurableFlow
     * @returns {DurableFlow} Resultado verificado de la operación.
     */
    """
    def test_identity_ownership_immutable_snapshot_and_variants(self):
        """/**
         * Comprueba identity ownership immutable snapshot and variants.
         *
         * @method test_identity_ownership_immutable_snapshot_and_variants
         * @returns {Any} Resultado verificado de la operación.
         */
        """
        first,other=Client(),Client()
        credentials,session=first.account()
        other.account()
        self.assertEqual(session['usuario']['rol'],'USUARIO')
        self.assertNotIn('password',json.dumps(session))
        status,_,_=first.call('/auth/registro',{**credentials,'rol':'ADMINISTRADOR'})
        self.assertEqual(status,422)
        self.assertEqual(first.call('/admin/usuarios')[0],403)
        self.assertEqual(first.call('/admin/usuarios',headers={'X-Actor-Role':'ADMINISTRADOR','X-Actor-ID':session['usuario']['id']})[0],403)
        status,place,_=first.call('/lugares',{'nombre':'Hospital de prueba','ubicacion':'Recurso manual','superficieM2':50})
        self.assertEqual(status,201,place)
        status,config,_=first.call('/configuraciones',{'nombre':'Equipos hospitalarios','lugarId':place['id'],'consumo':INPUT})
        self.assertEqual(status,201,config)
        self.assertEqual(other.call('/configuraciones/'+config['id'])[0],404)
        key=uuid.uuid4().hex
        body={'version':config['version'],'versionLugar':place['version'],'solar':OPTIONS}
        status,evaluation,_=first.call('/configuraciones/'+config['id']+'/evaluaciones',body,headers={'Idempotency-Key':key})
        self.assertEqual(status,201,evaluation)
        status,repeated,_=first.call('/configuraciones/'+config['id']+'/evaluaciones',body,headers={'Idempotency-Key':key})
        self.assertEqual(status,200,repeated)
        self.assertEqual(evaluation['id'],repeated['id'])
        changed={**body,'solar':{**OPTIONS,'coberturaObjetivo':.5}}
        self.assertEqual(first.call('/configuraciones/'+config['id']+'/evaluaciones',changed,headers={'Idempotency-Key':key})[0],409)
        study_id=evaluation['estudioIds'][0]
        deadline=time.monotonic()+25
        study={}
        while time.monotonic()<deadline:
            status,study,_=first.call('/estudios/'+study_id)
            if status==200 and study['estado'] in ('COMPLETADO','SIN_ALTERNATIVA'): break
            time.sleep(.5)
        self.assertEqual(study.get('estado'),'COMPLETADO',study)
        self.assertIsNotNone(study['resultado']['cotizacion'])
        self.assertEqual(other.call('/estudios/'+study_id)[0],404)
        self.assertEqual(other.call('/evaluaciones/'+evaluation['id'])[0],404)
        status,updated,_=first.call('/configuraciones/'+config['id'],{'nombre':'Editado','lugarId':place['id'],'version':config['version'],'consumo':{**INPUT,'diasPeriodo':31}},'PUT')
        self.assertEqual(status,200,updated)
        self.assertEqual(updated['version'],2)
        status,old,_=first.call('/evaluaciones/'+evaluation['id'])
        self.assertEqual(old['snapshot']['consumo']['diasPeriodo'],30)
        self.assertEqual(first.call('/configuraciones/'+config['id']+'/evaluaciones',body,headers={'Idempotency-Key':uuid.uuid4().hex})[0],409)
        self.assertEqual(first.call('/configuraciones/'+config['id']+'/duplicar',{})[0],201)
        status,new_study,_=first.call('/evaluaciones/'+evaluation['id']+'/estudios',{**OPTIONS,'coberturaObjetivo':.5},headers={'Idempotency-Key':uuid.uuid4().hex})
        self.assertEqual(status,202,new_study)
        self.assertNotEqual(new_study['estudioId'],study_id)
        prior_cookie=next(c.value for c in first.jar if c.name=='renewable_refresh')
        status,refreshed,_=first.call('/auth/refresh',{})
        self.assertEqual(status,200,refreshed)
        self.assertNotEqual(prior_cookie,next(c.value for c in first.jar if c.name=='renewable_refresh'))
        self.assertEqual(first.call('/auth/logout',{})[0],204)
        self.assertEqual(first.call('/auth/refresh',{})[0],401)
        self.assertEqual(first.call('/perfil')[0],401)

    def test_concurrent_confirmation_has_one_effect_and_place_version_is_frozen(self):
        """/**
         * Comprueba concurrent confirmation has one effect and place version is frozen.
         *
         * @method test_concurrent_confirmation_has_one_effect_and_place_version_is_frozen
         * @returns {Any} Resultado verificado de la operación.
         */
        """
        user=Client();user.account()
        _,place,_=user.call('/lugares',{'nombre':'Concurrencia','ubicacion':'Manual','superficieM2':10})
        _,config,_=user.call('/configuraciones',{'nombre':'Idempotencia','lugarId':place['id'],'consumo':INPUT})
        key=uuid.uuid4().hex;body={'version':1,'versionLugar':1,'solar':OPTIONS}
        def submit(_):
            """/**
             * Confirma concurrentemente la misma entrada para detectar duplicación del efecto.
             *
             * @method submit
             * @param {Any} _ - Parámetro _ de la operación.
             * @returns {Any} Resultado verificado de la operación.
             */
            """
            client=Client();client.token=user.token
            return client.call('/configuraciones/'+config['id']+'/evaluaciones',body,headers={'Idempotency-Key':key})
        with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool: results=list(pool.map(submit,range(2)))
        self.assertTrue(all(r[0] in (200,201) for r in results),results)
        self.assertEqual(results[0][1]['id'],results[1][1]['id'])
        self.assertEqual(len(results[0][1]['estudioIds']),1)
        _,edited,_=user.call('/lugares/'+place['id'],{**place,'superficieM2':.1},'PUT')
        status,new,_=user.call('/configuraciones/'+config['id']+'/evaluaciones',{**body,'versionLugar':edited['version']},headers={'Idempotency-Key':uuid.uuid4().hex})
        self.assertEqual(status,201,new)
        self.assertNotEqual(new['id'],results[0][1]['id'])
        self.assertEqual(new['snapshot']['lugar']['superficieM2'],.1)

    @unittest.skipUnless(os.getenv('ADMIN_PASSWORD'),'Credencial administrativa local requerida')
    def test_admin_index_cache_version_and_deactivation(self):
        """/**
         * Comprueba admin index cache version and deactivation.
         *
         * @method test_admin_index_cache_version_and_deactivation
         * @returns {Any} Resultado verificado de la operación.
         */
        """
        admin=Client();status,session,_=admin.call('/auth/login',{'email':'admin@renewable.local','password':os.environ['ADMIN_PASSWORD']})
        self.assertEqual(status,200,session);admin.token=session['accessToken']
        identifier='verif'+uuid.uuid4().hex[:16]
        item={'id':identifier,'nombre':'Ensayo indice '+identifier,'categoria':'Prueba','modo':'POTENCIA','potenciaW':321,'energiaPorCicloKWh':0,'factorFuncionamiento':1,'fuente':'Ensayo de integración automatizado','activo':True}
        started=time.monotonic();status,created,_=admin.call('/admin/electrodomesticos',item)
        self.assertEqual(status,201,created)
        self.addCleanup(admin.call,'/admin/electrodomesticos/'+identifier,{**item,'version':1,'activo':False},'PUT')
        query='/publico/electrodomesticos?q='+identifier+'&potenciaMin=300&potenciaMax=400&orden=-potencia'
        deadline=time.monotonic()+8
        while True:
            status,page,headers=admin.call(query)
            if page.get('total')==1:break
            if time.monotonic()>deadline:self.fail(str(page))
            time.sleep(.5)
        self.assertLess(time.monotonic()-started,5,'Indexación sana supera 5 s')
        self.assertEqual(admin.call(query)[2]['X-Catalog-Source'],'cache')
        status,updated,_=admin.call('/admin/electrodomesticos/'+identifier,{**item,'version':1,'activo':False},'PUT')
        self.assertEqual(status,200,updated)
        self.assertEqual(admin.call('/admin/electrodomesticos/'+identifier,{**item,'version':1},'PUT')[0],409)
        deadline=time.monotonic()+8
        while True:
            status,page,_=admin.call(query)
            if page['total']==0:break
            if time.monotonic()>deadline:self.fail('Desactivación no indexada')
            time.sleep(.5)
        self.assertEqual(admin.call('/admin/catalogo/reindexar',{})[0],200)
        self.assertEqual(admin.call('/publico/electrodomesticos?modo=CICLOS&potenciaMin=1')[0],422)

if __name__=='__main__':unittest.main(verbosity=2)
