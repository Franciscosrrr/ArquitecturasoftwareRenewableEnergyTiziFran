# Limpieza de datos para la entrega

El 11/11/2026 se retiraron los restos de ejecuciones anteriores sin eliminar pruebas ejecutables, fixtures necesarios, contratos ni evidencias de evaluación.

## Resultado aplicado

- Conservadas `admin@renewable.local` y `2407889@ucc.edu.ar`, con sus roles, contraseñas, sesiones y datos.
- Eliminadas 61 cuentas sintéticas y sus lugares, consumos, evaluaciones, idempotencia, outbox y estudios asociados.
- Eliminadas 9 fichas de electrodomésticos y 10 paneles generados por pruebas que no estaban referenciados por datos conservados.
- Conservadas seis fichas base de equipos y dos paneles que utiliza la integración real y su respaldo. Sus identificadores SIM y sus fuentes didácticas siguen siendo necesarios para el funcionamiento actual.
- Retirados los documentos derivados de ensayo de Meilisearch, invalidada la generación de caché y limpiada la DLQ que solo contenía mensajes sintéticos identificados. No se eliminaron colas ni exchanges.
- La huella de los registros conservados coincidió antes y después de borrar. Se verificaron el login administrador y las listas entregadas por la API.

[Resumen](evidence/limpieza-final-2026-11-11/resumen.json) y [verificación de la página](evidence/limpieza-final-2026-11-11/verificacion-api.json). Los dumps SQL y el snapshot Mongo completos se guardaron en `.runtime/respaldos-limpieza/`, excluidos de Git y del contexto de build. No se copian datos privados a las evidencias públicas.

## Qué permanece

Las suites Go/HTTP/navegador, pruebas de carga/fallas, mock histórico usado por contratos, simulador de respaldo, ADR y evidencias permanecen disponibles. Los usuarios y fichas de ejecución se generan de nuevo en cada prueba y no necesitan los registros antiguos retirados.

Se retiraron nueve archivos `doc.go` de paquetes sin implementación ni imports, tras comprobar que solo contenían un comentario y `package`. Las importaciones de los seis módulos resolvieron correctamente con las fuentes actuales. Se conservaron todos los Dockerfiles y el resto de las fuentes. La revisión automática rechazó una retirada más amplia por riesgo de afectar builds; se eligió esta limpieza acotada y comprobada.

## Repetir después de ejecutar pruebas

La herramienta opcional requiere Python 3 en el host y Docker Compose. Ese requisito no se agrega al arranque normal de la aplicación. Mostrar primero el plan desde la raíz:

```powershell
python scripts/limpiar-datos-prueba.py
```

Aplicar la limpieza ordinaria de cuentas sintéticas y fichas no utilizadas:

```powershell
python scripts/limpiar-datos-prueba.py --aplicar
```

El modo ordinario conserva cuentas ajenas a los patrones de las pruebas. La preparación final solicitada agrega `--solo-cuentas-finales`: conserva exclusivamente admin y 2407889 y borra cualquier otra cuenta. Exige que existan las dos cuentas protegidas; no es el modo apropiado para conservar clientes adicionales.

```powershell
python scripts/limpiar-datos-prueba.py --solo-cuentas-finales --aplicar
```

La aplicación y dependencias deben estar operativas antes de iniciar. Al aplicar se pausan brevemente los servicios de negocio, se respalda antes de borrar y se reanudan los que estaban activos mediante finally. Ejecutar fuera de otra prueba, carga o escritura de usuarios. Si falla una dependencia, revisar el resultado y el respaldo antes de repetir. No se borran volúmenes ni se altera Render, la VM del proveedor o WSL.

Las cuatro pruebas de `tests/test_cleanup.py` comprueban selección, referencias utilizadas, cuentas protegidas, huellas y literales SQL con datos en memoria. No generan usuarios en la aplicación. No se volvió a ejecutar la suite e2e sobre las bases limpias para evitar repoblarlas antes de presentar.
