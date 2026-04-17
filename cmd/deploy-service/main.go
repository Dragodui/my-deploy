package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"strconv"

	deploysvc "github.com/dragodui/my-deploy/internal/deploySvc"
	shareddb "github.com/dragodui/my-deploy/internal/shared/db"
	_ "github.com/lib/pq"
)

func main() {
	cfg := deploysvc.NewConfig()

	// db init
	db, err := sql.Open("postgres", cfg.DBDsn)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping db: %v", err)
	}
	log.Println("connected to postgres")

	// auto-migration
	migrationDir := "/migrations"
	if _, err := os.Stat(migrationDir); os.IsNotExist(err) {
		migrationDir = "migrations/deploy"
	}
	if err := shareddb.Migrate(db, migrationDir); err != nil {
		log.Printf("Warning: migrations failed: %v", err)
	}

	// repo, service, handler
	agentClient := deploysvc.NewAgentClient(cfg.AgentURL)
	templateClient := deploysvc.NewTemplateClient(cfg.TemplateURL)
	repo := deploysvc.NewDeployRepository(db)

	svc := deploysvc.NewDeployService(repo, agentClient, *templateClient)
	handler := deploysvc.NewDeployHandler(svc, agentClient)

	// http for gateway
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("POST /api/deployments", handler.Create)
	mux.HandleFunc("GET /api/deployments", handler.ListByAgent)
	mux.HandleFunc("GET /api/deployments/{id}", handler.GetByID)
	mux.HandleFunc("DELETE /api/deployments/{id}", handler.Delete)
	mux.HandleFunc("POST /api/deployments/{id}/start", handler.Start)
	mux.HandleFunc("POST /api/deployments/{id}/stop", handler.Stop)
	mux.HandleFunc("PATCH /api/deployments/{id}", handler.Update)

	log.Printf("Starting HTTP server on port %d...", cfg.Port)
	if err := http.ListenAndServe(":"+strconv.Itoa(cfg.Port), mux); err != nil {
		log.Fatalf("failed to start http server: %v", err)
	}
}
