> Copia recibida del proveedor el 11/11/2026. Sus enlaces relativos se resolvieron contra el repositorio original; los resultados que declara corresponden al otro grupo. Nuestra evidencia está en [la revisión de integración](../../evidence/proveedor-real-2026-11-11/README.md).

# Cotización de paneles para Renewable Energy

Hoja de integración para el grupo **Renewable Energy**, que calcula consumo eléctrico
y compara paneles, y necesita cotizar por SKU contra nuestra capacidad.

Responde punto por punto el pedido que nos hicieron. Todo lo que dice "medido" o
"verificado" se corrió el **11/11/2026** contra la instancia desplegada; la evidencia
cruda está en
[`tests/carga/resultados/2026-11-11-latencia-nube/`](https://github.com/matiasdemetriorufeil/arq-soft-proveedores/blob/main/tests/carga/resultados/2026-11-11-latencia-nube/).

> **La credencial no está en este archivo ni en ningún otro del repositorio.** Se
> entrega por canal privado. Si alguna vez aparece una clave en Git, hay que rotarla.

## 1. No hace falta una API nueva: la que tenemos ya cotiza

`GET /api/v1/proveedores/recomendados?producto=<SKU>&cantidad=<N>` ya hace lo que
piden: dado un SKU y una cantidad devuelve precio y stock, y **es un `GET` de solo
lectura: no compra, no reserva y no descuenta inventario**. Consultarlo mil veces no
mueve un solo stock.

Así que **no publicamos una versión nueva**: el contrato sigue en **1.1.0**. Lo que sí
hicimos fue declarar en él lo que estaba implícito (moneda, unidades, frescura de los
datos y el hecho de que no tiene efectos), porque eso era un hueco real.

### Su propuesta contra lo que existe

| Su propuesta | Nuestro contrato real | Comentario |
|---|---|---|
| `sku` (body) | `producto` (query param) | Mismo concepto, otro nombre y va en la URL |
| `potenciaObjetivoW` | **no existe** | Ver el punto 3: la potencia del panel no es un dato nuestro |
| `cantidad` (derivada) | `cantidad` (query param, **la mandan ustedes**) | El techo `potenciaObjetivoW / potenciaPanelW` lo calculan de su lado |
| `cotizacionId` | header `X-Request-Id` | Sirve para correlacionar con nuestros logs, pero **no es una cotización guardada**: no se puede recuperar después |
| `potenciaPanelW` | **no existe** | Punto 3 |
| `stockDisponible` | `stock_disponible` | Entero, en la unidad del catálogo |
| `precioUnitarioCentavos` | `precio_unitario` | **ARS con 2 decimales, no centavos enteros** |
| `totalCentavos` | `costo_total` | **No es precio × cantidad**: ver abajo |
| `moneda` | sin campo | Todos los importes son **ARS**, ahora declarado en el contrato |

### Dos diferencias que les van a romper el test si no las miran

**a) Devolvemos un ranking, no una cotización.** La respuesta es un **array** de
ofertas de distintos proveedores, ordenado por `costo_total` ascendente (con desempate
determinista por nombre y después por id). Si quieren una sola cotización, la más
conveniente es `limite=1` o el primer elemento del array.

**b) `costo_total` incluye descuento por volumen y flete.** No es una multiplicación.
La fórmula del contrato es:

```
costo_total = precio_unitario × cantidad × (1 − descuento_pct/100) + flete
```

y el descuento **solo** se aplica si `cantidad ≥ cantidad_minima_descuento` de esa
oferta, un umbral que no exponemos. Medido hoy con `PAN-550W-MONO`, oferta de
*Cordoba Energia SA* (`precio_unitario = 189573.99`, `descuento_pct = 15`,
`flete = 43000`):

| `cantidad` | `descuento_aplicado` | `costo_total` | precio × cantidad |
|---|---|---|---|
| 2 | `false` | **422 147,98** | 379 147,98 |
| 10 | `true` | **1 654 378,92** | 1 895 739,90 |
| 50 | `true` | **8 099 894,58** | 9 478 699,50 |

Con `cantidad = 2` el total es **mayor** que precio × cantidad (por el flete); con
`cantidad = 10` es **menor** (el descuento pesa más que el flete). Un test que asuma
`total = precio × cantidad` falla en los dos casos. El campo `descuento_aplicado` dice
si el umbral se alcanzó.

