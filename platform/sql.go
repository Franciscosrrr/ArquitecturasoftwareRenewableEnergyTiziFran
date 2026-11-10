package platform

import (
	"database/sql"
	"github.com/go-sql-driver/mysql"
	"time"
)

/**
 * Abre el pool SQL con credenciales y tiempos de espera propios del servicio.
 *
 * @method OpenSQL
 * @returns {*sql.DB, error} Resultado de la operación; error si no puede completarse.
 */
func OpenSQL() (*sql.DB, error) {
	cfg := mysql.NewConfig()
	cfg.User = Env("MYSQL_USER", "")
	cfg.Passwd = Secret("MYSQL_PASSWORD")
	cfg.Net = "tcp"
	cfg.Addr = Env("MYSQL_ADDR", "mysql:3306")
	cfg.DBName = Env("MYSQL_DATABASE", "")
	cfg.ParseTime = true
	cfg.Timeout = 2 * time.Second
	cfg.ReadTimeout = 3 * time.Second
	cfg.WriteTimeout = 3 * time.Second
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	return db, nil
}
