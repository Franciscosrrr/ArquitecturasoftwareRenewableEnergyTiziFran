# Control de primera entrega: 9 de octubre de 2026

Versión 1.2.0. Alcance: diseño, contratos, mock local y estructura inicial compilable. La versión anterior era exclusivamente documental.

| Requisito | Evidencia | Estado |
|---|---|---|
| README y alcance | README.md y SPEC.md | Incluidos |
| Arquitectura y datos propios | ARCHITECTURE.md y ADR-001 | Incluidos |
| Contexto y contenedores | PNG y Mermaid en diagrams | Incluidos; representación simplificada equivalente |
| Capacidad y contrato | contracts/v1/openapi.json y README | Incluidos |
| Contrato con mock | mock/server.py, pruebas, Compose, MOCK.md | Implementado localmente; ejecución Docker pendiente de validar |
| Estructura y dependencias | Cinco módulos Go, paquetes internos y Dockerfiles | Compilan; dependencias de negocio aún previstas |
| D1 y D8 | ADR-001 y ADR-008 | Adoptados para el diseño; implementación final pendiente |
| D3 y D5 iniciales | ADR-003 y ADR-005 | Incluidos |
| D2 adicional | ADR-002 | Incluido como propuesta |
| Repositorio público y dominio aprobado | URL informada; aprobación externa | No verificados; requieren confirmación del equipo/cátedra |

## Lo que falta fuera de este paquete

Publicar estos archivos en el repositorio del equipo y comprobar su acceso; confirmar aprobación del dominio y número de grupo/comisión; ejecutar el build/arranque Docker con el motor activo. No se atribuye aprobación docente ni publicación remota a la revisión local.

El mock resuelve el hito de simulación del contrato, no el servicio real de entrega 2 ni la integración con otro grupo. Frontend, BD, broker, autenticación real, balanceo, pruebas de carga y POSTMORTEM corresponden al desarrollo posterior según cronograma.

Ver [REVIEW](REVIEW.md), [revisión de propuesta](REVISION-PROPUESTA.md) y [estructura](STRUCTURE.md).
