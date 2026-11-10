"""Sonda administrativa del broker; solo se ejecuta dentro de la red de pruebas."""
import base64,json,os,sys
from urllib.request import Request,urlopen
auth=base64.b64encode(('renewable:'+os.environ['RABBITMQ_PASSWORD']).encode()).decode()
headers={'Authorization':'Basic '+auth,'Content-Type':'application/json'}
mode=sys.argv[1]
if mode=='publish':
    payload=json.load(sys.stdin)
    data={'properties':{'delivery_mode':2},'routing_key':'consumo.evaluado.v1','payload':json.dumps(payload),'payload_encoding':'string'}
    exchange='renewable.dlx' if len(sys.argv)>2 and sys.argv[2]=='dlq' else 'renewable.events'
    req=Request('http://rabbitmq:15672/api/exchanges/%2F/'+exchange+'/publish',headers=headers,data=json.dumps(data).encode())
else:req=Request('http://rabbitmq:15672/api/queues/%2F/solar.estudios.dlq',headers=headers)
with urlopen(req,timeout=5) as response:result=json.load(response)
print(json.dumps(result if mode=='publish' else {'messages':result.get('messages',0),'messages_ready':result.get('messages_ready',0)}))
