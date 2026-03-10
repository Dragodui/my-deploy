package server

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/dragodui/my-deploy/internal/config"
	"github.com/dragodui/my-deploy/internal/db"
	myhttp "github.com/dragodui/my-deploy/internal/http"
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

	deployRepo := repository.NewDeployRepository(database)
	agentRegistry := registry.New()
	deployService := service.NewDeployService(deployRepo, agentRegistry, tplRegistry)
	wsHandler := myhttp.NewWSHandler(agentRegistry)

	_ = deployService // TODO: use in HTTP handlers

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})
	mux.HandleFunc("GET /ws/agent", wsHandler.HandleAgentWS)

	return mux
}
