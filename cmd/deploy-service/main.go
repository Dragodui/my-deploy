package main

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"

	deploysvc "github.com/dragodui/my-deploy/internal/deploySvc"
	_ "github.com/lib/pq"
)

func main() {
	cfg := deploysvc.NewConfig()

	// db
	db, err := sql.Open("postgres", cfg.DBDsn)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping db: %v", err)
	}
	log.Println("connected to postgres")

	// agent gRPC client
	agentClient := deploysvc.NewAgentClient(cfg.AgentURL)

	// template http clien
	templateClient := deploysvc.NewTemplateClient(cfg.TemplateURL)

	// repo, service, handler
	repo := deploysvc.NewDeployRepository(db)

	svc := deploysvc.NewDeployService(repo, agentClient, *templateClient)
	handler := deploysvc.NewDeployHandler(svc, agentClient)

	// HTTP
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/deployments", handler.Create)
	mux.HandleFunc("GET /api/deployments", handler.ListByAgent)
	mux.HandleFunc("GET /api/deployments/{id}", handler.GetByID)
	mux.HandleFunc("DELETE /api/deployments/{id}", handler.Delete)
	mux.HandleFunc("POST /api/deployments/{id}/start", handler.Start)
	mux.HandleFunc("POST /api/deployments/{id}/stop", handler.Stop)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	log.Printf("deploy service starting on port %d", cfg.Port)
	http.ListenAndServe(":"+strconv.Itoa(cfg.Port), mux)
}
