package main

import (
	"database/sql"
	"log"
	"net"
	"net/http"
	"strconv"

	agentsvc "github.com/dragodui/my-deploy/internal/agentSvc"
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
	mux.HandleFunc("POST /api/agent", handler.RegisterOrGet)
	mux.HandleFunc("GET /api/agents", handler.ListByUser)
	mux.HandleFunc("GET /ws/agent", wsHandler.HandleAgentWS)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	log.Printf("agent service: HTTP :%d, gRPC :%d", cfg.Port, cfg.GRPCPort)
	http.ListenAndServe(":"+strconv.Itoa(cfg.Port), mux)
}
