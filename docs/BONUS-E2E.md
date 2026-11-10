# Bonus: navegador contra el sistema real

Estado actualizado 2026-11-09. Suite completa: **12 casos en Chromium**, seis por tamaño, escritorio 1280×900 y móvil 390×844. Se ejecuta con un solo worker, sin retries automáticos y manteniendo las cuotas reales.

Por tamaño se comprueban tres escenarios de calculadora, uno de cuenta/guardados/historial, uno de búsqueda de paneles y uno de cotización comercial. El caso de cuenta crea lugar/consumo, archiva y restaura sin nueva versión, calcula paneles, observa el estudio durable, edita a v2 conservando el resultado v1, recarga y restaura la sesión, elimina los guardados y cierra sesión. El caso de paneles verifica filtros, ficha técnica opcional y ausencia de desborde horizontal.

Resultado de la suite completa: **12 aprobados, 0 fallos, 0 omitidos y 0 flaky**. [JSON](evidence/experiencia-cliente-2026-11-09/e2e/resultados.json), [cuenta escritorio](evidence/experiencia-cliente-2026-11-09/e2e/cuenta-escritorio.png), [cuenta móvil](evidence/experiencia-cliente-2026-11-09/e2e/cuenta-movil.png), [búsqueda móvil](evidence/experiencia-cliente-2026-11-09/e2e/paneles-movil.png).

La publicación web HTTPS tuvo una comprobación adicional de **cuatro casos**, cuenta y paneles en ambos tamaños, aprobados sobre el enlace público. Es un subconjunto de la suite, no otras cuatro funcionalidades ni una ejecución de los doce sobre cloud. [Reporte HTTPS](evidence/publicacion-web-2026-11-09/e2e/resultados.json). El backend seguía ejecutándose en la PC tras Cloudflare; no se interpreta ese ensayo como alojamiento del código en la nube.

Los casos de cálculo y cuenta usan Go/MySQL/Mongo/Rabbit/vendedor reales del entorno. Solo la última respuesta de error de cotización se intercepta para comprobar su presentación en el navegador; la caída del proceso vendedor se verifica aparte en [POSTMORTEM](POSTMORTEM.md). Se distingue esa dependencia simulada del grupo proveedor real, cuyo contrato no se recibió.

Imagen oficial Playwright 1.64.1. scripts/verify.ps1 o sh scripts/verify.sh inicia el sistema y verifica secuencialmente. La aprobación y puntuación del bonus corresponden a la cátedra.

## Antecedente conservado

La suite del incremento anterior tenía diez casos: [reporte histórico de 10 aprobados](evidence/final-2026-11-09/e2e/resultados.json). La ampliación agregó búsqueda solar y extendió el flujo de cuenta. No se borran esos resultados ni se los presenta como la suite actual.
