# Experiencia de cliente — 2026-11-09

Cambios aplicados a la web local, manteniendo el contrato M2M 1.0.1 de Clínica. Recargar http://localhost:8080 con Ctrl+F5 para cargar los nuevos archivos.

## Cambios solicitados

1. Avisos, selector de lugar y títulos de consumos muestran nombres sin UUID ni v1/v2. Las referencias y versiones permanecen en fichas e historial, cerrados cuando no hacen falta.
2. Archivar/restaurar usa PATCH de estado y no incrementa la versión de contenido ni recalcula el snapshot. Las versiones anteriores no se renumeraron.
3. Eliminar consumo quita el guardado de las listas y evita reabrirlo/confirmarlo. Eliminar lugar exige que no tenga consumos guardados, incluidos los archivados. Se pide confirmación y se conservan los resultados históricos. Es eliminación lógica, no borrado físico de auditoría.
4. La búsqueda de la interfaz corresponde a paneles: texto, potencia mínima/máxima, superficie máxima por unidad, moneda, precio máximo, orden y páginas. Se reutiliza Meilisearch con un índice separado y Mongo de respaldo. El selector de equipos sigue disponible para armar consumos; el API anterior se conserva.
5. Los botones se explican abajo y la comparación avanzada queda plegada en el historial.

| Antes | Ahora | Utilidad |
|---|---|---|
| Confirmar evaluación | Calcular paneles | Congela el consumo/lugar y solicita el análisis solar; permite conservar resultados sin que posteriores cambios los alteren. Necesario para ese flujo durable del TP. |
| Archivar | Archivar / Restaurar | Organización opcional, reversible y sin crear una versión de contenido. Mostrar consumos archivados permite encontrarlos. |
| Solicitar estudio con estos supuestos | Comparar con otras condiciones solares → Calcular otra alternativa | Opcional: reutiliza el consumo histórico con otras horas de sol/rendimiento/cobertura; conserva el resultado anterior. Las condiciones se eligen en Ajustar condiciones para calcular paneles. |
| Sin eliminación | Eliminar consumo / Eliminar lugar | Quita guardados con confirmación y conserva el historial. |

## Contratos y arquitectura

[Privado 1.1.0](contracts/private-v1/openapi.json) añade PATCH /lugares/{id}/archivo, PATCH /configuraciones/{id}/archivo y DELETE sobre ambos recursos, con versión requerida. Ambos PATCH usan archivado booleano; los DTO existentes mantienen archivado/archivada. 204 no tiene cuerpo. Un lugar ocupado devuelve 409 LUGAR_CON_CONSUMOS; otro propietario recibe 404. La migración añade eliminado_at sin borrar filas existentes.

[Búsqueda solar 1.0.0](contracts/panel-search-v1/openapi.json): GET /api/v1/publico/paneles/busqueda. precioMax usa unidades de moneda; superficieMax son m² por panel. Comparar precios exige moneda, y los desconocidos quedan fuera de ese filtro/orden. Se omiten filtros vacíos. X-Panel-Source distingue meilisearch/mongo-degradado para operación, sin mostrar nombres de infraestructura al cliente. POST /admin/paneles/reindexar marca todas las fichas para volver a indexar. La marca pendiente comparte la escritura Mongo con la ficha; los leases coordinan las dos réplicas. El dominio y el vendedor mantienen validación canónica, independiente de un índice eventualmente desactualizado.

No se añadieron servicios ni contenedores. Se conservan las alternativas de clase y se amplían los ADR existentes: D4 para consistencia/eliminación, D6 para índice solar y ADR-GUEST para la interfaz. El contrato M2M compartido, URL y claves de Clínica no cambian.

## Verificación

- Go: los módulos Consumo, Solar y Gateway aprobaron, incluidas fronteras del nuevo buscador.
- [HTTP real](evidence/experiencia-cliente-2026-11-09/http.txt): 19 pruebas aprobadas; archivo/restauración, borrado, propiedad, historial inmutable y sincronización solar incluidos.
- [Navegador](evidence/experiencia-cliente-2026-11-09/e2e/resultados.json): 12 aprobadas, sin fallos ni flaky; escritorio y móvil.
- El [primer intento](evidence/experiencia-cliente-2026-11-09/e2e-primer-intento.txt) detectó 429 al recargar toda la cuenta tras cada botón. Se redujeron lecturas por acción y se respetó Retry-After; la repetición completa pasó conservando los límites.
- Capturas revisadas: cuenta móvil y búsqueda solar de escritorio. No hay desbordamiento horizontal; las tablas históricas se desplazan dentro de su contenedor.

La [comprobación de respaldo y recuperación](evidence/experiencia-cliente-2026-11-09/respaldo-y-publicacion.json) confirmó HTTP 200 con Mongo al detener Meilisearch. Tras eliminar únicamente el índice derivado de paneles, el worker lo reconstruyó automáticamente en 29.407 s. Las fichas canónicas no se eliminaron. La API HTTPS de Clínica respondió 200 y su OpenAPI mantuvo el hash congelado.

También se probó archivar el lugar seleccionado: permanece identificado como archivado, no cambia a otro lugar y bloquea guardar/calcular hasta restaurarlo o elegir uno activo. La prueba adicional detectó un nombre accesible ambiguo en el selector; se corrigió y la ejecución completa de 12 casos aprobó.

La revisión encontró Jaeger detenido por OOM, con retención en memoria sin límite. Se mantuvo su contenedor de 128 MiB y se limitaron retención, cola y workers. La [sonda de retención](evidence/experiencia-cliente-2026-11-09/jaeger-recuperacion.json) envió 1600 trazas y observó 506 de ese servicio, dentro del máximo global de 512. Una nueva evaluación completó su estudio y produjo una [traza entre seis servicios](evidence/experiencia-cliente-2026-11-09/observabilidad.json). No se modificó WSL.

La [muestra de recursos posterior](evidence/experiencia-cliente-2026-11-09/recursos.json) incluye el sistema completo y los dos contenedores aislados de publicación: 20 contenedores, 1646.5 MiB sumados; la RAM del host incluye WSL y las otras aplicaciones y no equivale a esa suma. Es una muestra puntual.
