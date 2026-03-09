package templates

import (
	"os"
	"path/filepath"

	"github.com/dragodui/my-deploy/internal/models"
	"github.com/goccy/go-yaml"
)

type TemplatesRegistry struct {
	templates map[string]*models.AppTemplate
}

func NewTemplatesRegistry(dir string) (*TemplatesRegistry, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	templates := map[string]*models.AppTemplate{}
	for _, file := range files {
		data, err := os.ReadFile(filepath.Join(dir, file.Name()))
		if err != nil {
			return nil, err
		}

		var tpl models.AppTemplate
		if err := yaml.Unmarshal(data, &tpl); err != nil {
			return nil, err
		}
		templates[tpl.ID] = &tpl
	}

	return &TemplatesRegistry{templates: templates}, nil
}

func (r *TemplatesRegistry) Get(id string) (*models.AppTemplate, bool) {
	tpl, ok := r.templates[id]
	return tpl, ok
}
