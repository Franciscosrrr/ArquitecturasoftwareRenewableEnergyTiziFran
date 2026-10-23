"""Pruebas contra servicios Go y MySQL reales, sin usar el mock como servidor."""
import json
import os
import sys
import time
import unittest
from decimal import Decimal
from pathlib import Path
from urllib.request import Request, urlopen
from urllib.error import HTTPError

ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT))
from mock.server import validate

BASE = os.getenv('BASE_URL', 'http://localhost:8080')
EXAMPLE = json.loads((ROOT / 'docs/contracts/v1/examples/solicitud-valida.json').read_text())
DOC = json.loads((ROOT / 'docs/contracts/v1/openapi.json').read_text())
CATALOG_REQUEST = {'diasPeriodo': 30, 'equipos': [
    {'catalogoId': 'iluminacion', 'cantidad': 4, 'horasPorDia': 5, 'diasUso': 30, 'factorFuncionamiento': 1},
    {'catalogoId': 'heladera', 'cantidad': 1, 'horasPorDia': 24, 'diasUso': 30, 'factorFuncionamiento': 0.3},
    {'catalogoId': 'lavarropas', 'cantidad': 1, 'ciclosPeriodo': 12},
]}

def call(path, body=None, headers=None, raw=None):
    headers = {'Content-Type': 'application/json', **(headers or {})}
    data = raw if raw is not None else json.dumps(body).encode() if body is not None else None
    req = Request(BASE + path, data=data, headers=headers)
    try:
        response = urlopen(req, timeout=12)
    except HTTPError as e:
        response = e
    payload = response.read()
    try:
        value = json.loads(payload, parse_float=Decimal)
    except (ValueError, UnicodeError):
        value = payload.decode()
    return response.status, value, response.headers

class RealServices(unittest.TestCase):
    def setUp(self):
        # Rate policy is part of the deployment; do not disable it just for tests.
        time.sleep(1.1)

    def test_m2m_matches_contract(self):
        status, result, headers = call('/api/v1/estimaciones-consumo', EXAMPLE, {'X-API-Key': os.environ['M2M_API_KEY'], 'X-Request-Id': 'entrega2-m2m'})
        self.assertEqual(status, 200, result)
        validate(result, DOC['components']['schemas']['ResultadoEstimacion'], DOC)
        self.assertEqual(result['totalPeriodoKWh'], Decimal('47.4'))
        self.assertEqual(result['promedioDiarioKWh'], Decimal('1.58'))
        self.assertEqual(headers['X-Request-Id'], 'entrega2-m2m')
        self.assertRegex(result['traceId'], r'^[0-9a-f]{32}$')

    def test_catalog_flow_uses_mysql(self):
        status, result, _ = call('/api/v1/publico/consumo-catalogo', CATALOG_REQUEST, {'X-Request-Id': 'entrega2-catalogo'})
        self.assertEqual(status, 200, result)
        validate(result, DOC['components']['schemas']['ResultadoEstimacion'], DOC)
        self.assertEqual(result['totalPeriodoKWh'], Decimal('47.4'))

    def test_guest_requires_no_account(self):
        status, result, _ = call('/api/v1/publico/consumo', EXAMPLE)
        self.assertEqual(status, 200, result)
        self.assertNotIn('evaluacionId', result)
        self.assertEqual(result['totalPeriodoKWh'], Decimal('47.4'))

    def test_authentication(self):
        for headers, expected in [({}, 401), ({'X-API-Key': 'incorrecta'}, 401), ({'X-API-Key': os.environ['M2M_DENIED_KEY']}, 403)]:
            status, result, _ = call('/api/v1/estimaciones-consumo', EXAMPLE, headers)
            self.assertEqual(status, expected, result)
            validate(result, DOC['components']['schemas']['Error'], DOC)

    def test_catalog_read_search_pagination(self):
        status, result, _ = call('/api/v1/publico/electrodomesticos?tamano=2&pagina=1&orden=nombre')
        self.assertEqual(status, 200, result)
        self.assertEqual(len(result['items']), 2)
        self.assertGreaterEqual(result['total'], 6)
        status, result, _ = call('/api/v1/publico/electrodomesticos?q=Heladera&modo=POTENCIA')
        self.assertEqual(status, 200, result)
        self.assertEqual(result['items'][0]['id'], 'heladera')
        self.assertEqual(call('/api/v1/publico/electrodomesticos?tamano=101')[0], 422)

    def test_invalid_days_and_client_power(self):
        body = json.loads(json.dumps(EXAMPLE))
        body['equipos'][0]['diasUso'] = 31
        status, result, _ = call('/api/v1/publico/consumo', body)
        self.assertEqual(status, 422, result)
        self.assertEqual(result['detalles'][0]['campo'], 'equipos[0].diasUso')
        body = json.loads(json.dumps(CATALOG_REQUEST))
        body['equipos'][0]['potenciaW'] = 1
        self.assertEqual(call('/api/v1/publico/consumo-catalogo', body)[0], 422)

    def test_json_and_size_limits(self):
        self.assertEqual(call('/api/v1/publico/consumo', raw=b'{')[0], 400)
        self.assertEqual(call('/api/v1/publico/consumo', EXAMPLE, {'Content-Type': 'text/plain'})[0], 415)
        self.assertEqual(call('/api/v1/publico/consumo', raw=b' ' * 262145)[0], 413)

    def test_private_routes_and_health(self):
        self.assertEqual(call('/api/v1/configuraciones')[0], 401)
        self.assertEqual(call('/health/ready')[0], 401)
        status, result, _ = call('/health/ready', headers={'Authorization': 'Bearer ' + os.environ['HEALTHCHECK_TOKEN']})
        self.assertEqual(status, 200, result)
        self.assertEqual(result['estado'], 'disponible')

    def test_web_and_published_contract(self):
        status, result, _ = call('/')
        self.assertEqual(status, 200)
        self.assertIn('<html lang="es">', result)
        status, result, _ = call('/openapi.json')
        self.assertEqual(status, 200)
        self.assertEqual(result['info']['version'], '1.0.1')

if __name__ == '__main__':
    unittest.main()
