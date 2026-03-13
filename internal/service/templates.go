package service

import (
	"context"

	"github.com/dragodui/my-deploy/internal/models"
	"github.com/dragodui/my-deploy/internal/templates"
)

type TemplateService struct {
	registry *templates.TemplatesRegistry
}

func NewTemplateService(registry *templates.TemplatesRegistry) *TemplateService {
	return &TemplateService{registry}
}

func (s *TemplateService) GetAll(ctx context.Context) []*models.AppTemplate {
	return s.registry.GetAll(ctx)
}
