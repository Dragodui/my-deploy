package service

import (
	"context"
	"fmt"
	"net/netip"
	"strconv"

	"github.com/dragodui/my-deploy/internal/docker"
	"github.com/dragodui/my-deploy/internal/models"
	"github.com/dragodui/my-deploy/internal/repository"
	"github.com/dragodui/my-deploy/internal/templates"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

func mapToEnv(defaults, overrides map[string]string) []string {
	env := map[string]string{}

	for k, v := range defaults {
		env[k] = v
	}

	for k, v := range overrides {
		env[k] = v
	}

	var result []string
	for k, v := range env {
		result = append(result, k+"="+v)
	}

	return result
}

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
	config := &container.Config{}

	// image setup
	if tpl != nil {
		config.Image = tpl.Image
		config.Env = mapToEnv(tpl.Env, req.Env)
	} else if req.Image != nil {
		config.Image = *req.Image
		config.Env = mapToEnv(nil, req.Env)
	} else {
		return fmt.Errorf("image not specified")
	}

	// ports setup
	exposedPorts := network.PortSet{}
	portBindings := network.PortMap{}
	var ports []models.PortBinding

	if tpl != nil && len(tpl.Ports) > 0 {
		for _, p := range tpl.Ports {
			containerPort, _ := network.ParsePort(fmt.Sprintf("%d/tcp", p.Container))
			exposedPorts[containerPort] = struct{}{}

			hostPort := 0
			if len(req.Ports) > 0 {
				for _, pb := range req.Ports {
					if pb.ContainerPort == p.Container {
						hostPort = pb.HostPort
					}
				}
			}
			if hostPort == 0 {
				hostPort = p.Container
			}
			portBindings[containerPort] = []network.PortBinding{
				{HostIP: netip.MustParseAddr("0.0.0.0"), HostPort: strconv.Itoa(hostPort)},
			}
			ports = append(ports, models.PortBinding{HostPort: hostPort, ContainerPort: p.Container})
		}
	} else {
		ports = req.Ports
		for _, pb := range ports {
			containerPort, _ := network.ParsePort(fmt.Sprintf("%d/tcp", pb.ContainerPort))
			exposedPorts[containerPort] = struct{}{}

			portBindings[containerPort] = []network.PortBinding{
				{HostIP: netip.MustParseAddr("0.0.0.0"), HostPort: strconv.Itoa(pb.HostPort)},
			}
		}
	}
	config.ExposedPorts = exposedPorts

	// host config
	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
		RestartPolicy: container.RestartPolicy{
			Name: "always",
		},
	}

	// volumes
	if tpl != nil && len(tpl.Volumes) > 0 {
		hostConfig.Binds = []string{}
		for _, vol := range tpl.Volumes {
			hostPath := fmt.Sprintf("/var/lib/mydeploy/%s/%s", req.Name, vol.Name)
			hostConfig.Binds = append(hostConfig.Binds, fmt.Sprintf("%s:%s", hostPath, vol.ContainerPath))
		}
	}

	// container logic here
	res, err := svc.docker.Client.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config:     config,
		HostConfig: hostConfig,
		Name:       req.Name,
	})
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	if _, err := svc.docker.Client.ContainerStart(ctx, res.ID, client.ContainerStartOptions{}); err != nil {
		return err
	}

	deploy := models.Deployment{
		ID:          res.ID,
		Name:        req.Name,
		AppID:       req.AppID,
		Image:       config.Image,
		ContainerID: res.ID,
		Ports:       ports,
		Env:         config.Env,
	}

	// create in db
	if err := svc.repo.Create(&deploy); err != nil {
		return err
	}

	return nil
}
