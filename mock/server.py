"""Mock local de contrato; sin BD, proveedores ni persistencia de negocio."""
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


def validate(value, schema, doc, path='$'):
    """Comprueba el subconjunto de schemas usado aquí; no es un validador OpenAPI general."""
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
            raise ValueError(path + ': variante inválida o campos incompatibles')
        return
    kind = schema.get('type')
    numeric = isinstance(value, (int, float, Decimal)) and not isinstance(value, bool)
    valid = {'object': isinstance(value, dict), 'array': isinstance(value, list), 'string': isinstance(value, str), 'number': numeric, 'integer': numeric and (kind != 'integer' or value == value // 1), 'boolean': isinstance(value, bool)}
    if kind and not valid[kind]:
        raise ValueError(path + ': tipo inválido')
    if 'enum' in schema and value not in schema['enum']:
        raise ValueError(path + ': valor no admitido')
    if kind == 'object':
        props = schema.get('properties', {})
        if any(k not in value for k in schema.get('required', [])):
            raise ValueError(path + ': faltan campos obligatorios')
        if schema.get('additionalProperties') is False and set(value) - set(props):
            raise ValueError(path + ': campos desconocidos')
        for key, item in value.items():
            if key in props:
                validate(item, props[key], doc, path + '.' + key)
    elif kind == 'array':
        if not schema.get('minItems', 0) <= len(value) <= schema.get('maxItems', float('inf')):
            raise ValueError(path + ': tamaño inválido')
        for i, item in enumerate(value):
            validate(item, schema['items'], doc, f'{path}[{i}]')
    elif kind == 'string':
        if not schema.get('minLength', 0) <= len(value) <= schema.get('maxLength', float('inf')):
            raise ValueError(path + ': longitud inválida')
        if 'pattern' in schema and not re.fullmatch(schema['pattern'], value):
            raise ValueError(path + ': formato inválido')
    elif kind in ('number', 'integer'):
        for key, exclusive, sign in [('minimum', 'exclusiveMinimum', -1), ('maximum', 'exclusiveMaximum', 1)]:
            if key in schema and ((value - schema[key]) * sign > 0 or (schema.get(exclusive) and value == schema[key])):
                raise ValueError(path + ': fuera de rango')


def precision(value):
    if isinstance(value, Decimal) and value != value.quantize(Decimal('0.000001')):
        raise ValueError('Se admiten como máximo seis decimales')
    if isinstance(value, dict):
        for v in value.values(): precision(v)
    if isinstance(value, list):
        for v in value: precision(v)


def rounded(value):
    return float(Decimal(value).quantize(Decimal('0.000001'), rounding=ROUND_HALF_UP))


def consumo(data, trace):
    seen, rows, total = set(), [], Decimal(0)
    for e in data['equipos']:
        if e['id'] in seen: raise ValueError('equipos.id: identificador repetido')
        seen.add(e['id'])
        if e['modo'] == 'POTENCIA':
            if e['diasUso'] > data['diasPeriodo']: raise ValueError('equipos.diasUso: supera diasPeriodo')
            energy = Decimal(e['potenciaW']) * e['cantidad'] * e['horasPorDia'] * e['diasUso'] * e['factorFuncionamiento'] / 1000
        else:
            energy = Decimal(e['energiaPorCicloKWh']) * e['cantidad'] * e['ciclosPeriodo']
        total += energy
        rows.append({'id': e['id'], 'energiaPeriodoKWh': rounded(energy)})
    return dict(versionAlgoritmo='consumo-v1', diasPeriodo=int(data['diasPeriodo']), desglose=rows, totalPeriodoKWh=rounded(total), promedioDiarioKWh=rounded(total / data['diasPeriodo']), supuestos=['Estimación basada en equipos y hábitos declarados; no es una medición real.', 'El factor de funcionamiento se aplica una sola vez.'], traceId=trace)


def solar(data, trace):
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
    def __init__(self):
        self.entries, self.lock = {}, threading.Lock()

    def allow(self, key, capacity, rate):
        now = time.monotonic()
        with self.lock:
            self.entries = {k: v for k, v in self.entries.items() if now - v[1] < 120}
            tokens, stamp = self.entries.get(key, (capacity, now))
            tokens = min(capacity, tokens + (now - stamp) * rate)
            allowed = tokens >= 1
            self.entries[key] = (tokens - 1 if allowed else tokens, now)
            return allowed


class Handler(BaseHTTPRequestHandler):
    limiter = Limiter()

    def log_message(self, *_): pass  # No cuerpos, IP, credenciales ni query strings.

    def send_json(self, code, body, trace=None):
        raw = json.dumps(body, ensure_ascii=False, allow_nan=False).encode('utf-8')
        self.send_response(code)
        self.send_header('Content-Type', 'application/json; charset=utf-8')
        self.send_header('Content-Length', str(len(raw)))
        self.send_header('X-Mock', 'entrega-1')
        if trace: self.send_header('X-Request-Id', trace)
        if code in (429, 503): self.send_header('Retry-After', '6')
        self.end_headers()
        self.wfile.write(raw)

    def error(self, code, trace, message=None):
        self.send_json(code, dict(codigo=ERRORS[code], mensaje=message or ERRORS[code], traceId=trace, detalles=[]), trace)

    def do_GET(self):
        if self.path == '/health/live': self.send_json(200, {'estado': 'disponible', 'tipo': 'mock'})
        elif self.path in ('/openapi.json', '/guest-openapi.json'):
            self.send_json(200, DOCS['v1' if self.path == '/openapi.json' else 'guest-v1'])
        else: self.send_json(404, {'mensaje': 'Ruta no incluida en el mock'})

    def do_POST(self):
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
        except (ValueError, ArithmeticError): return self.error(422, trace, 'Entrada inválida: revisar esquema, ids, precisión y días')
        scenario = self.headers.get('X-Mock-Scenario', '')
        if scenario in ('429', '500', '503'): return self.error(int(scenario), trace)
        if result is None: return self.error(503, trace, 'Mock sin proveedor de recurso por ubicación; usar recurso MANUAL')
        self.send_json(200, result, trace)


def create_server(host='127.0.0.1', port=8080):
    return ThreadingHTTPServer((host, port), Handler)


if __name__ == '__main__':
    server = create_server(os.getenv('HOST', '127.0.0.1'), int(os.getenv('PORT', '8080')))
    print('Mock Entrega 1 listo; sin persistencia ni proveedor real.', flush=True)
    server.serve_forever()
