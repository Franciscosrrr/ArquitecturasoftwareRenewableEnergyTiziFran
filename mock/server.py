"""Mock local de contrato; sin BD, proveedores ni persistencia de negocio."""
import hashlib
import hmac
import json
import os
import re
import threading
import time
import uuid
from decimal import Decimal, ROUND_HALF_UP, ROUND_CEILING, ROUND_FLOOR, localcontext
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
DOCS = {v: json.loads((ROOT / 'docs/contracts' / v / 'openapi.json').read_text(encoding='utf-8')) for v in ('v1', 'guest-v1')}
M2M = '/api/v1/estimaciones-consumo'
CONSUMO = '/api/v1/publico/consumo'
SOLAR = '/api/v1/publico/recomendaciones-solares'
ERRORS = {400: 'FORMATO_INVALIDO', 401: 'NO_AUTENTICADO', 403: 'SIN_PERMISO', 413: 'CUERPO_DEMASIADO_GRANDE', 415: 'TIPO_CONTENIDO_NO_SOPORTADO', 422: 'VALIDACION', 429: 'LIMITE_DE_TRAFICO', 500: 'ERROR_INTERNO', 503: 'SERVICIO_NO_DISPONIBLE'}


class ValidationError(ValueError):
    """/**
     * Representa un rechazo del mock con ruta del campo y motivo público.
     *
     * @method ValidationError
     * @returns {ValidationError} Tipo que agrupa el estado y las verificaciones descritas.
     */
    """
    def __init__(self, field, reason):
        """/**
         * Conserva el campo y la causa de la infracción de validación.
         *
         * @method __init__
         * @param {str} field - Ruta del campo que incumple la validación.
         * @param {str} reason - Motivo público de rechazo del campo.
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        self.field = field[2:] if field.startswith('$.') else field
        self.reason = reason
        super().__init__(self.field + ': ' + reason)


def validate(value, schema, doc, path='$'):
    """/**
     * Valida un documento contra el contrato procesable y sus reglas.
     *
     * @method validate
     * @param {Any} value - Valor recibido que se valida o redondea según la operación.
     * @param {dict} schema - Esquema de la operación o variante que debe cumplir el documento.
     * @param {dict} doc - Contrato completo utilizado para resolver sus referencias internas.
     * @param {str} path - Ruta HTTP o JSON que identifica el recurso o campo de la operación.
     * @returns {None} No devuelve un valor; lanza ValidationError cuando el documento no cumple el contrato.
     */
    """
    if '$ref' in schema:
        target = doc
        for part in schema['$ref'][2:].split('/'):
            target = target[part]
        return validate(value, target, doc, path)
    if 'oneOf' in schema:
        matches = 0
        for variant in schema['oneOf']:
            try:
                validate(value, variant, doc, path)
                matches += 1
            except ValueError:
                pass
        if matches != 1:
            raise ValidationError(path, 'variante inválida o campos incompatibles')
        return
    kind = schema.get('type')
    numeric = isinstance(value, (int, float, Decimal)) and not isinstance(value, bool)
    valid = {'object': isinstance(value, dict), 'array': isinstance(value, list), 'string': isinstance(value, str), 'number': numeric, 'integer': numeric and (kind != 'integer' or value == value // 1), 'boolean': isinstance(value, bool)}
    if kind and not valid[kind]:
        raise ValidationError(path, 'tipo inválido')
    if 'enum' in schema and value not in schema['enum']:
        raise ValidationError(path, 'valor no admitido')
    if kind == 'object':
        props = schema.get('properties', {})
        if any(k not in value for k in schema.get('required', [])):
            raise ValidationError(path, 'faltan campos obligatorios')
        if schema.get('additionalProperties') is False and set(value) - set(props):
            raise ValidationError(path, 'campos desconocidos')
        for key, item in value.items():
            if key in props:
                validate(item, props[key], doc, path + '.' + key)
    elif kind == 'array':
        if not schema.get('minItems', 0) <= len(value) <= schema.get('maxItems', float('inf')):
            raise ValidationError(path, 'tamaño inválido')
        for i, item in enumerate(value):
            validate(item, schema['items'], doc, f'{path}[{i}]')
    elif kind == 'string':
        if not schema.get('minLength', 0) <= len(value) <= schema.get('maxLength', float('inf')):
            raise ValidationError(path, 'longitud inválida')
        if 'pattern' in schema and not re.fullmatch(schema['pattern'], value):
            raise ValidationError(path, 'formato inválido')
    elif kind in ('number', 'integer'):
        for key, exclusive, sign in [('minimum', 'exclusiveMinimum', -1), ('maximum', 'exclusiveMaximum', 1)]:
            if key in schema and ((value - schema[key]) * sign > 0 or (schema.get(exclusive) and value == schema[key])):
                raise ValidationError(path, 'fuera de rango')


def precision(value, path='$'):
    """/**
     * Valida y ejecuta precision según el contrato de prueba.
     *
     * @method precision
     * @param {Any} value - Valor recibido que se valida o redondea según la operación.
     * @param {str} path - Ruta HTTP o JSON que identifica el recurso o campo de la operación.
     * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
     */
    """
    if isinstance(value, Decimal) and value != value.quantize(Decimal('0.000001')):
        raise ValidationError(path, 'Se admiten como máximo seis decimales')
    if isinstance(value, dict):
        for key, item in value.items(): precision(item, path + '.' + key)
    if isinstance(value, list):
        for index, item in enumerate(value): precision(item, f'{path}[{index}]')


def rounded(value):
    """/**
     * Valida y ejecuta rounded según el contrato de prueba.
     *
     * @method rounded
     * @param {Any} value - Valor recibido que se valida o redondea según la operación.
     * @returns {float} Salida numérica del mock redondeada a seis decimales, mitad hacia arriba.
     */
    """
    return float(Decimal(value).quantize(Decimal('0.000001'), rounding=ROUND_HALF_UP))


def consumo(data, trace):
    """/**
     * Calcula el consumo ilustrativo por potencia o ciclos y redondea al construir la respuesta.
     *
     * @method consumo
     * @param {Any} data - Datos estructurados que se validan o registran sin credenciales.
     * @param {str} trace - Identificador de correlación que acompaña el error o la respuesta.
     * @returns {dict} Respuesta del mock con desglose, total y promedio diario calculados con Decimal.
     */
    """
    seen, rows, total = set(), [], Decimal(0)
    for index, e in enumerate(data['equipos']):
        if e['id'] in seen: raise ValidationError(f'equipos[{index}].id', 'Identificador repetido')
        seen.add(e['id'])
        if e['modo'] == 'POTENCIA':
            if e['diasUso'] > data['diasPeriodo']: raise ValidationError(f'equipos[{index}].diasUso', 'No puede superar diasPeriodo')
            energy = Decimal(e['potenciaW']) * e['cantidad'] * e['horasPorDia'] * e['diasUso'] * e['factorFuncionamiento'] / 1000
        else:
            energy = Decimal(e['energiaPorCicloKWh']) * e['cantidad'] * e['ciclosPeriodo']
        total += energy
        rows.append({'id': e['id'], 'energiaPeriodoKWh': rounded(energy)})
    return dict(versionAlgoritmo='consumo-v1', diasPeriodo=int(data['diasPeriodo']), desglose=rows, totalPeriodoKWh=rounded(total), promedioDiarioKWh=rounded(total / data['diasPeriodo']), supuestos=['Estimación basada en equipos y hábitos declarados; no es una medición real.', 'El factor de funcionamiento se aplica una sola vez.'], traceId=trace)


def solar(data, trace):
    """/**
     * Dimensiona la alternativa ilustrativa del mock con supuestos manuales, sin contactar un proveedor real.
     *
     * @method solar
     * @param {Any} data - Datos estructurados que se validan o registran sin credenciales.
     * @param {str} trace - Identificador de correlación que acompaña el error o la respuesta.
     * @returns {dict|None} Comparación solar ilustrativa o None cuando no hay fuente por ubicación disponible.
     */
    """
    recurso = data['recursoSolar']
    if recurso['tipo'] != 'MANUAL': return None
    demand = Decimal(data['promedioDiarioKWh'])
    generation = Decimal('0.5') * recurso['hsp'] * data['rendimientoGlobal']
    count = int((demand * data['coberturaObjetivo'] / generation).to_integral_value(rounding=ROUND_CEILING))
    if 'superficieUtilM2' in data:
        count = min(count, int((Decimal(data['superficieUtilM2']) / Decimal('2.5')).to_integral_value(rounding=ROUND_FLOOR)))
    actual = generation * count
    meets = actual >= demand * data['coberturaObjetivo']
    row = dict(panelId='panel-demo-500', modelo='Panel ilustrativo 500 W', versionFicha=1, cantidad=count, generacionDiariaKWh=rounded(actual), coberturaAlcanzada=rounded(actual / demand) if demand else 0, superficieRequeridaM2=rounded(Decimal('2.5') * count), cumpleObjetivo=meets)
    return dict(estado='COMPLETADO' if meets else 'SIN_ALTERNATIVA', versionAlgoritmo='solar-v1', alternativas=[row], totalAlternativas=1, alternativasTruncadas=False, criterioSeleccion='MENOR_SUPERFICIE' if meets else 'MAYOR_COBERTURA_ALCANZABLE', fuenteRecurso='Manual: ' + recurso['fuente'], hsp=float(recurso['hsp']), supuestos=['Catálogo de ejemplo: único panel activo, 500 W, superficie efectiva 2,5 m², sin precio.', 'Cobertura es balance energético; no autonomía horaria.'], traceId=trace)


class Limiter:
    """/**
     * Mantiene token buckets por identidad y sincroniza su acceso concurrente en el mock.
     *
     * @method Limiter
     * @returns {Limiter} Tipo que agrupa el estado y las verificaciones descritas.
     */
    """
    def __init__(self):
        """/**
         * Inicializa reposición, capacidad de ráfaga, buckets y bloqueo del limitador.
         *
         * @method __init__
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        self.entries, self.lock = {}, threading.Lock()

    def allow(self, key, capacity, rate):
        """/**
         * Repone tokens de la identidad y consume uno si la solicitud cumple la cuota.
         *
         * @method allow
         * @param {str|None} key - Identidad o credencial utilizada por esta operación de prueba.
         * @param {int} capacity - Máximo de tokens que admite la ráfaga del limitador.
         * @param {float} rate - Reposición de tokens por segundo y por identidad.
         * @returns {bool} Indica si hay tokens disponibles para aceptar la solicitud.
         */
        """
        now = time.monotonic()
        with self.lock:
            self.entries = {k: v for k, v in self.entries.items() if now - v[1] < 120}
            tokens, stamp = self.entries.get(key, (capacity, now))
            tokens = min(capacity, tokens + (now - stamp) * rate)
            allowed = tokens >= 1
            self.entries[key] = (tokens - 1 if allowed else tokens, now)
            return allowed


class Handler(BaseHTTPRequestHandler):
    """/**
     * Atiende las rutas del mock histórico con validación, autenticación, cuotas y salud protegida.
     *
     * @method Handler
     * @returns {Handler} Tipo que agrupa el estado y las verificaciones descritas.
     */
    """
    limiter = Limiter()

    def log_message(self, *_):
        """/**
         * Suprime el log HTTP predeterminado para mantener legible la salida de pruebas.
         *
         * @method log_message
         * @param {Any} _ - Argumentos de logging del servidor que se ignoran deliberadamente.
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        pass  # No cuerpos, IP, credenciales ni query strings.

    def send_json(self, code, body, trace=None):
        """/**
         * Escribe JSON y cabeceras del mock, respetando las respuestas sin cuerpo de HEAD.
         *
         * @method send_json
         * @param {int} code - Estado HTTP que se envía o espera en la respuesta.
         * @param {dict|None} body - Cuerpo de respuesta que se serializa como JSON.
         * @param {str} trace - Identificador de correlación que acompaña el error o la respuesta.
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        raw = json.dumps(body, ensure_ascii=False, allow_nan=False).encode('utf-8')
        self.send_response(code)
        self.send_header('Content-Type', 'application/json; charset=utf-8')
        self.send_header('Content-Length', str(len(raw)))
        self.send_header('X-Mock', 'entrega-1')
        if self.path.startswith('/health/'):
            self.send_header('Cache-Control', 'no-store')
            if code == 401: self.send_header('WWW-Authenticate', 'Bearer realm="healthcheck"')
        if trace: self.send_header('X-Request-Id', trace)
        if code in (429, 503): self.send_header('Retry-After', '6')
        self.end_headers()
        if self.command != 'HEAD': self.wfile.write(raw)

    def error(self, code, trace, message=None, details=None):
        """/**
         * Escribe el error del contrato con código, mensaje, correlación y detalles opcionales.
         *
         * @method error
         * @param {int} code - Estado HTTP que se envía o espera en la respuesta.
         * @param {str} trace - Identificador de correlación que acompaña el error o la respuesta.
         * @param {str} message - Descripción pública del error o del registro de prueba.
         * @param {list|None} details - Detalle opcional de campos rechazados en el error público.
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        self.send_json(code, dict(codigo=ERRORS[code], mensaje=message or ERRORS[code], traceId=trace, detalles=details or []), trace)

    def do_GET(self):
        """/**
         * Responde la salud protegida y rechaza rutas GET ajenas al mock.
         *
         * @method do_GET
         * @returns {None} Escribe la respuesta HTTP de salud o ruta desconocida.
         */
        """
        if self.path.startswith('/health/'):
            supplied = hashlib.sha256(self.headers.get('Authorization', '').encode('utf-8')).digest()
            if not hmac.compare_digest(supplied, self.server.healthcheck_auth_digest):
                return self.send_json(401, {'mensaje': 'No autenticado'})
            if self.path == '/health/live': return self.send_json(200, {'estado': 'disponible'})
            return self.send_json(404, {'mensaje': 'Ruta de salud no disponible'})
        elif self.path in ('/openapi.json', '/guest-openapi.json'):
            self.send_json(200, DOCS['v1' if self.path == '/openapi.json' else 'guest-v1'])
        else: self.send_json(404, {'mensaje': 'Ruta no incluida en el mock'})

    def do_HEAD(self):
        """/**
         * Reutiliza la respuesta de salud sin enviar un cuerpo HTTP.
         *
         * @method do_HEAD
         * @returns {None} No devuelve un valor; comprueba o aplica los efectos descritos.
         */
        """
        self.do_GET()

    def do_POST(self):
        """/**
         * Valida framing, credenciales, cuota y esquema antes de ejecutar consumo o comparación ilustrativa.
         *
         * @method do_POST
         * @returns {None} Valida la solicitud y escribe un resultado o error del contrato del mock.
         */
        """
        trace = self.headers.get('X-Request-Id') or str(uuid.uuid4())
        self.connection.settimeout(5)
        if self.headers.get('Transfer-Encoding'): return self.error(400, str(uuid.uuid4()), 'Mock requiere Content-Length, sin Transfer-Encoding')
        try: length = int(self.headers.get('Content-Length', '-1'))
        except ValueError: return self.error(400, str(uuid.uuid4()))
        if length < 0: return self.error(400, str(uuid.uuid4()))
        if length > 262144:
            # Drenaje acotado para que cerrar una solicitud rechazada no oculte el 413.
            try: self.rfile.read(min(length, 1048576))
            except TimeoutError: pass
            return self.error(413, str(uuid.uuid4()))
        try: raw = self.rfile.read(length)
        except TimeoutError: return self.error(400, str(uuid.uuid4()), 'Cuerpo incompleto')
        if not re.fullmatch(r'[A-Za-z0-9_-]{1,64}', trace):
            return self.error(422, str(uuid.uuid4()), 'X-Request-Id inválido')
        if self.path not in (M2M, CONSUMO, SOLAR): return self.send_json(404, {'mensaje': 'Ruta no incluida en el mock'})
        if self.path == M2M:
            key = self.headers.get('X-API-Key')
            if key == 'mock-sin-permiso': return self.error(403, trace)
            if key != 'mock-consumidor': return self.error(401, trace)
        identity = 'm2m' if self.path == M2M else self.client_address[0]
        if not self.limiter.allow((identity, self.path), 2 if self.path == SOLAR else 10, 1/6 if self.path == SOLAR else 1):
            return self.error(429, trace)
        if self.headers.get('Content-Type', '').split(';')[0].strip().lower() != 'application/json': return self.error(415, trace)
        try:
            data = json.loads(raw, parse_float=Decimal, parse_constant=lambda x: (_ for _ in ()).throw(ValueError(x)))
        except (ValueError, UnicodeError): return self.error(400, trace)
        doc = DOCS['v1' if self.path == M2M else 'guest-v1']
        operation = doc['paths'][self.path]['post']
        try:
            validate(data, operation['requestBody']['content']['application/json']['schema'], doc)
            with localcontext() as ctx:
                ctx.prec = 80
                precision(data)
                result = solar(data, trace) if self.path == SOLAR else consumo(data, trace)
        except ValidationError as error:
            return self.error(422, trace, 'Entrada inválida', [{'campo': error.field, 'motivo': error.reason}])
        except (ValueError, ArithmeticError): return self.error(422, trace, 'Entrada inválida: revisar esquema, ids, precisión y días')
        scenario = self.headers.get('X-Mock-Scenario', '')
        if scenario in ('429', '500', '503'): return self.error(int(scenario), trace)
        if result is None: return self.error(503, trace, 'Mock sin proveedor de recurso por ubicación; usar recurso MANUAL')
        self.send_json(200, result, trace)


def create_server(host='127.0.0.1', port=8080, healthcheck_token=None):
    """/**
     * Configura el servidor HTTP del mock y exige una credencial operativa válida antes de escuchar.
     *
     * @method create_server
     * @param {str} host - Dirección de escucha del servidor de prueba.
     * @param {int} port - Puerto de escucha del servidor de prueba.
     * @param {str} healthcheck_token - Credencial operativa de prueba que protege la consulta de salud.
     * @returns {ThreadingHTTPServer} Servidor de prueba configurado con credencial operativa y limitadores.
     */
    """
    token = os.getenv('HEALTHCHECK_TOKEN', '') if healthcheck_token is None else healthcheck_token
    if len(token) < 32:
        raise ValueError('HEALTHCHECK_TOKEN debe tener al menos 32 caracteres; usar scripts/compose para el arranque local')
    server = ThreadingHTTPServer((host, port), Handler)
    server.healthcheck_auth_digest = hashlib.sha256(('Bearer ' + token).encode('utf-8')).digest()
    return server


if __name__ == '__main__':
    server = create_server(os.getenv('HOST', '127.0.0.1'), int(os.getenv('PORT', '8080')))
    print('Mock Entrega 1 listo; sin persistencia ni proveedor real.', flush=True)
    server.serve_forever()
