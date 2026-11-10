# Inicialización de Usuarios

El esquema ejecutable está en [Store.Migrate](../internal/identity/identity.go): crea usuarios, sesiones e índices mediante CREATE TABLE IF NOT EXISTS. La semilla administrativa se ejecuta sin sobrescribir cuentas. La carpeta no contiene un segundo mecanismo de migración; no se afirma que la persistencia siga pendiente. No borrar volúmenes para aplicar cambios.
