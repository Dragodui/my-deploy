package db

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

func New(dsn string) *sql.DB {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping db: %v", err)
	}

	log.Println("connected to postgres")
	return db
}
