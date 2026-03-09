package server

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/dragodui/my-deploy/internal/config"
	"github.com/dragodui/my-deploy/internal/docker"
	"github.com/dragodui/my-deploy/internal/repository"
	"github.com/dragodui/my-deploy/internal/service"
	"github.com/dragodui/my-deploy/internal/templates"
)

func NewServer(cfg *config.Config) *http.ServeMux {
	docker := docker.NewDockerClient(cfg)
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	templatesDir := filepath.Join(dir, "internal", "templates")
	templates, err := templates.NewTemplatesRegistry(templatesDir)
	if err != nil {
		log.Fatal(err)
	}
	// !!! CHANGE TO REAL DB
	deployRepo := repository.NewDeployRepository(&sql.DB{})
	deployService := service.NewDeployService(deployRepo, docker, templates)

	server := http.NewServeMux()
	server.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	return server
}
