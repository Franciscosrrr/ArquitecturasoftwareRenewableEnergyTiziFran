"""Contrato de la comparación solar temporal y privacidad sin cuenta."""
import json,time,unittest
from pathlib import Path
from test_durable import Client
from mock.server import validate

DOC=json.loads((Path(__file__).resolve().parents[1]/'docs/contracts/guest-v1/openapi.json').read_text(encoding='utf-8-sig'))
class GuestSolar(unittest.TestCase):
    """/**
     * Verifica el contrato y la ausencia de identidad durable en la comparación invitada.
     *
     * @method GuestSolar
     * @returns {GuestSolar} Casos de aceptación sin sesión.
     */
    """
    def test_comparison_source_surface_zero_and_contract(self):
        """/**
         * Verifica fuente manual, falta de superficie y respuesta compatible sin id de estudio.
         *
         * @method test_comparison_source_surface_zero_and_contract
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        client=Client()
        body={'promedioDiarioKWh':3,'coberturaObjetivo':1,'rendimientoGlobal':.8,'superficieUtilM2':0,'recursoSolar':{'tipo':'MANUAL','hsp':4,'fuente':'Supuesto explícito de prueba'}}
        status,result,_=client.call('/publico/recomendaciones-solares',body)
        self.assertEqual(status,200,result)
        validate(result,DOC['components']['schemas']['ResultadoSolarTemporal'],DOC)
        self.assertEqual(result['estado'],'SIN_ALTERNATIVA')
        self.assertTrue(all(a['cantidad']==0 and not a['cumpleObjetivo'] for a in result['alternativas']))
        self.assertNotIn('estudioId',result)
        time.sleep(6.1)
        body.pop('superficieUtilM2');body['promedioDiarioKWh']=0
        status,result,_=client.call('/publico/recomendaciones-solares',body)
        self.assertEqual(status,200,result)
        validate(result,DOC['components']['schemas']['ResultadoSolarTemporal'],DOC)
        self.assertEqual(result['estado'],'COMPLETADO')
        self.assertTrue(all(a['cantidad']==0 for a in result['alternativas']))
