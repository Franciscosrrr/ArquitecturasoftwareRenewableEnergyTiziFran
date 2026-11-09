# Prompt para la IA del grupo proveedor de paneles

Copiar el texto siguiente en el chat que tenga acceso al repositorio del proveedor. Es una solicitud para acordar la integración, no un contrato ya aceptado. No contiene claves reales.

---

Actuá como arquitecto y desarrollador del proyecto proveedor de paneles solares. Analizá nuestro repositorio e implementá o completá la capacidad que consumirá el grupo Renewable Energy para su TP Integrador de Arquitectura de Software 2026. Aprovechá lo que ya existe y preservá los contratos que otros grupos estén usando.

**Nuestra necesidad comercial:** Renewable Energy estima el consumo de un lugar y compara paneles. Necesitamos consultar una cotización real por modelo/SKU, con stock y precio, para decidir qué alternativa ofrecer. La llamada sale de nuestro microservicio Solar y forma parte del cálculo principal. No necesitamos que ustedes calculen el consumo de nuestros equipos ni la tarifa eléctrica de Clínica. Cotizar no debe comprar, reservar ni descontar stock.

1. **Revisá su API antes de proponer cambios.** Si ya resuelve este caso, entreganos su contrato y adaptaremos nuestro cliente. Si falta, implementá una operación versionada de cotización por SKU y documentá sus reglas. La ruta y campos siguientes son una propuesta de interoperabilidad; no exigimos reemplazar un contrato existente ni usar /v2 porque nuestro simulador lo use.

   Propuesta de solicitud:

   ```json
   {"sku":"MODELO-550","potenciaObjetivoW":1100}
   ```

   Propuesta de respuesta:

   ```json
   {
     "cotizacionId":"cotizacion-de-ejemplo",
     "sku":"MODELO-550",
     "potenciaPanelW":550,
     "cantidad":2,
     "stockDisponible":100,
     "precioUnitarioCentavos":30000000,
     "totalCentavos":60000000,
     "moneda":"ARS"
   }
   ```

   En esta propuesta, cantidad = techo(potenciaObjetivoW / potenciaPanelW), stockDisponible debe alcanzar la cantidad y totalCentavos = cantidad × precioUnitarioCentavos. Los importes son centavos enteros, no pesos ni tarifa energética; 30000000 centavos son ARS 300000. Si su API trabaja por cantidad en lugar de potencia objetivo, indiquen el formato exacto: nosotros calcularemos la cantidad y adaptaremos el cliente.

2. **Identificación de paneles.** Entreganos al menos dos modelos existentes y cotizables con SKU estable, nombre, potencia en W, superficie efectiva por unidad con unidad explícita, fuente de ficha, moneda y precio de referencia. Puede ser un catálogo API o fixtures documentadas; indiquen cómo detectar cambios de versión/precio. Los SKU del simulador de Renewable no son identificadores de su empresa. No inventen modelos como si estuvieran disponibles.

3. **Contrato formal versionado.** Guardá OpenAPI JSON/YAML en docs/contracts/, con operaciones, autenticación, tipos, campos obligatorios, unidades, límites, reglas cruzadas, ejemplos válidos/inválidos y errores. Definí qué ocurre con SKU inexistente, falta de stock, entrada inválida, credencial ausente o sin permiso, cuota excedida y caída del servicio. Incluí códigos HTTP y cuerpo de error estables; para 429, Retry-After. Informá si una cotización tiene vencimiento y qué garantiza sobre el stock. No cambies una versión publicada sin avisar y acordar la transición.

4. **Acceso y disponibilidad.** Necesitamos URL HTTPS pública verificable, enlace al OpenAPI, versión/commit desplegado y fecha hasta la cual quedará operativo durante la evaluación. Autenticación M2M por API key/Bearer o el mecanismo existente, con permiso para cotizar/consultar stock. Entregá las claves por privado; nunca en Git, frontend, OpenAPI, ejemplos ni este mensaje. Si es posible, otra credencial sin permiso para probar 403.

5. **Tiempos y fallas.** Nuestro cliente aplica aproximadamente 1 segundo por llamada, circuit breaker y reintentos asíncronos acotados. Informá latencia caliente, cold start, cuotas y tiempos que realmente pueden sostener para acordarlos. Necesitamos un modo de ensayo coordinado para SKU sin oferta, stock insuficiente, indisponibilidad y timeout. No simules éxito, stock o precio cuando el proveedor falle.

6. **Pruebas y entrega.** Verificá implementación contra el contrato y sus ejemplos. Entreganos ejemplos reproducibles de éxito y error para construir nuestro test consumidor capaz de detectar cambios incompatibles en campos, tipos, unidades y semántica. Las pruebas de ustedes no sustituyen nuestra prueba de contrato ni la integración entre grupos. Al finalizar, devolvé un mensaje listo para compartir con URL, OpenAPI/version, operación/ejemplos, autenticación, modelos/SKU, errores, cuotas, tiempos y disponibilidad. Indicá cualquier pendiente real; no inventes URLs, claves, resultados ni compromisos.

Nosotros reemplazaremos el adaptador simulado por el de su contrato y comprobaremos el flujo completo. No contacten automáticamente a otros grupos ni envíen secretos desde la IA.
