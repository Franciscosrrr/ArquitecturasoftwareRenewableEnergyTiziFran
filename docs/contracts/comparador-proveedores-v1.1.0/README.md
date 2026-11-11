# Integración real: Comparador de Proveedores 1.1.0

Estado de implementación: 11/11/2026. Las comprobaciones y sus resultados se registran en `docs/evidence/proveedor-real-2026-11-11/` una vez ejecutadas.

Fuentes recibidas del grupo: [OpenAPI original](https://github.com/matiasdemetriorufeil/arq-soft-proveedores/blob/main/docs/contracts/openapi.yaml) y [hoja de integración](https://github.com/matiasdemetriorufeil/arq-soft-proveedores/blob/main/docs/contracts/para-renewable-energy.md). Se conservan copias locales sin credenciales: [OpenAPI 1.1.0](openapi.yaml) y [hoja original](para-renewable-energy.md). No se modifica unilateralmente su contrato.

## Operación y formatos

`GET http://20.226.84.78/api/v1/proveedores/recomendados?producto=PAN-550W-MONO&cantidad=2&limite=1`

Autenticación de servidor a servidor: `X-API-Key`, tomada exclusivamente de `PANEL_COMPARATOR_API_KEY` en `.env`. No se envía al navegador, no se incluye en ejemplos ni evidencia. Se envían siempre SKU, cantidad y límite aunque el OpenAPI marque cantidad opcional. La cantidad es `ceil(potenciaObjetivoW / potenciaFichaLocalW)`.

La respuesta es un array de ofertas ordenadas por costo total. Con `limite=1` se acepta una oferta o `[]`. Requiere proveedor, precio unitario, descuento porcentual, indicador de descuento aplicado, flete, stock y costo total. Los importes ARS se leen como decimales exactos y se convierten a centavos enteros; nunca pasan por `float64`. Se valida el total redondeado a dos decimales: unitario × cantidad, descuento únicamente si está aplicado, más flete. No se impone `total = precio × cantidad`.

Ejemplo del proveedor para dos unidades: precio 189573.99 ARS, descuento no aplicado y flete 43000 ARS → total **422147.98 ARS**. Para diez unidades, descuento aplicado del 15% → **1654378.92 ARS**. Los importes son ejemplos, no precios congelados.

No hay `cotizacionId` comercial: la adaptación deja ese campo vacío y conserva `referenciaProveedor` desde `X-Request-Id`. La oferta no se puede recuperar por esa referencia. En un estudio propio se conserva la foto de la oferta recibida, junto con su procedencia y fecha; eso no reserva inventario ni precio en el proveedor.

## Catálogo técnico y equivalencias

| Identidad local conservada | SKU comercial acordado | Potencia local | Superficie local de referencia |
|---|---|---|---|
| PANEL-SIM-450 | PAN-450W-MONO | 450 W | 2.5 m² |
| PANEL-SIM-550 | PAN-550W-MONO | 550 W | 2.8 m² |

Se conservan identidades, versiones, historial y fuentes existentes. Los nombres/SKU no se parsean para deducir potencia. Los precios y el stock pueden ser reales mientras las especificaciones técnicas continúan siendo referencias didácticas locales, expresamente indicadas. **Las superficies no fueron suministradas ni certificadas por el proveedor.** Sus otros tres modelos no se agregan al dimensionamiento sin fichas técnicas locales adecuadas. El adaptador reconoce también los cinco SKUs oficiales cuando exista una ficha local con ese identificador.

En búsqueda se consulta el motor existente sin filtrar por el precio histórico; se enriquece la lista con la oferta para una unidad y después se aplican precio, orden y paginación. El precio filtrado es unitario, sin flete ni descuento; la tarjeta lo aclara. No se escribe el precio observado sobre el catálogo técnico. Las fichas sin oferta no adquieren un precio inventado.

## Flujo principal y respaldo

Calculadora invitada, comparación temporal y estudios durables utilizan el mismo adaptador. El estudio consulta las alternativas factibles y elige el menor **costo total comercial** disponible, con desempate por el orden técnico existente. Un precio local didáctico no invalida el precio real del vendedor.

Ante conexión fallida, timeout, 429, 5xx, autenticación rechazada o respuesta incompatible, se repite **todo el estudio** con el simulador local. Nunca se mezclan precios reales y simulados en una selección. Las ofertas incluyen `origen: simulado`, y `motivoSimulacion: PROVEEDOR_NO_DISPONIBLE` cuando corresponde. La página muestra **Oferta simulada: stock y precios didácticos** y explica el respaldo; el catálogo marca **Panel simulado**. El historial conserva esa procedencia aunque el proveedor se recupere después.

`200 []` y `404 PRODUCTO_NO_ENCONTRADO` indican falta de oferta para ese modelo: se intenta otra alternativa y **no** se inventa stock mediante el respaldo. `404 RUTA_NO_ENCONTRADA` es un contrato/destino incorrecto y activa simulación identificada. Datos de entrada inválidos o cancelación del usuario tampoco activan una cotización ficticia. Si falla también el simulador, se devuelve un error recuperable.

## Tiempos, cuotas y seguridad

- Timeout comercial: 1 s por llamada; sin reintentos inmediatos dentro de esa llamada.
- Circuito: tres fallos, apertura de 10 s, una prueba de recuperación. Un `429` conserva como mínimo su `Retry-After`, aunque exceda esos 10 s. No hay llamadas anticipadas durante esa espera.
- Cuota declarada por el grupo: 50 req/s sostenidas y ráfaga de 100 por clave. No se ejecutan ensayos de saturación contra su instancia.
- Cuerpo máximo de respuesta: 64 KiB, tipo JSON y campos esenciales comprobados. Campos aditivos desconocidos se toleran. No se siguen redirecciones con la clave.
- El proveedor entregó únicamente HTTP. La excepción académica se limita a **esa IP exacta** y exige `PANEL_PROVIDER_ALLOW_HTTP=true`. La clave viaja sin TLS en ese tramo; no se afirma HTTPS entre grupos. La web puede seguir accediéndose por HTTPS porque la llamada comercial sale del backend. Al recibir HTTPS, actualizar URL y desactivar esa excepción.
- Los circuitos son locales a cada réplica Solar; no constituyen un límite distribuido ni disponibilidad garantizada.

## Configuración y comprobaciones

En `.env`, fuera de Git:

```dotenv
PANEL_PROVIDER_MODE=externo
PANEL_PROVIDER_URL=http://20.226.84.78
PANEL_PROVIDER_ALLOW_HTTP=true
PANEL_COMPARATOR_API_KEY=<clave entregada por privado>
```

`PANEL_PROVIDER_API_KEY` continúa siendo la clave del simulador; no se reemplaza por la comercial. No se modifican claves de Clínica. Un clon sin la configuración externa usa el simulador por defecto y lo informa.

Pruebas: `go test -tags=nomsgpack ./services/solar/...`; el test `TestComparatorOpenAPICompatibility` revisa la copia congelada. Para validar otra copia del OpenAPI, configurar `PANEL_COMPARATOR_OPENAPI` con su ruta antes de ejecutar ese test. Se comprueban además mutaciones de campos/tipos/importes, stock vacío, errores, 429, timeout, circuito, recuperación y redirecciones.

Con el entorno operativo: `docker compose --profile verificar run --rm -e EXPECTED_PROVIDER_SOURCE=externo verificar-real python -B -m unittest discover -s tests -p test_comparator_integration.py -v`. Para los ensayos de respaldo se usa un destino local cerrado temporal y se restaura la configuración real al terminar; no se apaga la API del otro grupo.

Los contratos de la web incorporan metadatos de oferta: Solar 1.1.0, comparación invitada 1.2.0, API privada 1.2.0 y búsqueda 1.1.0. El OpenAPI **1.0.1 de Clínica permanece intacto**.
