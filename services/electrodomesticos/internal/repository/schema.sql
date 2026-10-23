CREATE TABLE IF NOT EXISTS electrodomesticos (
 id VARCHAR(64) PRIMARY KEY,
 nombre VARCHAR(120) NOT NULL,
 categoria VARCHAR(64) NOT NULL,
 modo ENUM('POTENCIA','CICLOS') NOT NULL,
 potencia_w DECIMAL(14,6) NOT NULL DEFAULT 0,
 energia_ciclo DECIMAL(14,6) NOT NULL DEFAULT 0,
 factor DECIMAL(8,6) NOT NULL DEFAULT 1,
 fuente VARCHAR(200) NOT NULL,
 activo BOOLEAN NOT NULL DEFAULT TRUE,
 version BIGINT NOT NULL DEFAULT 1,
 INDEX idx_activo_nombre (activo,nombre,id),
 INDEX idx_categoria_activo (categoria,activo,id)
) ENGINE=InnoDB;
