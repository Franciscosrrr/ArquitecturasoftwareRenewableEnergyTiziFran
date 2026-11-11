"""Limpieza operativa explícita: conserva pruebas ejecutables y elimina sus restos persistidos."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
from datetime import datetime, timezone

ROOT = Path(__file__).resolve().parents[1]
APPS = ['gateway', 'usuarios', 'consumo', 'electrodomesticos', 'solar', 'solar-2']
KEEP_EMAILS = {'admin@renewable.local', '2407889@ucc.edu.ar'}
SQL_TABLES = {'mysql-usuarios': ['usuarios', 'sesiones'], 'mysql-consumo': ['lugares', 'configuraciones', 'evaluaciones', 'idempotencia', 'outbox'], 'mysql': ['electrodomesticos', 'catalogo_outbox']}


def docker(*args, data=None, env=None):
    """/**
     * Ejecuta Docker Compose del repositorio sin construir comandos con datos externos.
     *
     * @method docker
     * @param {tuple} args - Argumentos separados de Compose.
     * @param {bytes|None} data - Entrada estándar opcional.
     * @param {dict|None} env - Entorno temporal sin imprimir valores privados.
     * @returns {bytes} Salida del proceso; lanza error saneado si falla.
     */
    """
    result = subprocess.run(['docker', 'compose', *args], cwd=ROOT, input=data, capture_output=True, env=env)
    if result.returncode:
        raise RuntimeError('Falló Compose ' + ' '.join(args[:3]) + ': ' + result.stderr.decode('utf-8', errors='replace')[-1200:])
    return result.stdout


def sql(service, query):
    """/**
     * Consulta la base propia del contenedor con su credencial interna y entrada estándar.
     *
     * @method sql
     * @param {str} service - Servicio MySQL autorizado en el inventario.
     * @param {str} query - SQL compuesto con tablas fijas y literales seguros.
     * @returns {str} Filas UTF-8 sin cabeceras ni credenciales.
     */
    """
    if service not in SQL_TABLES:
        raise ValueError('Base fuera del alcance de limpieza')
    shell = 'exec env MYSQL_PWD="$MYSQL_PASSWORD" mysql --default-character-set=utf8mb4 --batch --raw --skip-column-names -u"$MYSQL_USER" -D "$MYSQL_DATABASE"'
    return docker('exec', '-T', service, 'sh', '-c', shell, data=(query + '\n').encode()).decode('utf-8')


def mongo(script):
    """/**
     * Ejecuta una operación Mongo acotada a solar_db usando la credencial del contenedor.
     *
     * @method mongo
     * @param {str} script - JavaScript de mantenimiento con datos serializados como JSON.
     * @returns {dict} Resultado EJSON de la operación.
     */
    """
    shell = 'exec mongosh --quiet --username root --password "$MONGO_INITDB_ROOT_PASSWORD" --authenticationDatabase admin --file /dev/stdin'
    raw = docker('exec', '-T', 'mongo', 'sh', '-c', shell, data=script.encode()).decode('utf-8')
    return json.loads(raw.strip().splitlines()[-1])


def snapshot():
    """/**
     * Captura registros canónicos para planificar y comprobar la conservación de datos reales.
     *
     * @method snapshot
     * @returns {dict} Filas SQL y documentos Mongo; no imprime hashes ni sesiones.
     */
    """
    result = {}
    for service, tables in SQL_TABLES.items():
        result[service] = {}
        for table in tables:
            columns = [line.split('\t')[0] for line in sql(service, 'SHOW COLUMNS FROM `' + table + '`;').splitlines()]
            fields = ','.join("'" + c + "',`" + c + '`' for c in columns)
            result[service][table] = [json.loads(line) for line in sql(service, 'SELECT JSON_OBJECT(' + fields + ') FROM `' + table + '`;').splitlines()]
    result['mongo'] = mongo('const d=db.getSiblingDB("solar_db");print(EJSON.stringify({paneles:d.paneles.find().toArray(),estudios:d.estudios.find().toArray()},{relaxed:false}));')
    return result


def strings(value):
    """/**
     * Recorre snapshots para detectar referencias que impiden eliminar una ficha todavía usada.
     *
     * @method strings
     * @param {Any} value - Documento, lista o valor escalar.
     * @returns {set} Cadenas presentes, incluidas las de JSON almacenado como texto.
     */
    """
    if isinstance(value, str):
        result = {value}
        if value.startswith(('{', '[')):
            try:
                result |= strings(json.loads(value))
            except ValueError:
                pass
        return result
    if isinstance(value, dict):
        return set().union(*(strings(v) for v in value.values())) if value else set()
    if isinstance(value, list):
        return set().union(*(strings(v) for v in value)) if value else set()
    return set()


def plan(data, final_only=False):
    """/**
     * Selecciona cuentas sintéticas y fichas generadas por pruebas sin tocar referencias conservadas.
     *
     * @method plan
     * @param {dict} data - Inventario canónico actual.
     * @param {bool} final_only - Conserva exclusivamente las dos cuentas autorizadas cuando es verdadero.
     * @returns {dict} Identificadores a borrar y conteos verificables.
     */
    """
    users = data['mysql-usuarios']['usuarios']
    retained = [u for u in users if u['email'].lower() in KEEP_EMAILS]
    if final_only and {u['email'].lower() for u in retained} != KEEP_EMAILS:
        raise ValueError('Falta una cuenta que debe conservarse')
    if not any(u['email'] == 'admin@renewable.local' and u['rol'] == 'ADMINISTRADOR' for u in retained):
        raise ValueError('No se confirmó el administrador que debe conservarse')
    removed = {u['id'] for u in users if u['email'].lower() not in KEEP_EMAILS and (final_only or re.fullmatch(r'(?:prueba|navegador|web-edge)-.+@example\.test', u['email']))}
    protected = []
    for rows in data['mysql-consumo'].values():
        protected += [r for r in rows if r.get('owner_id') not in removed]
    protected += [r for r in data['mongo']['estudios'] if r.get('owner') not in removed]
    references = strings(protected)
    equipment = {r['id'] for r in data['mysql']['electrodomesticos'] if re.fullmatch(r'(?:verif-?|reconnect)[0-9a-f]{16}', r['id']) and r['categoria'] == 'Prueba' and r['id'] not in references}
    panels = {r['_id'] for r in data['mongo']['paneles'] if re.fullmatch(r'(?:verif-|solar)[0-9a-f]{16}', r['_id']) and ('prueba' in r.get('fuente', '').lower() or 'fixture' in r.get('fuente', '').lower()) and r['_id'] not in references}
    return {'usuarios': sorted(removed), 'equipos': sorted(equipment), 'paneles': sorted(panels), 'conservados': sorted(u['email'] for u in retained), 'conteos': {'usuarios': len(removed), 'equipos': len(equipment), 'paneles': len(panels)}}


def fingerprint(data, selection):
    """/**
     * Calcula una huella de los registros que deben permanecer intactos durante la limpieza.
     *
     * @method fingerprint
     * @param {dict} data - Inventario antes o después de la operación.
     * @param {dict} selection - Identificadores seleccionados para retirar.
     * @returns {str} SHA-256 de credenciales y datos conservados, sin exponer sus valores.
     */
    """
    result = {}
    for service, tables in data.items():
        result[service] = {}
        for table, rows in tables.items():
            kept = []
            for row in rows:
                if service == 'mysql-usuarios' and row.get('usuario_id', row.get('id')) in selection['usuarios']:
                    continue
                if service == 'mysql-consumo' and row.get('owner_id') in selection['usuarios']:
                    continue
                if service == 'mysql' and row.get('entity_id', row.get('id')) in selection['equipos']:
                    continue
                if service == 'mongo' and (row.get('owner') in selection['usuarios'] or row.get('_id') in selection['paneles']):
                    continue
                kept.append(json.dumps(row, sort_keys=True, ensure_ascii=False))
            result[service][table] = sorted(kept)
    return hashlib.sha256(json.dumps(result, sort_keys=True, ensure_ascii=False).encode()).hexdigest()


def literals(values):
    """/**
     * Forma literales SQL hexadecimales sin interpolar texto dentro de comillas del shell.
     *
     * @method literals
     * @param {list} values - Identificadores exactos seleccionados en el inventario.
     * @returns {str} Lista segura para IN; NULL cuando no hay elementos.
     */
    """
    return ','.join('CONVERT(0x' + v.encode().hex() + ' USING utf8mb4)' for v in values) or 'NULL'


def backup(destination, data):
    """/**
     * Guarda copias privadas antes de borrar registros y evita incluirlas en Git.
     *
     * @method backup
     * @param {Path} destination - Carpeta nueva dentro de .runtime.
     * @param {dict} data - Captura Mongo EJSON y SQL previa a la limpieza.
     * @returns {None} Escribe dumps SQL y snapshot privado; falla antes de borrar si no puede respaldar.
     */
    """
    destination.mkdir(parents=True, exist_ok=False)
    for service in SQL_TABLES:
        shell = 'exec env MYSQL_PWD="$MYSQL_PASSWORD" mysqldump --single-transaction --no-tablespaces --set-gtid-purged=OFF --default-character-set=utf8mb4 -u"$MYSQL_USER" "$MYSQL_DATABASE"'
        (destination / (service + '.sql')).write_bytes(docker('exec', '-T', service, 'sh', '-c', shell))
    (destination / 'snapshot-privado.json').write_text(json.dumps(data, ensure_ascii=False), encoding='utf-8')


def dependencies(selection):
    """/**
     * Retira documentos derivados y purga una cola solo si todos sus mensajes son restos identificados.
     *
     * @method dependencies
     * @param {dict} selection - Identidades sintéticas y fichas retiradas.
     * @returns {dict} Tareas de índice y colas limpiadas o conservadas.
     */
    """
    environment = os.environ.copy()
    for line in (ROOT / '.env').read_text(encoding='utf-8-sig').splitlines():
        if line.startswith('SEARCH_API_KEY='):
            environment['CLEANUP_SEARCH_KEY'] = line.split('=', 1)[1].strip().strip('"').strip("'")
    program = '''import base64,json,os,time,sys
from urllib.request import Request,urlopen
p=json.load(sys.stdin)
def call(base,path,headers,method='GET',body=None):
 """/**
  * Ejecuta una operación JSON de mantenimiento en una dependencia interna.
  *
  * @method call
  * @param {str} base - URL interna fija de la dependencia.
  * @param {str} path - Ruta del índice o cola dentro del proyecto.
  * @param {dict} headers - Credencial interna y tipo de contenido, sin registrarlos.
  * @param {str} method - Método HTTP de la operación.
  * @param {Any} body - Cuerpo JSON opcional.
  * @returns {Any} Respuesta JSON o None si la operación no devuelve cuerpo.
  */
 """
 req=Request(base+path,method=method,headers=headers,data=None if body is None else json.dumps(body).encode())
 with urlopen(req,timeout=8) as r:
  raw=r.read();return json.loads(raw) if raw else None
h={'Content-Type':'application/json','Authorization':'Bearer '+os.environ['CLEANUP_SEARCH_KEY']}
tasks=[]
for index,ids in [('catalogo',p['equipos']),('paneles',p['paneles'])]:
 if ids:
  task=call('http://meilisearch:7700','/indexes/'+index+'/documents/delete-batch',h,'POST',ids)
  deadline=time.monotonic()+20
  while True:
   state=call('http://meilisearch:7700','/tasks/'+str(task['taskUid']),h)
   if state['status']=='succeeded':break
   if state['status']=='failed' or time.monotonic()>deadline:raise RuntimeError('No se confirmó la eliminación del índice')
   time.sleep(.2)
  tasks.append(index)
auth=base64.b64encode(('renewable:'+os.environ['RABBITMQ_PASSWORD']).encode()).decode()
h={'Content-Type':'application/json','Authorization':'Basic '+auth}
queues=call('http://rabbitmq:15672','/api/queues/%2F',h)
removed=[];kept=[];ids=set(p['usuarios']+p['equipos']+p['paneles'])
for q in queues:
 if not q['name'].startswith(('solar.estudios','catalogo.indice')) or not q.get('messages',0):continue
 messages=call('http://rabbitmq:15672','/api/queues/%2F/'+q['name']+'/get',h,'POST',{'count':q['messages'],'ackmode':'ack_requeue_true','encoding':'auto','truncate':1048576})
 disposable=True
 for m in messages:
  raw=base64.b64decode(m['payload']).decode() if m.get('payload_encoding')=='base64' else m['payload']
  try:payload=json.loads(raw)
  except ValueError:disposable=False;break
  if not any(v in raw for v in ids) and not str(payload.get('tipo','')).endswith('.v99'):disposable=False;break
 if disposable and len(messages)==q['messages']:
  call('http://rabbitmq:15672','/api/queues/%2F/'+q['name']+'/contents',h,'DELETE');removed.append(q['name'])
 else:kept.append(q['name'])
print(json.dumps({'indices':tasks,'colas_limpiadas':removed,'colas_conservadas':kept}))
'''
    raw = docker('--profile', 'verificar', 'run', '--rm', '--no-deps', '-T', '-e', 'CLEANUP_SEARCH_KEY', 'verificar-real', 'python', '-B', '-c', program, data=json.dumps(selection).encode(), env=environment)
    result = json.loads(raw.decode().strip().splitlines()[-1])
    environment['REDISCLI_AUTH'] = next(line.split('=', 1)[1].strip().strip('"').strip("'") for line in (ROOT / '.env').read_text().splitlines() if line.startswith('VALKEY_PASSWORD='))
    docker('exec', '-T', '-e', 'REDISCLI_AUTH', 'valkey', 'valkey-cli', 'INCR', 'catalogo:generacion', env=environment)
    return result


def main():
    """/**
     * Planifica por defecto y aplica limpieza únicamente mediante una opción explícita.
     *
     * @method main
     * @returns {None} Informa conteos, respalda y conserva los registros legítimos; restaura los servicios al salir.
     */
    """
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--aplicar', action='store_true')
    parser.add_argument('--solo-cuentas-finales', action='store_true', help='Conservar exclusivamente admin y 2407889, autorizado para preparar la entrega')
    args = parser.parse_args()
    initial = snapshot()
    selected = plan(initial, args.solo_cuentas_finales)
    print(json.dumps({'plan': selected['conteos'], 'conservar': selected['conservados'], 'aplicar': args.aplicar}, ensure_ascii=False), flush=True)
    if not args.aplicar:
        return
    raw_states = docker('ps', '--format', 'json').decode().strip()
    states = json.loads(raw_states) if raw_states.startswith('[') else [json.loads(line) for line in raw_states.splitlines() if line.strip()]
    running = [name for name in APPS if any(row['Service'] == name and row['State'] == 'running' for row in states)]
    directory = ROOT / '.runtime' / 'respaldos-limpieza' / datetime.now(timezone.utc).strftime('%Y%m%dT%H%M%S%fZ')
    try:
        if running:
            docker('stop', *running)
        initial = snapshot()
        selected = plan(initial, args.solo_cuentas_finales)
        expected = fingerprint(initial, selected)
        backup(directory, initial)
        users = literals(selected['usuarios'])
        sql('mysql-consumo', 'START TRANSACTION;' + ''.join('DELETE FROM `' + table + '` WHERE owner_id IN (' + users + ');' for table in ['outbox', 'idempotencia', 'evaluaciones', 'configuraciones', 'lugares']) + 'COMMIT;')
        sql('mysql-usuarios', 'START TRANSACTION;DELETE FROM sesiones WHERE usuario_id IN (' + users + ');DELETE FROM usuarios WHERE id IN (' + users + ');COMMIT;')
        equipment = literals(selected['equipos'])
        sql('mysql', 'START TRANSACTION;DELETE FROM catalogo_outbox WHERE entity_id IN (' + equipment + ');DELETE FROM electrodomesticos WHERE id IN (' + equipment + ');COMMIT;')
        mongo('const p=' + json.dumps(selected) + ';const d=db.getSiblingDB("solar_db");const a=d.estudios.deleteMany({owner:{$in:p.usuarios}});const b=d.paneles.deleteMany({_id:{$in:p.paneles}});print(EJSON.stringify({estudios:a.deletedCount,paneles:b.deletedCount}));')
        derived = dependencies(selected)
        final = snapshot()
        if fingerprint(final, selected) != expected:
            raise RuntimeError('Revisar respaldo: cambió un registro que debía conservarse')
        if plan(final, args.solo_cuentas_finales)['conteos'] != {'usuarios': 0, 'equipos': 0, 'paneles': 0}:
            raise RuntimeError('Quedaron registros de ensayo por revisar')
        result = {'fecha': datetime.now(timezone.utc).isoformat(), 'eliminados': selected['conteos'], 'usuarios_finales': [{'email': row['email'], 'rol': row['rol']} for row in final['mysql-usuarios']['usuarios']], 'equipos_finales': len(final['mysql']['electrodomesticos']), 'paneles_finales': len(final['mongo']['paneles']), 'datos_conservados_sin_cambios': True, 'derivados': derived}
        (directory / 'resultado.json').write_text(json.dumps(result, ensure_ascii=False, indent=2), encoding='utf-8')
        print(json.dumps(result, ensure_ascii=False, indent=2), flush=True)
    finally:
        if running:
            docker('start', *running)
        print('Servicios previos reanudados; respaldo privado en .runtime/respaldos-limpieza.', flush=True)


if __name__ == '__main__':
    main()
