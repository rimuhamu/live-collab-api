package db

import (
	"database/sql"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func Connect(dsn string) *sql.DB {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatal("failed to connect to db:", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal("failed to ping db:", err)
	}
	return db
}
