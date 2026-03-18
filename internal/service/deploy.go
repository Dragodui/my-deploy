package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

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
	Get(agentID string) (*registry.AgentConn, bool)
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

type deployTask struct {
	cmdID   string
	agentID string
}
type DeployService struct {
	repo      DeployRepo
	registry  AgentRegistryProvider
	templates TemplateProvider
	cmdMap    sync.Map
}

func NewDeployService(repo DeployRepo, reg AgentRegistryProvider, templates TemplateProvider) *DeployService {
	return &DeployService{repo: repo, registry: reg, templates: templates}
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

	var ports []models.PortBinding
	for _, p := range payload.Ports {
		ports = append(ports, models.PortBinding{HostPort: p.HostPort, ContainerPort: p.ContainerPort})
	}

	// create cmd task
	cmdID := uuid.New().String()
	deploy := &models.Deployment{
		AgentID:     agentID,
		Name:        req.Name,
		AppID:       req.AppID,
		Image:       payload.Image,
		ContainerID: nil,
		Ports:       ports,
		Env:         mapToEnv(payload.Env, nil),
		Status:      "deploying",
	}

	saved, err := svc.repo.Create(ctx, deploy)
	if err != nil {
		return nil, fmt.Errorf("failed to create deployment: %w", err)
	}

	svc.cmdMap.Store(saved.ID, deployTask{
		cmdID, agentID,
	})

	go func() {
		localCtx := context.Background()
		defer svc.cmdMap.Delete(saved.ID)
		result, err := ac.SendCommand(localCtx, agent.Command{
			Type:    "create",
			ID:      cmdID,
			Payload: payloadJSON,
		})
		if err != nil {
			svc.UpdateStatus(localCtx, saved.ID, "error")
			return
		}
		if !result.Success {
			svc.UpdateStatus(localCtx, saved.ID, "error")
			return
		}
		svc.UpdateContainerID(localCtx, saved.ID, result.ContainerID)
		svc.UpdateStatus(localCtx, saved.ID, "running")
	}()

	return saved, nil
}

func (svc *DeployService) InspectDeployment(ctx context.Context, id string) (string, error) {
	deploy, err := svc.repo.GetByID(ctx, id)
	if err != nil {
		return "", err
	}

	if deploy.ContainerID == nil || *deploy.ContainerID == "" {
		return "", fmt.Errorf("no container inside deploy found")
	}

	ac, ok := svc.registry.Get(deploy.AgentID)
	if !ok {
		return "", fmt.Errorf("no agent inside deploy found")
	}

	res, err := ac.SendCommand(ctx, agent.Command{
		Type:    "inspect",
		ID:      uuid.New().String(),
		Payload: []byte(fmt.Sprintf(`{"container_id":"%s"}`, *deploy.ContainerID)),
	})

	if err != nil {
		return "", err
	}

	return res.Status, nil
}

func (svc *DeployService) GetProgress(deployID string) string {
	deploy, ok := svc.cmdMap.Load(deployID)
	if !ok {
		return "error"
	}
	task := deploy.(deployTask)
	conn, ok := svc.registry.Get(task.agentID)
	if !ok {
		return "error"
	}

	return conn.GetProgress(task.cmdID)
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
	deploy, err := svc.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if deploy.ContainerID != nil && *deploy.ContainerID != "" {
		ac, ok := svc.registry.Get(deploy.AgentID)
		if ok {
			svc.rollbackContainer(ctx, ac, *deploy.ContainerID)
		}
	}

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

func (svc *DeployService) Start(ctx context.Context, containerID, agentID string) error {
	ac, ok := svc.registry.Get(agentID)
	if !ok {
		return fmt.Errorf("no agent found")
	}

	_, err := ac.SendCommand(ctx, agent.Command{
		Type:    "start",
		ID:      uuid.New().String(),
		Payload: []byte(fmt.Sprintf(`{"container_id":"%s"}`, containerID)),
	})

	return err
}

func (svc *DeployService) Stop(ctx context.Context, containerID, agentID string) error {
	ac, ok := svc.registry.Get(agentID)
	if !ok {
		return fmt.Errorf("no agent found")
	}
	_, err := ac.SendCommand(ctx, agent.Command{
		Type:    "stop",
		ID:      uuid.New().String(),
		Payload: []byte(fmt.Sprintf(`{"container_id":"%s"}`, containerID)),
	})

	return err
}
