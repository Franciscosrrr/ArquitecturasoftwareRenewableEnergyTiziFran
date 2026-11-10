# Servicio Usuarios

Estado 2026-11-09. Implementación Go/Gin en capas: transporte HTTP, identidad y persistencia en MySQL usuarios_db. Registro y login usan Argon2id con salt; el access token es JWT y el refresh se conserva como hash en una sesión durable con vencimiento. Refresh rota la credencial y logout revoca la sesión. No se devuelven contraseñas ni sus hashes.

El gateway consulta identidad para verificar sesión y rol vigentes. Perfil y administración exigen credencial interna/actor; gestionar roles requiere administrador y no permite perder el último administrador. La cookie de refresh es HttpOnly/SameSite Strict y la entrada web HTTPS agrega Secure sin afectar localhost. Las operaciones de autenticación verifican Origin y aplican límites de solicitudes/cómputo.

Migrate crea tablas e índices repetiblemente; el administrador inicial se crea con la contraseña de entorno generada por los scripts, sin sobrescribir usuarios existentes. /health y /metrics requieren HEALTHCHECK_TOKEN; readiness comprueba MySQL. El arranque del entorno completo usa scripts/compose, no el perfil histórico de estructura.

[Arquitectura](../../docs/ARCHITECTURE.md), [contrato privado](../../docs/contracts/private-v1/README.md), [verificación](../../docs/VERIFICACION-FINAL.md).
