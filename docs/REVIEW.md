# Verificación de la primera entrega revisada

Fecha: 8 de octubre de 2026. Versión documental 1.2.0; contratos 1.0.0 sin cambios de operaciones/esquemas.

## Evidencia ejecutada

| Comprobación | Resultado y alcance |
|---|---|
| Mock HTTP local | 16 pruebas unittest aprobadas, con servidor real en puerto efímero y solicitudes HTTP |
| Consumo de referencia | 6 + 32,4 + 9 = 47,4 kWh; promedio 1,58; respuesta comparada con fixture |
| Redondeo | Mitad hacia arriba a seis decimales; prueba de total sin redondeos intermedios |
| Contrato/validaciones | Campos extra, variantes, tipos, rangos, ids repetidos, días inválidos y precisión excesiva |
| Credenciales simuladas y errores | 401/403, JSON inválido, 413/415/422, rate limit 429 real y fallas controladas 429/500/503 |
| Invitado | Consumo sin clave; Solar manual, consumo cero, superficie insuficiente y 503 por proveedor no asignado |
| OpenAPI | JSON legible, referencias locales resueltas y ejemplos embebidos contrastados con el subconjunto de esquemas usado |
| Go | go test compila cinco módulos y paquetes; no hay tests de negocio Go porque esos casos no están implementados |
| Arranque Go directo | Cinco binarios iniciados localmente: live=200, ready=503 y negocio=501; evidencia en docs/evidence/go-smoke.txt |
| Compose | docker compose --profile estructura --profile verificar config --quiet: configuración válida |
| Documentos | Enlaces Markdown locales existentes; PNG inspeccionados; Mermaid alineado con la misma vista simplificada |

## Reproducción

Desde raíz:

```sh
python -m unittest discover -s mock -p "test_*.py" -v
go test ./gateway/... ./services/usuarios/... ./services/electrodomesticos/... ./services/consumo/... ./services/solar/...
docker compose --profile estructura --profile verificar config --quiet
```

Scripts/check.ps1 y scripts/check.sh sirven para entornos con Python y Go. Con solo Docker, el servicio verificar prueba el mock y el build del perfil estructura compila los módulos.

## Límites

- La primera ejecución HTTP estuvo bloqueada por permisos locales del entorno; después de conceder acceso de red, la ejecución final aprobó las 16 pruebas.
- Docker Desktop está instalado; se intentó iniciarlo, pero el motor Linux siguió sin estar accesible. No se ejecutaron builds ni contenedores Docker ni se afirma arranque probado en otra máquina.
- No hay pruebas contra MySQL, MongoDB, RabbitMQ o proveedor externo, ni mediciones de carga o caída controlada: corresponden a hitos futuros.
- El validador propio cubre las restricciones usadas; no constituye certificación independiente de conformidad OpenAPI completa.
- Se inspeccionaron PNG; no se ejecutó un renderizador Mermaid. Fuentes e imágenes son representaciones equivalentes mantenidas manualmente.
- No se verificaron aprobación docente ni acceso público al repositorio. La lectura remota no permitió confirmar acceso, lo que no demuestra que sea privado o inexistente. No se publicaron cambios remotos.
- Los originales recibidos no fueron modificados. No se inventaron resultados de integración, POSTMORTEM, disponibilidad cloud ni aprobación docente.
