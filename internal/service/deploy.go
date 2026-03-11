package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dragodui/my-deploy/internal/agent"
	"github.com/dragodui/my-deploy/internal/models"
	"github.com/dragodui/my-deploy/internal/registry"
	"github.com/google/uuid"
	"maps"
)

type DeployRepo interface {
	Create(deploy *models.Deployment) error
}

type AgentRegistryProvider interface {
	Get(token string) (*registry.AgentConn, bool)
}

type TemplateProvider interface {
	Get(id string) (*models.AppTemplate, bool)
}

func mapToEnv(defaults, overrides map[string]string) []string {
	env := map[string]string{}

	maps.Copy(env, defaults)

	maps.Copy(env, overrides)

	var result []string
	for k, v := range env {
		result = append(result, k+"="+v)
	}

	return result
}

type DeployService struct {
	repo      DeployRepo
	registry  AgentRegistryProvider
	templates TemplateProvider
}

func NewDeployService(repo DeployRepo, reg AgentRegistryProvider, templates TemplateProvider) *DeployService {
	return &DeployService{repo, reg, templates}
}

// Create sends a deploy command to the connected agent.
func (svc *DeployService) Create(ctx context.Context, agentToken string, req models.DeployRequest) error {
	ac, ok := svc.registry.Get(agentToken)
	if !ok {
		return fmt.Errorf("agent not connected")
	}

	var tpl *models.AppTemplate
	if req.AppID != nil {
		var found bool
		tpl, found = svc.templates.Get(*req.AppID)
		if !found {
			return fmt.Errorf("not found template with id: %s", *req.AppID)
		}
	}

	// build payload for agent
	payload := agent.CreatePayload{
		Name: req.Name,
	}

	// image
	if tpl != nil {
		payload.Image = tpl.Image
		payload.Env = mergeEnv(tpl.Env, req.Env)
	} else if req.Image != nil {
		payload.Image = *req.Image
		payload.Env = req.Env
	} else {
		return fmt.Errorf("image not specified")
	}

	// ports
	if tpl != nil && len(tpl.Ports) > 0 {
		for _, p := range tpl.Ports {
			hostPort := p.Container
			for _, pb := range req.Ports {
				if pb.ContainerPort == p.Container {
					hostPort = pb.HostPort
				}
			}
			payload.Ports = append(payload.Ports, agent.PortBinding{
				HostPort:      hostPort,
				ContainerPort: p.Container,
			})
		}
	} else {
		for _, pb := range req.Ports {
			payload.Ports = append(payload.Ports, agent.PortBinding{
				HostPort:      pb.HostPort,
				ContainerPort: pb.ContainerPort,
			})
		}
	}

	// volumes
	if tpl != nil && len(tpl.Volumes) > 0 {
		for _, vol := range tpl.Volumes {
			payload.Volumes = append(payload.Volumes, agent.VolumeBinding{
				HostPath:      fmt.Sprintf("/var/lib/mydeploy/%s/%s", req.Name, vol.Name),
				ContainerPath: vol.ContainerPath,
			})
		}
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	result, err := ac.SendCommand(ctx, agent.Command{
		Type:    "create",
		ID:      uuid.New().String(),
		Payload: payloadJSON,
	})
	if err != nil {
		return fmt.Errorf("agent communication error: %w", err)
	}
	if !result.Success {
		return fmt.Errorf("agent error: %s", result.Error)
	}

	// save to db
	var ports []models.PortBinding
	for _, p := range payload.Ports {
		ports = append(ports, models.PortBinding{HostPort: p.HostPort, ContainerPort: p.ContainerPort})
	}

	deploy := &models.Deployment{
		Name:        req.Name,
		AppID:       req.AppID,
		Image:       payload.Image,
		ContainerID: result.ContainerID,
		Ports:       ports,
		Env:         mapToEnv(payload.Env, nil),
		Status:      "running",
	}

	return svc.repo.Create(deploy)
}

func mergeEnv(defaults, overrides map[string]string) map[string]string {
	merged := make(map[string]string, len(defaults)+len(overrides))
	for k, v := range defaults {
		merged[k] = v
	}
	for k, v := range overrides {
		merged[k] = v
	}
	return merged
}
