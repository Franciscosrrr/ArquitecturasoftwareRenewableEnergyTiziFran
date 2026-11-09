"""Roles efectivos y catálogo Mongo mediante las APIs reales del sistema."""
import os,time,unittest,uuid
from test_durable import Client

@unittest.skipUnless(os.getenv('ADMIN_PASSWORD'),'Credencial local requerida')
class Administration(unittest.TestCase):
    """/**
     * Verifica controles administrativos y persistencia de paneles.
     *
     * @method Administration
     * @returns {Administration} Pruebas de cambios de rol y versión en Mongo.
     */
    """
    def test_role_change_is_immediate_and_last_admin_is_preserved(self):
        """/**
         * Verifica roles vigentes sin reemitir JWT y conservación del último administrador.
         *
         * @method test_role_change_is_immediate_and_last_admin_is_preserved
         * @returns {None} Falla si un token conserva permisos revocados o desaparece el último administrador.
         */
        """
        admin,user=Client(),Client();_,session=user.account()
        status,login,_=admin.call('/auth/login',{'email':'admin@renewable.local','password':os.environ['ADMIN_PASSWORD']})
        self.assertEqual(status,200,login);admin.token=login['accessToken']
        uid=session['usuario']['id']
        try:
            self.assertEqual(admin.call('/admin/usuarios/'+uid+'/rol',{'rol':'ADMINISTRADOR'},'PUT')[0],204)
            self.assertEqual(user.call('/admin/usuarios')[0],200)
        finally:admin.call('/admin/usuarios/'+uid+'/rol',{'rol':'USUARIO'},'PUT')
        self.assertEqual(user.call('/admin/usuarios')[0],403)
        self.assertEqual(admin.call('/admin/usuarios/'+login['usuario']['id']+'/rol',{'rol':'USUARIO'},'PUT')[0],409)

    def test_panel_cas_and_inactive_visibility(self):
        """/**
         * Comprueba alta y edición Mongo con versión, sin exponer fichas inactivas al invitado.
         *
         * @method test_panel_cas_and_inactive_visibility
         * @returns {None} Falla si una versión obsoleta sobrescribe el catálogo solar.
         */
        """
        admin=Client();status,session,_=admin.call('/auth/login',{'email':'admin@renewable.local','password':os.environ['ADMIN_PASSWORD']})
        self.assertEqual(status,200,session);admin.token=session['accessToken']
        panel={'id':'verif-'+uuid.uuid4().hex[:16],'nombre':'Panel de ensayo inactivo','fuente':'Fixture de integración','potenciaW':450,'superficieEfectivaMilliM2':2500,'precioCentavos':0,'moneda':'ARS','activo':False}
        status,created,_=admin.call('/admin/paneles',panel);self.assertEqual(status,201,created)
        status,updated,_=admin.call('/admin/paneles/'+panel['id'],{**panel,'version':1,'nombre':'Panel actualizado'},'PUT');self.assertEqual(status,200,updated)
        self.assertEqual(updated['version'],2)
        self.assertEqual(admin.call('/admin/paneles/'+panel['id'],{**panel,'version':1},'PUT')[0],409)
        time.sleep(1.1)
        status,visible,_=admin.call('/publico/paneles');self.assertEqual(status,200,visible)
        self.assertNotIn(panel['id'],[p['id'] for p in visible])
