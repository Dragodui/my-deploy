package templatesvc

import (
	"context"
	"os"
	"path/filepath"

	"github.com/dragodui/my-deploy/internal/shared/models"
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
		if filepath.Ext(file.Name()) != ".yaml" && filepath.Ext(file.Name()) != ".yml" {
			continue
		}
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

func (r *TemplatesRegistry) GetAll(ctx context.Context) []*models.AppTemplate {
	templates := make([]*models.AppTemplate, 0, len(r.templates))
	for _, v := range r.templates {
		templates = append(templates, v)
	}
	return templates
}
