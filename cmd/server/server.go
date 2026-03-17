package server

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/dragodui/my-deploy/internal/config"
	"github.com/dragodui/my-deploy/internal/db"
	myhttp "github.com/dragodui/my-deploy/internal/http"
	"github.com/dragodui/my-deploy/internal/http/handler"
	"github.com/dragodui/my-deploy/internal/http/middleware"
	"github.com/dragodui/my-deploy/internal/registry"
	"github.com/dragodui/my-deploy/internal/repository"
	"github.com/dragodui/my-deploy/internal/service"
	"github.com/dragodui/my-deploy/internal/templates"
)

func NewServer(cfg *config.Config) *http.ServeMux {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	database := db.New(cfg.DBDSN)
	db.RunMigration(filepath.Join(dir, "migrations"), database)

	templatesDir := filepath.Join(dir, "internal", "templates")
	tplRegistry, err := templates.NewTemplatesRegistry(templatesDir)
	if err != nil {
		log.Fatal(err)
	}
	agentRegistry := registry.New()

	// auth
	userRepo := repository.NewUserRepository(database)
	authService := service.NewAuthService(userRepo, cfg.JWTSecret)
	authHandler := handler.NewAuthHandler(authService)

	// agent
	agentRepo := repository.NewAgentRepository(database)
	agentService := service.NewAgentService(agentRepo)
	agentHandler := handler.NewAgentHandler(agentService)

	// deploy
	deployRepo := repository.NewDeployRepository(database)
	deployService := service.NewDeployService(deployRepo, agentRegistry, tplRegistry)
	deployHandler := handler.NewDeployHandler(deployService, agentService)
	wsHandler := myhttp.NewWSHandler(agentRegistry)

	// templates
	templatesService := service.NewTemplateService(tplRegistry)
	templatesHandler := handler.NewTemplatesHandler(templatesService)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	jwtAuth := middleware.JWTAuth(cfg.JWTSecret)
	agentTokenAuth := middleware.AgentAuth(agentRepo)
	mux.Handle("GET /ws/agent", agentTokenAuth(http.HandlerFunc(wsHandler.HandleAgentWS)))

	// auth
	mux.HandleFunc("POST /api/auth/sign-up", authHandler.SignUp)
	mux.HandleFunc("POST /api/auth/sign-in", authHandler.SignIn)
	mux.Handle("GET /api/me", jwtAuth(http.HandlerFunc(authHandler.Me)))

	// agent
	mux.Handle("POST /api/agent", jwtAuth(http.HandlerFunc(agentHandler.RegisterOrGet)))
	mux.Handle("GET /api/agents", jwtAuth(http.HandlerFunc(agentHandler.ListByUser)))

	// deploy
	mux.Handle("POST /api/deployments", jwtAuth(http.HandlerFunc(deployHandler.Create)))
	mux.Handle("GET /api/deployments", jwtAuth(http.HandlerFunc(deployHandler.ListByAgent)))
	mux.Handle("GET /api/deployments/{id}", jwtAuth(http.HandlerFunc(deployHandler.GetByID)))
	mux.Handle("DELETE /api/deployments/{id}", jwtAuth(http.HandlerFunc(deployHandler.Delete)))
	mux.Handle("POST /api/deployments/{id}/stop", jwtAuth(http.HandlerFunc(deployHandler.Stop)))
	mux.Handle("POST /api/deployments/{id}/start", jwtAuth(http.HandlerFunc(deployHandler.Start)))

	// templates
	mux.Handle("GET /api/templates", jwtAuth(http.HandlerFunc(templatesHandler.GetAll)))

	return mux
}