## 2. Contrato OpenAPI

- **Archivo:** [`openapi.yaml`](https://github.com/matiasdemetriorufeil/arq-soft-proveedores/blob/main/docs/contracts/openapi.yaml) · **versión `1.1.0`** · OpenAPI 3.0.3
- **Tabla de errores, reglas y ejemplos:** [`README.md`](https://github.com/matiasdemetriorufeil/arq-soft-proveedores/blob/main/docs/contracts/README.md)
- Trae operaciones, autenticación, campos obligatorios, tipos, unidades, límites,
  reglas de negocio, errores y ejemplos.

**Unidades y moneda, declaradas:**

| Campo | Tipo | Unidad |
|---|---|---|
| `precio_unitario`, `flete`, `costo_total` | número decimal, 2 decimales | **ARS** |
| `cantidad`, `stock_disponible` | entero | unidades del catálogo (para paneles, `unidad`) |
| `descuento_pct` | número | porcentaje 0–100 (no fracción) |

**Sobre los centavos enteros que piden:** hoy los importes son decimales con 2
decimales. En la base son `NUMERIC(14,2)`, exactos; en el JSON viajan como número, así
que conviene leerlos como decimal y no como `float` binario. **Si necesitan centavos
enteros, lo acordamos**: agregar `precio_unitario_centavos` y `costo_total_centavos`
como campos nuevos es aditivo y no rompe a nadie, y saldría como **1.2.0**. Díganlo y
lo hacemos; no lo cambiamos unilateralmente porque hoy nadie los necesita.

## 3. Modelos cotizables

**Cinco modelos de panel, con SKU estable** (son la clave primaria de nuestro
catálogo; no cambian):

| SKU | Nombre | Categoría | Unidad |
|---|---|---|---|
| `PAN-100W-FLEX` | Panel solar 100 W flexible | paneles | unidad |
| `PAN-330W-POLI` | Panel solar 330 W policristalino | paneles | unidad |
| `PAN-450W-MONO` | Panel solar 450 W monocristalino | paneles | unidad |
| `PAN-550W-MONO` | Panel solar 550 W monocristalino | paneles | unidad |
| `PAN-600W-BIF` | Panel solar 600 W bifacial | paneles | unidad |

**El `MODELO-550` de su ejemplo no existe.** El equivalente nuestro es
`PAN-550W-MONO`. Un SKU inexistente devuelve `404 PRODUCTO_NO_ENCONTRADO`.

Además del rubro paneles tenemos inversores, baterías, estructuras, cableado,
protecciones y medición: **40 SKUs** en total, todos cotizables por la misma
operación.

### Lo que NO tenemos, y por qué

Pidieron **potencia en W, superficie por unidad y fuente técnica**. No los tenemos:
nuestra tabla `productos` guarda `sku`, `nombre`, `categoria` y `unidad`, y nada más.
**No los vamos a inventar**: una superficie o una ficha técnica inventada es peor que
no tenerla, porque entra en sus cálculos de consumo como si fuera un dato.

Nuestra propuesta, que además nos parece el corte correcto de dominio:

> **Las especificaciones técnicas del panel son de ustedes; el precio y el stock son
> nuestros.** Ustedes comparan paneles y calculan consumo: la potencia y la superficie
> son el centro de su dominio y seguro ya las tienen en su catálogo. Nosotros somos la
> autoridad de precio y disponibilidad. Con eso, `cantidad = ceil(potenciaObjetivoW /
> potenciaPanelW)` lo calculan de su lado con su propio dato, y nos preguntan por
> precio y stock de esa cantidad.

Si igual quieren que lo sirvamos nosotros, es viable pero necesita dos cosas: agregar
`potencia_w`, `superficie_m2` y `ficha_tecnica_url` al catálogo (aditivo, sería
**1.2.0**) y **que alguien aporte las fichas técnicas reales** de cada modelo.
Díganlo y lo coordinamos; no sale de nuestro lado solo.

> La potencia nominal aparece en el nombre y en el SKU (`PAN-550W-MONO` → 550 W), pero
> **eso es una convención de nombres, no un campo del contrato**. No lo parseen: si
> mañana renombramos un producto, su cálculo se rompe en silencio.

### Precios y vigencia

- **La respuesta es una foto del momento. No hay precio congelado ni cotización
  guardada.** El precio puede cambiar entre dos llamadas sin aviso.
- Las ofertas se **cachean 60 segundos**: un cambio de precio puede tardar hasta ese
  tiempo en verse. Un SKU inexistente se cachea 10 segundos.
- Las ofertas del proveedor externo que integramos se **re-importan cada 5 minutos**.
- Hoy los precios de nuestros proveedores propios cambian solo por migración del
  catálogo, no por un flujo automático.
- Si necesitan un precio sostenido por N minutos, **no lo tenemos y habría que
  diseñarlo**: implicaría persistir la cotización y darle un id, que es justo lo que
  hoy no hacemos.

## 4. Dónde está, cuotas y disponibilidad

| | |
|---|---|
| **URL** | `http://20.226.84.78` |
| **Operación** | `GET /api/v1/proveedores/recomendados` |
| **OpenAPI** | [`docs/contracts/openapi.yaml`](https://github.com/matiasdemetriorufeil/arq-soft-proveedores/blob/main/docs/contracts/openapi.yaml) en el repositorio |
| **Versión desplegada** | contrato **1.1.0**, tag `entrega-3` |
| **Autenticación** | header `X-API-Key`. Una clave propia para Renewable, por canal privado |
| **Cuota** | **50 req/s sostenidas, ráfagas de 100**, por clave (token bucket). Lo que exceda: `429` con `Retry-After` |
| **Infraestructura** | una sola VM (2 vCPU, 8 GB) en Azure `brazilsouth` |

**No es HTTPS, y no lo podemos resolver solos.** La URL es `http://` porque no tenemos
dominio y sin dominio no hay certificado. Pidieron HTTPS explícitamente, así que es una
decisión a tomar entre los dos grupos:

- **Aceptar `http://` para la evaluación académica.** No hay datos personales en juego;
  la clave viaja en un header sobre HTTP, con el riesgo que eso implica.
- **Lo ponemos detrás de TLS** si consiguen o aceptamos un dominio (uno gratuito de
  DuckDNS y Caddy con certificado automático alcanza). Es trabajo nuestro, de un rato,
  pero necesita esa decisión primero.

Nos inclinamos por la segunda si hay tiempo. Díganlo y lo hacemos.

**Disponibilidad:** la VM está levantada y la vamos a mantener levantada durante la
ventana de evaluación. Dos límites que preferimos decir antes:

- **Es una sola VM, sin réplicas ni respaldo.** Si se cae la máquina, se cae la
  capacidad: el balanceo interno cubre al servicio, no al host. Es una decisión de
  costo, documentada en [ADR-013](https://github.com/matiasdemetriorufeil/arq-soft-proveedores/blob/main/docs/adr/ADR-013-capacidad-y-costos.md).
- Si tenemos que apagarla por crédito o mantenimiento, **avisamos antes**. La IP es
  estática: la URL no cambia aunque se reinicie.

## 5. Errores

Todos en `application/problem+json` (RFC 9457) con la extensión `code`, y todos traen
`request_id` para que podamos cruzarlo con nuestros logs. **Respuestas reales,
copiadas de la corrida del 11/11** (evidencia en
[`errores.txt`](https://github.com/matiasdemetriorufeil/arq-soft-proveedores/blob/main/tests/carga/resultados/2026-11-11-latencia-nube/errores.txt)):

| Caso | HTTP | `code` | ¿Reintentar? |
|---|---|---|---|
| SKU inexistente | `404` | `PRODUCTO_NO_ENCONTRADO` | No |
| **Stock insuficiente** | **`200` con `[]`** | — | No |
| Entrada inválida | `400` | `PARAMETRO_INVALIDO` | No |
| Credencial ausente o inválida | `401` | `NO_AUTORIZADO` | No |
| Límite de solicitudes | `429` | `LIMITE_EXCEDIDO` | **Sí**, tras `Retry-After` |
| Ruta inexistente | `404` | `RUTA_NO_ENCONTRADA` | No |
| Base de datos caída | `503` | `NO_DISPONIBLE` | **Sí** |
| Réplicas sin responder | `502` / `504` | `NO_DISPONIBLE` | **Sí** |

**Ojo con "stock insuficiente": no es un error.** Si el producto existe pero ningún
proveedor tiene stock suficiente para la cantidad pedida, la respuesta es
**`200` con un array vacío**, no un `409` ni un `404`. Verificado: `PAN-550W-MONO` con
`cantidad=999999` devuelve `200 []`. Solo un SKU **que no existe** da `404`. Si su
cliente trata el array vacío como error de transporte, va a reintentar de más al
vacío.

**`429`:** medido contra la instancia desplegada, 300 llamadas concurrentes con la
misma clave dieron **127 × `200`** y **173 × `429`**, todas con `Retry-After: 1`.
Respetar esa cabecera; no reintentar antes.

**`401`:** la credencial ausente y la inválida dan **el mismo `401`**, a propósito: no
queremos que se pueda distinguir si una clave existe. No hay un `403` aparte, porque
hoy no hay permisos por recurso: la clave da acceso a la capacidad o a nada.

**No hay `405`:** el gateway tiene un catch-all, así que un método no soportado sobre
la ruta devuelve `404 RUTA_NO_ENCONTRADA` en lugar de "método no permitido". Está
decidido así y anotado en [`README.md`](https://github.com/matiasdemetriorufeil/arq-soft-proveedores/blob/main/docs/contracts/README.md).

## 6. Latencia y arranque en frío

Medido el 11/11 **desde Córdoba contra la instancia desplegada** (São Paulo), así que
incluye ida y vuelta de red: es lo que va a ver su cliente, no latencia de servidor.

| | |
|---|---|
| **p50** | **104 ms** |
| **p95** | **128 ms** |
| **p99** | 128 ms |
| mínimo / máximo | 99 ms / 128 ms |
| Primera llamada de un SKU sin cachear | **176 ms** |
| El mismo SKU ya cacheado (mediana de 10) | 107 ms |

40 llamadas sobre los cinco modelos de panel. El piso de ~100 ms es la red
Córdoba–São Paulo; desde otra ubicación cambia.

**Su presupuesto de ~1 segundo por llamada nos sobra**: el p95 es **ocho veces menor**.
No pedimos cambiarlo.

**Arranque en frío: no hay.** Son contenedores siempre encendidos, no funciones
serverless. Lo más parecido es la primera llamada de un SKU que no está en caché, y
son 176 ms contra 107 ms. Dos cosas que sí conviene saber:

- **Si nuestra caché se cae**, cada llamada paga 50 ms extra de timeout antes de ir a
  la base: medido, pasa de ~6 ms a ~56 ms de latencia interna. Sigue respondiendo
  correcto y sigue muy por debajo de su presupuesto.
- **Nuestro timeout interno hacia el servicio es de 3 s.** Si su cliente corta a 1 s,
  va a abandonar antes que nosotros: es una decisión razonable, pero significa que en
  un pico lento ustedes ven un timeout y nosotros un `200`. Si quieren alinearlos,
  lo acordamos.

Su circuit breaker y sus reintentos asíncronos nos parecen bien. Un pedido: **que los
reintentos respeten el `Retry-After` del `429`**, porque reintentar antes solo gasta la
cuota.

## 7. Ejemplos reproducibles

Reemplazando `$CLAVE` por la credencial que les pasamos. Los importes van a diferir si
el catálogo cambió; la **forma** de la respuesta es la del contrato.

### Éxito: cotizar 2 paneles de 550 W, la oferta más conveniente

```bash
curl -s "http://20.226.84.78/api/v1/proveedores/recomendados?producto=PAN-550W-MONO&cantidad=2&limite=1" \
  -H "X-API-Key: $CLAVE"
```

```json
[
  {
    "proveedor_id": 3,
    "proveedor_nombre": "Cordoba Energia SA",
    "precio_unitario": 189573.99,
    "descuento_pct": 15,
    "descuento_aplicado": false,
    "flete": 43000,
    "stock_disponible": 394,
    "costo_total": 422147.98
  }
]
```

### Éxito con descuento aplicado (mismo SKU, cantidad 10)

```bash
curl -s "http://20.226.84.78/api/v1/proveedores/recomendados?producto=PAN-550W-MONO&cantidad=10&limite=1" \
  -H "X-API-Key: $CLAVE"
```

Devuelve `descuento_aplicado: true` y `costo_total: 1654378.92`. Sirve de caso de
prueba para la fórmula completa.

### Stock insuficiente: `200` con array vacío

```bash
curl -s "http://20.226.84.78/api/v1/proveedores/recomendados?producto=PAN-550W-MONO&cantidad=999999" \
  -H "X-API-Key: $CLAVE"
```

```json
[]
```

### SKU inexistente: `404`

```bash
curl -s "http://20.226.84.78/api/v1/proveedores/recomendados?producto=MODELO-550&cantidad=2" \
  -H "X-API-Key: $CLAVE"
```

```json
{
  "type": "https://github.com/matiasdemetriorufeil/arq-soft-proveedores/blob/main/docs/contracts/README.md#PRODUCTO_NO_ENCONTRADO",
  "title": "Producto no encontrado",
  "status": 404,
  "detail": "no existe un producto con SKU 'MODELO-550'",
  "code": "PRODUCTO_NO_ENCONTRADO",
  "instance": "/api/v1/proveedores/recomendados",
  "request_id": "c6533a5b7837a5f37e84f4912a04ff62"
}
```

### Entrada inválida: `400`

```bash
curl -s "http://20.226.84.78/api/v1/proveedores/recomendados?producto=PAN-550W-MONO&cantidad=0" \
  -H "X-API-Key: $CLAVE"
```

Devuelve `400` con `code: PARAMETRO_INVALIDO` y
`detail: "'cantidad' debe ser un entero mayor a 0"`. Lo mismo con `cantidad=dos` o sin
el parámetro `producto`.

### Sin credencial: `401`

```bash
curl -s "http://20.226.84.78/api/v1/proveedores/recomendados?producto=PAN-550W-MONO&cantidad=2"
```

Devuelve `401` con `code: NO_AUTORIZADO` y
`detail: "falta el header X-API-Key o no es válido"`.

### Ensayo coordinado de caída y timeout

Para su test de consumidor y su circuit breaker, podemos provocar a pedido, en una
ventana que acordemos y avisando antes y después:

| Escenario | Qué van a ver | Cómo lo provocamos | ¿Ya medido? |
|---|---|---|---|
| Base de datos caída | `503 NO_DISPONIBLE` en ~2 s | detener PostgreSQL en la VM | **Sí**, 5 de 5 llamadas |
| Una réplica caída | nada: la capacidad sigue en `200` | detener una réplica | **Sí**, 20 de 20 en `200` |
| Caché caída | `200` igual, ~50 ms más lento | detener Redis | **Sí**, 6,5 ms → 56,6 ms |
| Límite de tráfico | `429` con `Retry-After: 1` | ráfagas sobre 100 req | **Sí**, 127 `200` / 173 `429` |
| Las dos réplicas caídas | `502 NO_DISPONIBLE` | detener las dos | **No**: está en [ADR-010](https://github.com/matiasdemetriorufeil/arq-soft-proveedores/blob/main/docs/adr/ADR-010-resiliencia.md) como implementado, pero no lo ejercitamos |
| Respuesta lenta / timeout | `504` a los 3 s | pausar el contenedor de la base | **No**: mismo caso |

Los cuatro primeros están medidos sobre el sistema completo, con la evidencia en
[`POSTMORTEM.md`](https://github.com/matiasdemetriorufeil/arq-soft-proveedores/blob/main/docs/POSTMORTEM.md). Los dos últimos están en el diseño y en el código
pero **no los corrimos todavía**, así que no los damos por verificados: si les sirven
para su test, los ejercitamos en el ensayo y les pasamos el resultado.

**El `429` lo pueden probar cuando quieran, sin coordinar.** Los demás los acordamos
por fecha y hora, porque dejan la capacidad caída mientras duran.

## Decisiones que necesitamos de ustedes

Cuatro cosas que no podemos resolver de nuestro lado:

1. **HTTPS**: ¿aceptan `http://` para la evaluación, o conseguimos un dominio y lo
   ponemos detrás de TLS?
2. **Centavos enteros**: ¿les alcanzan los decimales con 2 posiciones, o agregamos
   `*_centavos` como **1.2.0**?
3. **Potencia y superficie del panel**: ¿las toman de su catálogo (nuestra propuesta),
   o quieren que las sirvamos nosotros y consiguen las fichas técnicas reales?
4. **Ensayo de caída**: fecha y hora.
