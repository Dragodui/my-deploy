package main

import (
	"database/sql"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"

	authsvc "github.com/dragodui/my-deploy/internal/authSvc"
	shareddb "github.com/dragodui/my-deploy/internal/shared/db"
	"github.com/dragodui/my-deploy/internal/shared/http/middleware"
	authpb "github.com/dragodui/my-deploy/internal/shared/proto/authpb/proto"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
)

func main() {
	// config
	cfg := authsvc.NewConfig()

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
		// fallback for local development
		migrationDir = "migrations/auth"
	}
	if err := shareddb.Migrate(db, migrationDir); err != nil {
		log.Printf("Warning: migrations failed: %v", err)
	}

	// repo, service, handler
	userRepo := authsvc.NewUserRepository(db)
	svc := authsvc.NewAuthService(userRepo, cfg.JWTSecret)
	handler := authsvc.NewAuthHandler(svc)

	// grpc for communication between services
	lis, _ := net.Listen("tcp", ":"+strconv.Itoa(cfg.GRPCPort))
	grpcServer := grpc.NewServer()
	authpb.RegisterAuthInternalServer(grpcServer, &authsvc.AuthGRPCServer{Repo: userRepo})
	go grpcServer.Serve(lis)

	// http for gateway
	mux := http.NewServeMux()
	// health check
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("POST /api/auth/sign-up", handler.SignUp)
	mux.HandleFunc("POST /api/auth/sign-in", handler.SignIn)
	mux.Handle("GET /api/me", middleware.JWTAuth(cfg.JWTSecret)(http.HandlerFunc(handler.Me)))

	log.Printf("Starting HTTP server on port %d...", cfg.Port)
	if err := http.ListenAndServe(":"+strconv.Itoa(cfg.Port), mux); err != nil {
		log.Fatalf("failed to start http server: %v", err)
	}
}
