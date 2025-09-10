package db

import (
	"database/sql"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func Connect(dsn string) *sql.DB {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatal("failed to connect to db:", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal("failed to ping db:", err)
	}

	err = goose.SetDialect("postgres")
	if err != nil {
		return nil
	}

	// run migrations
	if err = goose.Up(db, "internal/db/migrations"); err != nil {
		log.Fatal("failed to run migrations", err)
	}

	log.Println("Migrations applied successfully")

	return db
}
