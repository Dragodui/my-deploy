package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/netip"
	"strconv"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

type Handler struct {
	docker *client.Client
}

func NewHandler(docker *client.Client) *Handler {
	return &Handler{docker: docker}
}

func (h *Handler) Handle(ctx context.Context, cmd Command) Result {
	switch cmd.Type {
	case "create":
		return h.handleCreate(ctx, cmd)
	case "start":
		return h.handleStart(ctx, cmd)
	case "stop":
		return h.handleStop(ctx, cmd)
	case "remove":
		return h.handleRemove(ctx, cmd)
	default:
		return Result{Type: "result", ID: cmd.ID, Success: false, Error: "unknown command: " + cmd.Type}
	}
}

func (h *Handler) handleCreate(ctx context.Context, cmd Command) Result {
	var p CreatePayload
	if err := json.Unmarshal(cmd.Payload, &p); err != nil {
		return Result{Type: "result", ID: cmd.ID, Success: false, Error: "invalid payload: " + err.Error()}
	}

	// build env
	var env []string
	for k, v := range p.Env {
		env = append(env, k+"="+v)
	}

	// build ports
	exposedPorts := network.PortSet{}
	portBindings := network.PortMap{}
	for _, pb := range p.Ports {
		port, err := network.ParsePort(fmt.Sprintf("%d/tcp", pb.ContainerPort))
		if err != nil {
			continue
		}
		exposedPorts[port] = struct{}{}
		portBindings[port] = []network.PortBinding{
			{HostIP: netip.MustParseAddr("0.0.0.0"), HostPort: strconv.Itoa(pb.HostPort)},
		}
	}

	// build volumes
	var binds []string
	for _, vol := range p.Volumes {
		binds = append(binds, vol.HostPath+":"+vol.ContainerPath)
	}

	config := &container.Config{
		Image:        p.Image,
		Env:          env,
		ExposedPorts: exposedPorts,
	}

	hostConfig := &container.HostConfig{
		PortBindings:  portBindings,
		Binds:         binds,
		RestartPolicy: container.RestartPolicy{Name: "always"},
	}

	res, err := h.docker.ContainerCreate(ctx, client.ContainerCreateOptions{
		Name:       p.Name,
		Config:     config,
		HostConfig: hostConfig,
	})
	if err != nil {
		return Result{Type: "result", ID: cmd.ID, Success: false, Error: err.Error()}
	}

	// auto-start after create
	if _, err := h.docker.ContainerStart(ctx, res.ID, client.ContainerStartOptions{}); err != nil {
		return Result{Type: "result", ID: cmd.ID, Success: false, ContainerID: res.ID, Error: "created but failed to start: " + err.Error()}
	}

	return Result{Type: "result", ID: cmd.ID, Success: true, ContainerID: res.ID}
}

func (h *Handler) handleStart(ctx context.Context, cmd Command) Result {
	var p ContainerPayload
	if err := json.Unmarshal(cmd.Payload, &p); err != nil {
		return Result{Type: "result", ID: cmd.ID, Success: false, Error: "invalid payload: " + err.Error()}
	}

	if _, err := h.docker.ContainerStart(ctx, p.ContainerID, client.ContainerStartOptions{}); err != nil {
		return Result{Type: "result", ID: cmd.ID, Success: false, Error: err.Error()}
	}

	return Result{Type: "result", ID: cmd.ID, Success: true, ContainerID: p.ContainerID}
}

func (h *Handler) handleStop(ctx context.Context, cmd Command) Result {
	var p ContainerPayload
	if err := json.Unmarshal(cmd.Payload, &p); err != nil {
		return Result{Type: "result", ID: cmd.ID, Success: false, Error: "invalid payload: " + err.Error()}
	}

	if _, err := h.docker.ContainerStop(ctx, p.ContainerID, client.ContainerStopOptions{}); err != nil {
		return Result{Type: "result", ID: cmd.ID, Success: false, Error: err.Error()}
	}

	return Result{Type: "result", ID: cmd.ID, Success: true, ContainerID: p.ContainerID}
}

func (h *Handler) handleRemove(ctx context.Context, cmd Command) Result {
	var p ContainerPayload
	if err := json.Unmarshal(cmd.Payload, &p); err != nil {
		return Result{Type: "result", ID: cmd.ID, Success: false, Error: "invalid payload: " + err.Error()}
	}

	if _, err := h.docker.ContainerRemove(ctx, p.ContainerID, client.ContainerRemoveOptions{}); err != nil {
		return Result{Type: "result", ID: cmd.ID, Success: false, Error: err.Error()}
	}

	return Result{Type: "result", ID: cmd.ID, Success: true, ContainerID: p.ContainerID}
}
