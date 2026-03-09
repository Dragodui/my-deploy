package docker

import (
	"log"

	"github.com/dragodui/my-deploy/internal/config"
	"github.com/moby/moby/client"
)

type Docker struct {
	Client *client.Client
}

func NewDockerClient(cfg *config.Config) *Docker {
	dockerClient, err := client.New(client.WithHost(cfg.DockerHost),
		client.WithAPIVersion(cfg.DockerAPIVersion))

	if err != nil {
		log.Fatalf("Error while initialize docker client: %v", err)
	}

	return &Docker{Client: dockerClient}
}
