"""Verifica organización, eliminación propia y búsqueda solar contra dependencias reales."""
import os,time,unittest,uuid
from urllib.parse import urlencode
from test_durable import Client,INPUT,OPTIONS

class CustomerFlows(unittest.TestCase):
    """/**
     * Comprueba acciones de cliente sin perder trazabilidad histórica.
     *
     * @method CustomerFlows
     * @returns {CustomerFlows} Casos HTTP con recursos propios de cada prueba.
     */
    """
    def test_archive_delete_and_immutable_history(self):
        """/**
         * Archiva sin versionar y elimina recursos propios conservando sus resultados previos.
         *
         * @method test_archive_delete_and_immutable_history
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        user,other=Client(),Client();user.account();other.account()
        _,place,_=user.call('/lugares',{'nombre':'Lugar de ciclo de vida','ubicacion':'Manual','superficieM2':40})
        _,config,_=user.call('/configuraciones',{'nombre':'Consumo eliminable','lugarId':place['id'],'consumo':INPUT})
        _,evaluation,_=user.call('/configuraciones/'+config['id']+'/evaluaciones',{'version':1,'versionLugar':1,'solar':OPTIONS},headers={'Idempotency-Key':uuid.uuid4().hex})
        self.assertIn('id',evaluation,evaluation)
        self.assertEqual(user.call('/lugares/'+place['id'],{'version':1},'DELETE')[0],409)
        for archived in [True,False,False]:
            self.assertEqual(user.call('/configuraciones/'+config['id']+'/archivo',{'version':1,'archivado':archived},'PATCH')[0],204)
            status,current,_=user.call('/configuraciones/'+config['id']);self.assertEqual(status,200)
            self.assertEqual(current['version'],1);self.assertEqual(current['snapshot'],config['snapshot']);self.assertEqual(current['archivada'],archived)
            self.assertEqual(user.call('/lugares/'+place['id']+'/archivo',{'version':1,'archivado':archived},'PATCH')[0],204)
            _,places,_=user.call('/lugares');self.assertEqual(next(p for p in places if p['id']==place['id'])['version'],1)
        self.assertEqual(other.call('/configuraciones/'+config['id'],{'version':1},'DELETE')[0],404)
        self.assertEqual(other.call('/lugares/'+place['id']+'/archivo',{'version':1,'archivado':True},'PATCH')[0],404)
        self.assertEqual(user.call('/configuraciones/'+config['id'],{'version':2},'DELETE')[0],409)
        self.assertEqual(user.call('/configuraciones/'+config['id'],{'version':1},'DELETE')[0],204)
        self.assertEqual(user.call('/configuraciones/'+config['id'])[0],404)
        self.assertEqual(user.call('/configuraciones/'+config['id']+'/duplicar',{})[0],404)
        self.assertEqual(user.call('/configuraciones/'+config['id']+'/archivo',{'version':1,'archivado':False},'PATCH')[0],404)
        _,configs,_=user.call('/configuraciones');self.assertNotIn(config['id'],[c['id'] for c in configs])
        self.assertEqual(user.call('/lugares/'+place['id'],{'version':1},'DELETE')[0],204)
        _,places,_=user.call('/lugares');self.assertNotIn(place['id'],[p['id'] for p in places])
        self.assertEqual(user.call('/configuraciones',{'nombre':'No permite reutilizar lugar eliminado','lugarId':place['id'],'consumo':INPUT})[0],404)
        status,history,_=user.call('/evaluaciones/'+evaluation['id']);self.assertEqual(status,200);self.assertEqual(history['snapshot'],evaluation['snapshot'])
        self.assertEqual(other.call('/evaluaciones/'+evaluation['id'])[0],404)

    @unittest.skipUnless(os.getenv('ADMIN_PASSWORD'),'Credencial administrativa local requerida')
    def test_solar_search_filters_and_latest_version(self):
        """/**
         * Comprueba filtros solares, sincronización tras cambios y exclusión de fichas inactivas.
         *
         * @method test_solar_search_filters_and_latest_version
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        admin=Client();status,session,_=admin.call('/auth/login',{'email':'admin@renewable.local','password':os.environ['ADMIN_PASSWORD']});self.assertEqual(status,200);admin.token=session['accessToken']
        identifier='solar'+uuid.uuid4().hex[:16]
        panel={'id':identifier,'nombre':'Modelo '+identifier,'potenciaW':520,'superficieEfectivaMilliM2':1800,'precioCentavos':12345,'moneda':'ARS','fuente':'Ficha de prueba de búsqueda','activo':True}
        status,created,_=admin.call('/admin/paneles',panel);self.assertEqual(status,201,created)
        def cleanup():
            """/**
             * Desactiva únicamente la ficha creada por este caso incluso si una aserción falla.
             *
             * @method cleanup
             * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
             */
            """
            status,items,_=admin.call('/admin/paneles')
            if status==200:
                current=next((p for p in items if p['id']==identifier),None)
                if current and current['activo']:admin.call('/admin/paneles/'+identifier,{**current,'activo':False},'PUT')
        self.addCleanup(cleanup)
        query='/publico/paneles/busqueda?'+urlencode({'q':identifier,'moneda':'ARS','potenciaMin':500,'potenciaMax':550,'superficieMax':2,'orden':'potencia','tamano':1})
        started=time.monotonic();deadline=started+8
        while True:
            status,page,headers=admin.call(query);self.assertEqual(status,200,page)
            if page['total']==1 and headers.get('X-Panel-Source')=='meilisearch':break
            if time.monotonic()>deadline:self.fail('No se sincronizó el panel: '+str(page))
            time.sleep(.6)
        self.assertLess(time.monotonic()-started,5)
        self.assertEqual(page['items'][0]['id'],identifier);self.assertEqual(page['items'][0]['nombre'],panel['nombre'])
        for version in [1,2]:
            status,updated,_=admin.call('/admin/paneles/'+identifier,{**panel,'nombre':'Edición '+str(version+1)+' '+identifier,'version':version},'PUT');self.assertEqual(status,200,updated)
        deadline=time.monotonic()+8
        while True:
            _,page,_=admin.call(query)
            if page['items'] and page['items'][0]['version']==3:break
            if time.monotonic()>deadline:self.fail('Una versión vieja reemplazó el índice')
            time.sleep(.6)
        self.assertEqual(page['items'][0]['nombre'],'Edición 3 '+identifier)
        self.assertEqual(admin.call('/publico/paneles/busqueda?precioMax=100')[0],422)
        self.assertEqual(admin.call('/publico/paneles/busqueda?potenciaMin=600&potenciaMax=500')[0],422)
        status,_,_=admin.call('/admin/paneles/'+identifier,{**panel,'version':3,'activo':False},'PUT');self.assertEqual(status,200)
        deadline=time.monotonic()+8
        while True:
            _,page,_=admin.call(query)
            if page['total']==0:break
            if time.monotonic()>deadline:self.fail('El índice conserva una ficha inactiva')
            time.sleep(.6)
        self.assertEqual(admin.call('/admin/paneles/reindexar',{})[0],202)

if __name__=='__main__':unittest.main(verbosity=2)
