# Verificación de la primera entrega revisada

Versión documental 1.2.0; contratos 1.0.1; rutas, fórmulas y campos conservados, con corrección de la cota de superficie solar.

## Evidencia ejecutada

| Comprobación | Resultado y alcance |
|---|---|
| Mock HTTP local | 21 pruebas unittest aprobadas tras las revisiones de seguridad y coherencia, con servidor real en puerto efímero y solicitudes HTTP |
| Consumo de referencia | 6 + 32,4 + 9 = 47,4 kWh; promedio 1,58; respuesta comparada con fixture |
| Redondeo | Mitad hacia arriba a seis decimales; prueba de total sin redondeos intermedios |
| Contrato/validaciones | Campos extra, variantes, tipos, rangos, ids repetidos, días inválidos y precisión excesiva |
| Credenciales simuladas y errores | 401/403, JSON inválido, 413/415/422, rate limit 429 real y fallas controladas 429/500/503 |
| Invitado | Consumo sin clave; Solar manual, consumo cero, superficie insuficiente y 503 por proveedor no asignado |
| OpenAPI | Ambos documentos OpenAPI 3.0.3 y sus ejemplos embebidos validados con una biblioteca independiente; ejemplos HTTP verificados por las pruebas del mock |
| Go | go test y go vet aprobados en cinco módulos; pruebas de autenticación de salud añadidas; negocio aún sin implementar |
| Arranque Go directo | Cinco binarios: sin credencial live/ready=401; con credencial live=200 y ready=503; negocio=501; arranque sin credencial/corta rechazado; evidencia en docs/evidence/go-smoke.txt |
| Compose | .\scripts\compose.ps1 --profile estructura --profile verificar config --quiet: configuración válida |
| Documentos | Enlaces locales verificados; fuentes Mermaid y PNG renderizados desde ellas; vistas prevista y actual diferenciadas |

## Reproducción

Desde raíz:

```powershell
python -m unittest discover -s mock -p "test_*.py" -v
go test ./gateway/... ./services/usuarios/... ./services/electrodomesticos/... ./services/consumo/... ./services/solar/...
.\scripts\compose.ps1 --profile estructura --profile verificar config --quiet
```

Scripts/check.ps1 y scripts/check.sh sirven para entornos con Python y Go. Con solo Docker, el servicio verificar prueba el mock y el build del perfil estructura compila los módulos.

## Límites

- Las pruebas HTTP requieren permitir conexiones a localhost en el entorno de ejecución; la verificación actual aprobó 21 pruebas.
- Docker Desktop está instalado; se intentó iniciarlo, pero el motor Linux siguió sin estar accesible. No se ejecutaron builds ni contenedores Docker ni se afirma arranque probado en otra máquina.
- No hay pruebas contra MySQL, MongoDB, RabbitMQ o proveedor externo, ni mediciones de carga o caída controlada: siguen pendientes de ejecución.
- El validador propio cubre las restricciones usadas; no constituye certificación independiente de conformidad OpenAPI completa.
- Los PNG se renderizan desde los archivos Mermaid canónicos; [generación de diagramas](diagrams/README.md).
- No se verificaron aprobación docente ni acceso público al repositorio. La lectura remota no permitió confirmar acceso, lo que no demuestra que sea privado o inexistente. No se publicaron cambios remotos.
- El PDF y los apuntes externos no fueron modificados; los archivos locales del proyecto sí incorporan las correcciones solicitadas. No se inventaron resultados de integración, POSTMORTEM, disponibilidad cloud ni aprobación docente.

## Revisión de seguridad posterior

La tabla anterior reúne la evidencia actual del mock y los servidores iniciales. La revisión posterior protege los healthchecks del mock y de los cinco componentes Go y añade pruebas específicas. La evidencia actualizada y los límites están en [SECURITY-AUDIT.md](SECURITY-AUDIT.md). El arranque directo Go también exige HEALTHCHECK_TOKEN; live=200 y ready=503 se esperan solo con credencial operativa válida, y sin ella ambos devuelven 401.
