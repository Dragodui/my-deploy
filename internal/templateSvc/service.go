package templatesvc

import (
	"context"

	"github.com/dragodui/my-deploy/internal/shared/models"
)

type TemplateService struct {
	registry *TemplatesRegistry
}

func NewTemplateService(registry *TemplatesRegistry) *TemplateService {
	return &TemplateService{registry: registry}
}

func (s *TemplateService) GetAll(ctx context.Context) []*models.AppTemplate {
	return s.registry.GetAll(ctx)
}
