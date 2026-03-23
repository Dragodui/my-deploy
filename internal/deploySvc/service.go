package deploysvc

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"maps"

	"github.com/dragodui/my-deploy/internal/shared/models"
	agentpb "github.com/dragodui/my-deploy/internal/shared/proto/agentpb/proto"
	"github.com/google/uuid"
)

type DeployRepo interface {
	Create(ctx context.Context, d *models.Deployment) (*models.Deployment, error)
	GetByID(ctx context.Context, id string) (*models.Deployment, error)
	ListByAgent(ctx context.Context, agentID string) ([]models.Deployment, error)
	UpdateStatus(ctx context.Context, id, status string) error
	UpdateContainerID(ctx context.Context, id, containerID string) error
	Update(ctx context.Context, id string, params models.UpdateDeploymentReq) error
	Delete(ctx context.Context, id string) error
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
	repo        DeployRepo
	agentClient agentpb.AgentInternalClient
	templates   TemplateClient
	cmdMap      sync.Map
}

func NewDeployService(repo DeployRepo, agentClient agentpb.AgentInternalClient, templates TemplateClient) *DeployService {
	return &DeployService{repo: repo, agentClient: agentClient, templates: templates}
}

// Create sends a deploy command to the connected agent.
func (svc *DeployService) Create(ctx context.Context, agentID string, req models.DeployRequest) (*models.Deployment, error) {

	var tpl *models.AppTemplate
	if req.AppID != nil {
		var found bool
		tpl, found = svc.templates.Get(*req.AppID)
		if !found {
			return nil, fmt.Errorf("not found template with id: %s", *req.AppID)
		}
	}

	// build payload for agent
	payload := models.CreatePayload{
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
			payload.Ports = append(payload.Ports, models.PortBinding{
				HostPort:      hostPort,
				ContainerPort: p.Container,
			})
		}
	} else {
		for _, pb := range req.Ports {
			payload.Ports = append(payload.Ports, models.PortBinding{
				HostPort:      pb.HostPort,
				ContainerPort: pb.ContainerPort,
			})
		}
	}

	// volumes
	if tpl != nil && len(tpl.Volumes) > 0 {
		for _, vol := range tpl.Volumes {
			payload.Volumes = append(payload.Volumes, models.VolumeBinding{
				HostPath:      fmt.Sprintf("/var/lib/mydeploy/%s/%s", req.Name, vol.Name),
				ContainerPath: vol.ContainerPath,
			})
		}
	}

	// resources
	if tpl != nil && tpl.Resources != nil {
		payload.Memory = tpl.Resources.Memory
		payload.CPU = tpl.Resources.CPU
	}
	if req.Memory != "" {
		payload.Memory = req.Memory
	}
	if req.CPU > 0 {
		payload.CPU = req.CPU
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
		result, err := svc.agentClient.SendCommand(localCtx, &agentpb.SendCommandRequest{
			AgentId: agentID,
			Cmd: &agentpb.Command{
				Type:    "create",
				Id:      cmdID,
				Payload: payloadJSON,
			},
		})
		if err != nil {
			svc.UpdateStatus(localCtx, saved.ID, "error")
			return
		}
		if !result.Success {
			svc.UpdateStatus(localCtx, saved.ID, "error")
			return
		}
		svc.UpdateContainerID(localCtx, saved.ID, result.ContainerId)
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

	res, err := svc.agentClient.SendCommand(ctx, &agentpb.SendCommandRequest{
		Cmd: &agentpb.Command{
			Type:    "inspect",
			Id:      uuid.New().String(),
			Payload: []byte(fmt.Sprintf(`{"container_id":"%s"}`, *deploy.ContainerID)),
		},
		AgentId: deploy.AgentID,
	})

	if err != nil {
		return "", err
	}

	return res.Status, nil
}

func (svc *DeployService) GetProgress(ctx context.Context, deployID string) string {
	deploy, ok := svc.cmdMap.Load(deployID)
	if !ok {
		return "error"
	}

	task := deploy.(deployTask)
	res, err := svc.agentClient.GetProgress(ctx, &agentpb.GetProgressRequest{
		CmdId:   task.cmdID,
		AgentId: task.agentID,
	})

	if err != nil {
		return "error"
	}

	return res.Progress
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

func (svc *DeployService) Update(ctx context.Context, userID, id string, params models.UpdateDeploymentReq) (*models.Deployment, error) {
	deploy, err := svc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	agent, err := svc.agentClient.GetAgent(ctx, &agentpb.GetAgentRequest{
		Id: deploy.AgentID,
	})
	if err != nil {
		return nil, err
	}

	if agent.UserId != userID {
		return nil, fmt.Errorf("invalid user id")
	}

	if err := svc.repo.Update(ctx, id, params); err != nil {
		return nil, err
	}

	return svc.repo.GetByID(ctx, id)
}

func (svc *DeployService) Delete(ctx context.Context, userID, id string) error {
	deploy, err := svc.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	agent, err := svc.agentClient.GetAgent(ctx, &agentpb.GetAgentRequest{
		Id: deploy.AgentID,
	})
	if err != nil {
		return err
	}
	if agent.UserId != userID {
		return fmt.Errorf("invalid user id")
	}

	if deploy.ContainerID != nil && *deploy.ContainerID != "" {
		svc.rollbackContainer(ctx, deploy.AgentID, *deploy.ContainerID)
	}

	return svc.repo.Delete(ctx, id)
}

func (svc *DeployService) rollbackContainer(ctx context.Context, agentID, containerID string) error {

	if _, err := svc.agentClient.SendCommand(ctx, &agentpb.SendCommandRequest{
		Cmd: &agentpb.Command{
			Type:    "stop",
			Id:      uuid.New().String(),
			Payload: []byte(fmt.Sprintf(`{"container_id":"%s"}`, containerID)),
		},
		AgentId: agentID,
	}); err != nil {
		return nil
	}

	_, err := svc.agentClient.SendCommand(ctx, &agentpb.SendCommandRequest{
		Cmd: &agentpb.Command{
			Type:    "remove",
			Id:      uuid.New().String(),
			Payload: []byte(fmt.Sprintf(`{"container_id":"%s"}`, containerID)),
		},
		AgentId: agentID,
	})

	return err
}

func (svc *DeployService) Start(ctx context.Context, containerID, agentID string) error {
	_, err := svc.agentClient.SendCommand(ctx, &agentpb.SendCommandRequest{
		Cmd: &agentpb.Command{
			Type:    "start",
			Id:      uuid.New().String(),
			Payload: []byte(fmt.Sprintf(`{"container_id":"%s"}`, containerID)),
		},
		AgentId: agentID,
	})

	return err
}

func (svc *DeployService) Stop(ctx context.Context, containerID, agentID string) error {
	_, err := svc.agentClient.SendCommand(ctx, &agentpb.SendCommandRequest{
		Cmd: &agentpb.Command{
			Type:    "stop",
			Id:      uuid.New().String(),
			Payload: []byte(fmt.Sprintf(`{"container_id":"%s"}`, containerID)),
		},
		AgentId: agentID,
	})

	return err
}
