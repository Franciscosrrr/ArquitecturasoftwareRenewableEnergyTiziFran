# Documentación del código

La revisión del 11/11/2026 contrasta comentarios con firmas y comportamiento, además de comprobar que existan. El inventario y sus límites están en [la auditoría](REVISION-DOCUMENTACION-2026-11-11.md).

## Formato solicitado

Funciones y métodos nombrados utilizan el formato JSDoc indicado por el equipo:

```javascript
/**
 * Calcula la energía de los equipos con sus parámetros de utilización.
 *
 * @method calcularConsumo
 * @param {Object} entrada - Equipos, cantidades y período a evaluar.
 * @returns {Object} Desglose por equipo y energía total estimada.
 */
```

Incluir un `@param` por parámetro recibido y un `@returns` que describa lo que efectivamente devuelve la función. No anunciar un error como valor retornado cuando la firma solo devuelve un número, texto o booleano. Explicar unidades, límites relevantes, efectos persistentes, errores y procedencia de datos cuando son necesarios para utilizar la operación.

## Aplicación por lenguaje

- **Go:** bloque JSDoc adyacente a la declaración. Los métodos de interfaces documentan el contrato del puerto. Los modelos/estructuras describen su responsabilidad, propiedad y unidades; una estructura de datos no constituye por sí sola una operación de negocio.
- **JavaScript/React:** JSDoc en las funciones nombradas, incluidos componentes, helpers y pruebas. Los objetos de propiedades desestructuradas se documentan como `props` o `context`, con los campos relevantes cuando corresponda.
- **Python:** el mismo bloque JSDoc se conserva dentro del docstring de la función/clase, que es el contenedor sintáctico válido del lenguaje. `self`/`cls` representan al receptor. Los métodos sin retorno de valor documentan `None`; las funciones que lanzan una excepción lo explican en la descripción del resultado.
- **Scripts de operación:** se explica su propósito en [configuración](CONFIGURACION.md) y [entrega](ENTREGA.md); los comentarios del lenguaje describen la secuencia y el manejo de secretos. No se inserta sintaxis JavaScript ejecutable en PowerShell, shell o Dockerfiles.

Los callbacks inline de UI, rutas y pruebas se revisan en el contexto de la función que los contiene; no se cuentan como funciones nombradas con un contrato público independiente. El informe distingue ese alcance: no afirma que cada lambda anónima tenga un bloque JSDoc propio. Los comentarios de paquete indican la implementación efectiva y distinguen las ubicaciones de diseño que siguen vacías.

## Reglas para mantenerlos

Al modificar una firma, revisar parámetros y retorno del comentario en el mismo cambio. Evitar frases como «valor requerido por la operación» sin explicar el significado del argumento. No incluir credenciales reales en comentarios o ejemplos. Al cambiar un contrato, versionar su documento y revisar ejemplos/pruebas; la capacidad compartida 1.0.1 de Clínica permanece congelada. Los comentarios no sustituyen el OpenAPI, la documentación operativa ni las decisiones con su contexto y porqué.

La auditoría emplea AST de Go y Python y parser de JavaScript/JSX, además de revisión de textos, enlaces y tipos retornados. Es una comprobación estática y de consistencia documental; las pruebas funcionales existentes se identifican por separado en [verificación](VERIFICACION-FINAL.md).
