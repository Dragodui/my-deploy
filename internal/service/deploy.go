package service

import (
	"context"
	"fmt"

	"github.com/dragodui/my-deploy/internal/docker"
	"github.com/dragodui/my-deploy/internal/models"
	"github.com/dragodui/my-deploy/internal/repository"
	"github.com/dragodui/my-deploy/internal/templates"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

type DeployService struct {
	repo      *repository.DeployRepository
	docker    *docker.Docker
	templates *templates.TemplatesRegistry
}

func NewDeployService(repo *repository.DeployRepository, docker *docker.Docker, templates *templates.TemplatesRegistry) *DeployService {
	return &DeployService{repo, docker, templates}
}

// create docker container here
func (svc *DeployService) Create(ctx context.Context, req models.DeployRequest) error {
	var tpl *models.AppTemplate

	// check if app exists
	if req.AppID != nil {
		var ok bool
		tpl, ok = svc.templates.Get(*req.AppID)
		if !ok {
			return fmt.Errorf("not found template with id: %s", *req.AppID)
		}
	}

	// !!! continue templates logic
	var config container.Config

	// container logic here
	res, err := svc.docker.Client.ContainerCreate(ctx, &client.ContainerCreateOptions{})
	// create in db
	if err := svc.repo.Create(name, file); err != nil {
		return err
	}

	return nil
}
