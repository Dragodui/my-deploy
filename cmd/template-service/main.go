package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	templatesvc "github.com/dragodui/my-deploy/internal/templateSvc"
)

func main() {
	portStr := os.Getenv("PORT")
	if portStr == "" {
		portStr = "8084"
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		log.Fatal("incorrect port value")
	}

	templatesDir := os.Getenv("TEMPLATES_DIR")
	if templatesDir == "" {
		templatesDir = "/templates"
	}

	registry, err := templatesvc.NewTemplatesRegistry(templatesDir)
	if err != nil {
		log.Fatalf("failed to load templates: %v", err)
	}

	svc := templatesvc.NewTemplateService(registry)
	handler := templatesvc.NewTemplatesHandler(svc)

	mux := http.NewServeMux()

	// external (via gateway)
	mux.HandleFunc("GET /api/templates", handler.GetAll)

	// internal (for deploy service)
	mux.HandleFunc("GET /internal/templates/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "id is required", http.StatusBadRequest)
			return
		}
		tpl, ok := registry.Get(id)
		if !ok {
			http.Error(w, "template not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tpl)
	})

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	log.Printf("template service starting on port %d", port)
	http.ListenAndServe(":"+strconv.Itoa(port), mux)
}
