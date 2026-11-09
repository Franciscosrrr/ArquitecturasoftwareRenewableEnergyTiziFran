# Inicialización de Consumo

[Store.Migrate](../internal/private/store.go) crea lugares, configuraciones, evaluaciones, idempotencia y outbox. Verifica information_schema antes de añadir eliminado_at a lugares/configuraciones; preserva filas y snapshots. No existe aquí un segundo juego de scripts ni una migración destructiva para reiniciar la base.
