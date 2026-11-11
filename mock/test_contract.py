import copy
import http.client
import json
import threading
import unittest
from decimal import Decimal
from server import ROOT, DOCS, Handler, Limiter, create_server, validate, M2M, CONSUMO, SOLAR


# Credencial ficticia exclusiva de pruebas, no un secreto de despliegue.
TEST_HEALTH_TOKEN = 'credencial-ficticia-solo-para-pruebas-123456'


class ContractTest(unittest.TestCase):
    """/**
     * Agrupa verificaciones del flujo ContractTest.
     *
     * @method ContractTest
     * @returns {ContractTest} Tipo que agrupa el estado y las verificaciones descritas.
     */
    """
    @classmethod
    def setUpClass(cls):
        """/**
         * Valida y ejecuta setUpClass según el contrato de prueba.
         *
         * @method setUpClass
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        cls.server = create_server(port=0, healthcheck_token=TEST_HEALTH_TOKEN)
        cls.thread = threading.Thread(target=cls.server.serve_forever, daemon=True)
        cls.thread.start()
        cls.valid = json.loads((ROOT / 'docs/contracts/v1/examples/solicitud-valida.json').read_text(encoding='utf-8'))
        cls.sun = json.loads((ROOT / 'docs/contracts/guest-v1/examples/solicitud-solar-manual.json').read_text(encoding='utf-8'))

    @classmethod
    def tearDownClass(cls):
        """/**
         * Valida y ejecuta tearDownClass según el contrato de prueba.
         *
         * @method tearDownClass
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        cls.server.shutdown()
        cls.server.server_close()
        cls.thread.join()

    def setUp(self):
        """/**
         * Espera la reposición de cuota antes de una prueba independiente.
         *
         * @method setUp
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        Handler.limiter = Limiter()

    def request(self, data=None, path=M2M, key='mock-consumidor', extra=None, raw=None):
        """/**
         * Valida y ejecuta request según el contrato de prueba.
         *
         * @method request
         * @param {Any} data - Datos estructurados que se validan o registran sin credenciales.
         * @param {str} path - Ruta HTTP o JSON que identifica el recurso o campo de la operación.
         * @param {str|None} key - Identidad o credencial utilizada por esta operación de prueba.
         * @param {dict|None} extra - Cabeceras o datos adicionales de la operación, según el helper.
         * @param {bytes|None} raw - Cuerpo HTTP original para ensayar también JSON o framing inválidos.
         * @returns {tuple} Estado HTTP, JSON y cabeceras de la operación probada.
         */
        """
        headers = {'Content-Type': 'application/json', 'X-Request-Id': 'prueba-001'}
        if key is not None: headers['X-API-Key'] = key
        headers.update(extra or {})
        conn = http.client.HTTPConnection(*self.server.server_address, timeout=5)
        conn.request('POST', path, body=raw if raw is not None else json.dumps(data if data is not None else self.valid), headers=headers)
        response = conn.getresponse()
        status, response_headers = response.status, dict(response.getheaders())
        result = json.loads(response.read())
        conn.close()
        doc = DOCS['v1' if path == M2M else 'guest-v1']
        schema = doc['paths'][path]['post']['responses'][str(status)]['content']['application/json']['schema']
        validate(result, schema, doc)
        self.assertEqual(response_headers['X-Mock'], 'entrega-1')
        return status, result, response_headers

    def health_request(self, auth=None, method='GET', path='/health/live'):
        """/**
         * Valida y ejecuta health request según el contrato de prueba.
         *
         * @method health_request
         * @param {str|None} auth - Valor completo de Authorization, o None para omitirlo.
         * @param {str} method - Método HTTP que se debe enviar.
         * @param {str} path - Ruta HTTP o JSON que identifica el recurso o campo de la operación.
         * @returns {tuple} Estado HTTP, cabeceras y cuerpo original de la consulta de salud.
         */
        """
        conn = http.client.HTTPConnection(*self.server.server_address, timeout=5)
        headers = {} if auth is None else {'Authorization': auth}
        conn.request(method, path, headers=headers)
        response = conn.getresponse()
        result = (response.status, dict(response.getheaders()), response.read())
        conn.close()
        return result

    def test_health_rejects_missing_or_wrong_credentials(self):
        """/**
         * Comprueba health rejects missing or wrong credentials.
         *
         * @method test_health_rejects_missing_or_wrong_credentials
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        for method in ('GET', 'HEAD'):
            for auth in (None, 'Bearer incorrecta', 'Bearer mock-consumidor', 'Basic ' + TEST_HEALTH_TOKEN):
                with self.subTest(method=method, auth=auth):
                    code, headers, body = self.health_request(auth, method)
                    self.assertEqual(code, 401)
                    self.assertEqual(headers['Cache-Control'], 'no-store')
                    self.assertIn('WWW-Authenticate', headers)
                    self.assertNotIn(b'estado', body)
                    self.assertNotIn(TEST_HEALTH_TOKEN.encode(), body)
        for path in ('/health/live?token=' + TEST_HEALTH_TOKEN, '/health/ready'):
            self.assertEqual(self.health_request(path=path)[0], 401)

    def test_health_with_operational_credential(self):
        """/**
         * Comprueba health with operational credential.
         *
         * @method test_health_with_operational_credential
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        code, headers, body = self.health_request('Bearer ' + TEST_HEALTH_TOKEN)
        self.assertEqual(code, 200)
        self.assertEqual(json.loads(body), {'estado': 'disponible'})
        self.assertEqual(headers['Cache-Control'], 'no-store')
        code, _, body = self.health_request('Bearer ' + TEST_HEALTH_TOKEN, 'HEAD')
        self.assertEqual(code, 200)
        self.assertEqual(body, b'')

    def test_server_requires_operational_credential(self):
        """/**
         * Comprueba server requires operational credential.
         *
         * @method test_server_requires_operational_credential
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        for token in ('', 'corta'):
            with self.subTest(token=token), self.assertRaises(ValueError):
                create_server(port=0, healthcheck_token=token)

    def test_reference_calculation(self):
        """/**
         * Comprueba reference calculation.
         *
         * @method test_reference_calculation
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        code, result, _ = self.request()
        self.assertEqual(code, 200)
        expected = json.loads((ROOT/'docs/contracts/v1/examples/respuesta-200.json').read_text(encoding='utf-8'))
        expected['traceId'] = 'prueba-001'
        self.assertEqual(result, expected)

    def test_guest_same_calculation_without_credentials(self):
        """/**
         * Comprueba guest same calculation without credentials.
         *
         * @method test_guest_same_calculation_without_credentials
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        self.assertEqual(self.request(path=CONSUMO, key=None)[1], self.request()[1])

    def test_authentication(self):
        """/**
         * Comprueba authentication.
         *
         * @method test_authentication
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        for key in (None, 'incorrecta'):
            with self.subTest(key=key): self.assertEqual(self.request(key=key)[0], 401)
        self.assertEqual(self.request(key='mock-sin-permiso')[0], 403)

    def test_business_rules(self):
        """/**
         * Comprueba business rules.
         *
         * @method test_business_rules
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        cases = []
        data = copy.deepcopy(self.valid); data['equipos'][0]['diasUso'] = 31; cases.append(data)
        data = copy.deepcopy(self.valid); data['equipos'][1]['id'] = data['equipos'][0]['id']; cases.append(data)
        data = copy.deepcopy(self.valid); data['equipos'][0]['factorFuncionamiento'] = 0.1234567; cases.append(data)
        for data in cases:
            with self.subTest(data=data): self.assertEqual(self.request(data)[0], 422)

    def test_schema_rejection(self):
        """/**
         * Comprueba schema rejection.
         *
         * @method test_schema_rejection
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        for change in ({'cantidad': 0}, {'cantidad': True}, {'cantidad': 1.5}, {'modo':'OTRO'}, {'potenciaW':None}, {'horasPorDia':25}, {'extra':1}, {'energiaPorCicloKWh':1}):
            data = copy.deepcopy(self.valid); data['equipos'][0].update(change)
            with self.subTest(change=change): self.assertEqual(self.request(data)[0], 422)

    def test_framing(self):
        """/**
         * Comprueba framing.
         *
         * @method test_framing
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        self.assertEqual(self.request(raw='{')[0], 400)
        self.assertEqual(self.request(raw='{"a":NaN}')[0], 400)
        self.assertEqual(self.request(raw=b'\xff')[0], 400)
        self.assertEqual(self.request(extra={'Content-Type':'text/plain'})[0], 415)
        self.assertEqual(self.request(raw=' ' * 262145)[0], 413)

    def test_period_and_array_bounds(self):
        """/**
         * Comprueba period and array bounds.
         *
         * @method test_period_and_array_bounds
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        for d in ({'diasPeriodo':0, 'equipos':self.valid['equipos']}, {'diasPeriodo':30,'equipos':[]}, {'diasPeriodo':30,'equipos':self.valid['equipos']*34}):
            with self.subTest(): self.assertEqual(self.request(d)[0], 422)

    def test_rounding_not_per_item(self):
        """/**
         * Comprueba rounding not per item.
         *
         * @method test_rounding_not_per_item
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        data = {'diasPeriodo':3,'equipos':[{'id':str(i),'modo':'POTENCIA','cantidad':1,'potenciaW':0.0005,'horasPorDia':1,'diasUso':1,'factorFuncionamiento':1} for i in range(3)]}
        code, result, _ = self.request(data)
        self.assertEqual(code,200)
        self.assertEqual(result['desglose'][0]['energiaPeriodoKWh'],0.000001)
        self.assertEqual(result['totalPeriodoKWh'],0.000002)
        self.assertEqual(result['promedioDiarioKWh'],0.000001)

    def test_forced_errors(self):
        """/**
         * Comprueba forced errors.
         *
         * @method test_forced_errors
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        for code in (429,500,503):
            with self.subTest(code=code):
                status, _, headers = self.request(extra={'X-Mock-Scenario':str(code)})
                self.assertEqual(status,code)
                if code != 500: self.assertIn('Retry-After',headers)

    def test_real_rate_limit(self):
        """/**
         * Comprueba real rate limit.
         *
         * @method test_real_rate_limit
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        codes = [self.request()[0] for _ in range(11)]
        self.assertEqual(codes[:10],[200]*10)
        self.assertEqual(codes[10],429)

    def test_solar_reference(self):
        """/**
         * Comprueba solar reference.
         *
         * @method test_solar_reference
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        code, result, _ = self.request(self.sun, SOLAR, key=None)
        self.assertEqual(code,200)
        expected = json.loads((ROOT/'docs/contracts/guest-v1/examples/respuesta-solar-manual.json').read_text(encoding='utf-8'))
        expected['traceId'] = 'prueba-001'
        self.assertEqual(result,expected)
        self.assertNotIn('estudioId',result)

    def test_solar_valid_extreme_inputs_conform_to_response_contract(self):
        """/**
         * Comprueba solar valid extreme inputs conform to response contract.
         *
         * @method test_solar_valid_extreme_inputs_conform_to_response_contract
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        data = dict(self.sun, promedioDiarioKWh=1000000000000, coberturaObjetivo=1, rendimientoGlobal=0.000001,
                    recursoSolar={'tipo': 'MANUAL', 'hsp': 0.000001, 'fuente': 'Caso límite de contrato'})
        data.pop('superficieUtilM2', None)
        code, result, _ = self.request(data, SOLAR, key=None)
        self.assertEqual(code, 200)
        self.assertEqual(result['alternativas'][0]['cantidad'], 2000000000000000000000000)
        self.assertTrue(result['alternativas'][0]['cumpleObjetivo'])

    def test_validation_identifies_observed_field(self):
        """/**
         * Comprueba validation identifies observed field.
         *
         * @method test_validation_identifies_observed_field
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        data = copy.deepcopy(self.valid)
        data['equipos'][0]['diasUso'] = 31
        code, result, _ = self.request(data)
        self.assertEqual(code, 422)
        self.assertEqual(result['detalles'][0]['campo'], 'equipos[0].diasUso')
        data = copy.deepcopy(self.valid)
        data['equipos'][0]['factorFuncionamiento'] = 0.1234567
        code, result, _ = self.request(data)
        self.assertEqual(code, 422)
        self.assertEqual(result['detalles'][0]['campo'], 'equipos[0].factorFuncionamiento')

    def test_solar_zero(self):
        """/**
         * Comprueba solar zero.
         *
         * @method test_solar_zero
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        data = dict(self.sun, promedioDiarioKWh=0)
        code, result, _ = self.request(data, SOLAR, key=None)
        self.assertEqual(code,200)
        self.assertEqual(result['alternativas'][0]['cantidad'],0)

    def test_solar_insufficient_area(self):
        """/**
         * Comprueba solar insufficient area.
         *
         * @method test_solar_insufficient_area
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        data = dict(self.sun, superficieUtilM2=0)
        code, result, _ = self.request(data, SOLAR, key=None)
        self.assertEqual(code,200)
        self.assertEqual(result['estado'],'SIN_ALTERNATIVA')
        self.assertFalse(result['alternativas'][0]['cumpleObjetivo'])

    def test_unassigned_provider(self):
        """/**
         * Comprueba unassigned provider.
         *
         * @method test_unassigned_provider
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        data = dict(self.sun, recursoSolar={'tipo':'UBICACION','latitud':-31,'longitud':-64})
        self.assertEqual(self.request(data,SOLAR,key=None)[0],503)

    def test_excessive_precision_extreme_exponent(self):
        """/**
         * Comprueba excessive precision extreme exponent.
         *
         * @method test_excessive_precision_extreme_exponent
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        raw = json.dumps(self.valid).replace('0.3','1e-100000')
        self.assertEqual(self.request(raw=raw)[0],422)

    def test_contract_refs_and_examples(self):
        """/**
         * Comprueba contract refs and examples.
         *
         * @method test_contract_refs_and_examples
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        for doc in DOCS.values():
            def walk(node):
                """/**
                 * Valida y ejecuta walk según el contrato de prueba.
                 *
                 * @method walk
                 * @param {Any} node - Nodo del contrato que se recorre para comprobar referencias.
                 * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
                 */
                """
                if isinstance(node,dict):
                    if '$ref' in node:
                        target = doc
                        for k in node['$ref'][2:].split('/'): target = target[k]
                    if 'schema' in node:
                        if 'example' in node: validate(node['example'],node['schema'],doc)
                        for example in node.get('examples',{}).values():
                            if 'value' in example: validate(example['value'],node['schema'],doc)
                    for value in node.values(): walk(value)
                elif isinstance(node,list):
                    for value in node: walk(value)
            walk(doc)


if __name__ == '__main__': unittest.main()
