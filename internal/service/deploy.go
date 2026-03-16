package service

import (
	"context"
	"encoding/json"
	"fmt"

	"maps"

	"github.com/dragodui/my-deploy/internal/agent"
	"github.com/dragodui/my-deploy/internal/models"
	"github.com/dragodui/my-deploy/internal/registry"
	"github.com/google/uuid"
)

type DeployRepo interface {
	Create(ctx context.Context, d *models.Deployment) (*models.Deployment, error)
	GetByID(ctx context.Context, id string) (*models.Deployment, error)
	ListByAgent(ctx context.Context, agentID string) ([]models.Deployment, error)
	UpdateStatus(ctx context.Context, id, status string) error
	UpdateContainerID(ctx context.Context, id, containerID string) error
	Delete(ctx context.Context, id string) error
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

type DeployService struct {
	repo      DeployRepo
	registry  AgentRegistryProvider
	templates TemplateProvider
}

func NewDeployService(repo DeployRepo, reg AgentRegistryProvider, templates TemplateProvider) *DeployService {
	return &DeployService{repo, reg, templates}
}

// Create sends a deploy command to the connected agent.
func (svc *DeployService) Create(ctx context.Context, agentID string, req models.DeployRequest) (*models.Deployment, error) {
	ac, ok := svc.registry.Get(agentID)
	if !ok {
		return nil, fmt.Errorf("agent not connected")
	}

	var tpl *models.AppTemplate
	if req.AppID != nil {
		var found bool
		tpl, found = svc.templates.Get(*req.AppID)
		if !found {
			return nil, fmt.Errorf("not found template with id: %s", *req.AppID)
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
		return nil, fmt.Errorf("image not specified")
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
		return nil, err
	}

	result, err := ac.SendCommand(ctx, agent.Command{
		Type:    "create",
		ID:      uuid.New().String(),
		Payload: payloadJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("agent communication error: %w", err)
	}
	if !result.Success {
		return nil, fmt.Errorf("agent error: %s", result.Error)
	}

	// save to db
	var ports []models.PortBinding
	for _, p := range payload.Ports {
		ports = append(ports, models.PortBinding{HostPort: p.HostPort, ContainerPort: p.ContainerPort})
	}

	deploy := &models.Deployment{
		AgentID:     agentID,
		Name:        req.Name,
		AppID:       req.AppID,
		Image:       payload.Image,
		ContainerID: &result.ContainerID,
		Ports:       ports,
		Env:         mapToEnv(payload.Env, nil),
		Status:      "running",
	}

	saved, err := svc.repo.Create(ctx, deploy)

	if err != nil {
		svc.rollbackContainer(ctx, ac, result.ContainerID)
		return nil, fmt.Errorf("failed to save deployment: %w", err)
	}
	return saved, nil

}

func (svc *DeployService) GetByID(ctx context.Context, id string) (*models.Deployment, error) {
	return svc.repo.GetByID(ctx, id)
}

func (svc *DeployService) ListByAgent(ctx context.Context, agentID string) ([]models.Deployment, error) {
	return svc.repo.ListByAgent(ctx, agentID)
}

func (svc *DeployService) UpdateStatus(ctx context.Context, id, status string) error {
	return svc.repo.UpdateStatus(ctx, id, status)
}

func (svc *DeployService) UpdateContainerID(ctx context.Context, id, containerID string) error {
	return svc.repo.UpdateContainerID(ctx, id, containerID)
}

func (svc *DeployService) Delete(ctx context.Context, id string) error {
	return svc.repo.Delete(ctx, id)
}

func (svc *DeployService) rollbackContainer(ctx context.Context, ac *registry.AgentConn, containerID string) error {

	if _, err := ac.SendCommand(ctx, agent.Command{
		Type:    "stop",
		ID:      uuid.New().String(),
		Payload: []byte(fmt.Sprintf(`{"container_id":"%s"}`, containerID)),
	}); err != nil {
		return nil
	}

	_, err := ac.SendCommand(ctx, agent.Command{
		Type:    "remove",
		ID:      uuid.New().String(),
		Payload: []byte(fmt.Sprintf(`{"container_id":"%s"}`, containerID)),
	})

	return err
}
