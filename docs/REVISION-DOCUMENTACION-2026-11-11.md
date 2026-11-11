# Revisión de documentación y comentarios — 11/11/2026

## Resultado

Se contrastaron el enunciado, la documentación vigente y el código actual. Se completaron comentarios, se corrigieron explicaciones incompatibles con sus retornos y se actualizaron los textos de la integración comercial. La lógica del sistema y el contrato 1.0.1 de Clínica se conservan. La revisión es estática; las pruebas funcionales previas se identifican por separado.

## Correcciones

- Se agregó un [índice documental](README.md), [configuración](CONFIGURACION.md), [guía de comentarios](DOCUMENTACION-CODIGO.md) y [cierre de entrega](ENTREGA.md).
- README, arquitectura, SPEC, modo invitado, servicio Solar y contratos distinguen proveedor real, respaldo identificado y superficies/HSP didácticas. Se separan resultados actuales e históricos.
- El [postmortem](POSTMORTEM.md) incorpora el ensayo de conexión comercial rechazada, su impacto, secuencia, detección, causa y restauración. No atribuye una caída provocada a la VM ajena.
- Se agregaron ejemplos de oferta real y respaldo adaptados por Solar. El contrato M2M congelado no se modifica para cambiar la URL anunciada.
- Los enlaces de la hoja copiada del proveedor se resolvieron contra su repositorio original. No se reescribieron sus resultados como si fueran pruebas nuestras.
- Se completaron métodos de interfaces y documentación de estructuras. Se eliminaron comentarios de paquete que decían «pendiente» cuando el código efectivo ya existía o estaba en otra ubicación.
- Se corrigieron comentarios genéricos de argumentos y retornos Go. En Python se precisaron resultados sin valor, tuplas, diccionarios y credenciales de prueba; se conserva JSDoc dentro del docstring.

## Cobertura y verificaciones

La [evidencia procesable](evidence/documentacion-2026-11-11/resumen.json) registra conteos, resultados y alcance. Se revisan declaraciones nombradas de producción y pruebas con AST de Go/Python y parser JavaScript/JSX. También se comprueban referencias locales, versiones, ejemplos JSON y la integridad del contrato compartido.

Cobertura: 244 funciones Go y 8 métodos de interfaces, 68 tipos Go documentados, 37 unidades nombradas JavaScript y 106 unidades Python (93 funciones y 13 clases). La comprobación final no detectó faltantes de JSDoc en ese alcance, enlaces locales o anclas rotas, ni valores de credenciales locales en los archivos compartibles revisados. Las firmas Go conservaron sus parámetros y retornos. Aprobaron los dos controles de integridad/referencias de contratos y los cuatro ejemplos de solicitud/respuesta Solar. [Validación de ejemplos](evidence/documentacion-2026-11-11/contratos-ejemplos.json).

Los callbacks inline se revisan como parte de sus operaciones contenedoras; no se afirma que cada lambda tenga un JSDoc propio. Los comentarios de tipos y paquetes describen responsabilidades y no sustituyen un contrato de operación. Los Markdown generados de trazas/errores y las evidencias inmutables se conservan como registros, sin corregir retrospectivamente sus resultados.

## ADR

Se verificaron D1–D13: un archivo canónico por número y presencia de contexto, alternativas, consecuencias y justificación. Esta revisión no introduce otra decisión arquitectónica ni crea un ADR sustituto. Las actualizaciones anteriores de D9/D10 del proveedor real permanecen dentro de sus archivos originales. El [índice ADR](adr/README.md) indica las revisiones aplicables.

## Límites y cierre

No se certifica que la explicación textual sustituya pruebas de ejecución. Las verificaciones de negocio del proveedor real, respaldo, navegador y carga conservan sus fechas y escenarios. Los paquetes vacíos de diseño se identifican como tales, sin presentarlos como capas ejecutables adicionales. Los ensayos antiguos de fallas que detenían el simulador requieren su escenario original y no acreditan una caída del proveedor real.

Los nuevos cambios siguen locales hasta revisar e integrar la entrega en main. La ficha Word debe completarse con los integrantes, JTP, grupo proveedor ya confirmado y el hash final publicado. Mantener las claves fuera de Git.
