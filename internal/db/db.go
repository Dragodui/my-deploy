package db

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"sort"

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

func RunMigration(migrationDir string, db *sql.DB) error {
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS schema_migrations (
		name TEXT PRIMARY KEY,
		applied_at TIMESTAMP DEFAULT now()
	)`)
	if err != nil {
		return err
	}
	files, err := os.ReadDir(migrationDir)

	if err != nil {
		return err
	}

	if len(files) == 0 {
		return nil
	}

	// sort migrations
	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	for _, file := range files {
		var exists bool
		err := db.QueryRow(
			"SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE name=$1)",
			file.Name(),
		).Scan(&exists)

		if err != nil {
			return err
		}

		if exists {
			log.Printf("skip migration: %s", file.Name())
			continue
		}
		log.Printf("running migration: %s", file.Name())
		sql, err := os.ReadFile(filepath.Join(migrationDir, file.Name()))
		if err != nil {
			return err
		}

		if _, err := db.Exec(string(sql)); err != nil {
			return err
		}

		_, err = db.Exec(
			"INSERT INTO schema_migrations (name) VALUES ($1)",
			file.Name(),
		)
		if err != nil {
			return err
		}
	}

	log.Println("Migrations success")
	return nil
}
