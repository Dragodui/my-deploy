package main

import (
	"database/sql"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"

	agentsvc "github.com/dragodui/my-deploy/internal/agentSvc"
	shareddb "github.com/dragodui/my-deploy/internal/shared/db"
	agentpb "github.com/dragodui/my-deploy/internal/shared/proto/agentpb/proto"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
)

func main() {
	// config
	cfg := agentsvc.NewConfig()

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
		migrationDir = "migrations/agent"
	}
	if err := shareddb.Migrate(db, migrationDir); err != nil {
		log.Printf("Warning: migrations failed: %v", err)
	}

	// repo, service, handler
	agentRepo := agentsvc.NewAgentRepository(db)
	svc := agentsvc.NewAgentService(agentRepo)
	handler := agentsvc.NewAgentHandler(svc)
	registry := agentsvc.NewAgentRegistry()
	wsHandler := agentsvc.NewWSHandler(registry, agentRepo)

	// grpc for communication between services
	lis, _ := net.Listen("tcp", ":"+strconv.Itoa(cfg.GRPCPort))
	grpcServer := grpc.NewServer()
	agentpb.RegisterAgentInternalServer(grpcServer, &agentsvc.AgentGRPCServer{Registry: registry, Repo: agentRepo})
	go grpcServer.Serve(lis)

	// http for gateway
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("POST /api/agent", handler.RegisterOrGet)
	mux.HandleFunc("GET /api/agents", handler.ListByUser)
	mux.HandleFunc("POST /api/agent/bootstrap", handler.CreateBootstrapToken)
	mux.HandleFunc("POST /api/agent/bootstrap/exchange", handler.ExchangeBootstrapToken)
	mux.HandleFunc("GET /ws/agent", wsHandler.HandleAgentWS)

	log.Printf("Starting HTTP server on port %d...", cfg.Port)
	if err := http.ListenAndServe(":"+strconv.Itoa(cfg.Port), mux); err != nil {
		log.Fatalf("failed to start http server: %v", err)
	}
}
