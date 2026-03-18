package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/netip"
	"strconv"
	"strings"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

type Handler struct {
	docker *client.Client
}

type ProgressFunc func(Progress)
type LogFunc func(LogChunk)

func NewHandler(docker *client.Client) *Handler {
	return &Handler{docker: docker}
}

func (h *Handler) Handle(ctx context.Context, cmd Command, progressFunc ProgressFunc, logFunc LogFunc) Result {
	switch cmd.Type {
	case "create":
		return h.handleCreate(ctx, cmd, progressFunc)
	case "start":
		return h.handleStart(ctx, cmd)
	case "stop":
		return h.handleStop(ctx, cmd)
	case "inspect":
		return h.handleInspect(ctx, cmd)
	case "logs":
		return h.handleLogs(ctx, cmd, logFunc)
	case "remove":
		return h.handleRemove(ctx, cmd)
	default:
		return Result{Type: "result", ID: cmd.ID, Success: false, Error: "unknown command: " + cmd.Type}
	}
}

func (h *Handler) handleInspect(ctx context.Context, cmd Command) Result {
	var p ContainerPayload
	if err := json.Unmarshal(cmd.Payload, &p); err != nil {
		return Result{Type: "result", ID: cmd.ID, Success: false, Error: "invalid payload: " + err.Error()}
	}

	res, err := h.docker.ContainerInspect(ctx, p.ContainerID, client.ContainerInspectOptions{})
	if err != nil {
		return Result{Type: "result", ID: cmd.ID, Success: false, Error: err.Error()}
	}

	return Result{
		Type:        "result",
		ID:          cmd.ID,
		Success:     true,
		ContainerID: p.ContainerID,
		Status:      string(res.Container.State.Status),
	}
}

func (h *Handler) handleCreate(ctx context.Context, cmd Command, notify ProgressFunc) Result {
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
		Resources: container.Resources{
			Memory:   parseMemory(p.Memory),
			NanoCPUs: int64(p.CPU) * 1e9,
		},
	}

	// pull image
	log.Printf("pulling image %s...", p.Image)
	notify(Progress{
		Type:    "progress",
		ID:      cmd.ID,
		Message: fmt.Sprintf("pulling image %s...", p.Image),
	})
	pullReader, err := h.docker.ImagePull(ctx, p.Image, client.ImagePullOptions{})
	if err != nil {
		return Result{Type: "result", ID: cmd.ID, Success: false, Error: "failed to pull image: " + err.Error()}
	}
	io.Copy(io.Discard, pullReader)
	pullReader.Close()
	log.Printf("image %s pulled", p.Image)

	notify(Progress{Type: "progress", ID: cmd.ID, Message: "Creating container..."})
	res, err := h.docker.ContainerCreate(ctx, client.ContainerCreateOptions{
		Name:       p.Name,
		Config:     config,
		HostConfig: hostConfig,
	})
	if err != nil {
		return Result{Type: "result", ID: cmd.ID, Success: false, Error: err.Error()}
	}

	// auto-start after create
	notify(Progress{Type: "progress", ID: cmd.ID, Message: "Starting container..."})
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

func (h *Handler) handleLogs(ctx context.Context, cmd Command, logFunc LogFunc) Result {
	var p ContainerPayload
	if err := json.Unmarshal(cmd.Payload, &p); err != nil {
		return Result{Type: "result", ID: cmd.ID, Success: false, Error: "invalid payload: " + err.Error()}
	}
	reader, err := h.docker.ContainerLogs(ctx, p.ContainerID, client.ContainerLogsOptions{Follow: true, ShowStdout: true, ShowStderr: true, Tail: "100", Timestamps: true})
	if err != nil {
		return Result{Type: "result", ID: cmd.ID, Success: false, Error: "invalid payload: " + err.Error()}
	}
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		logFunc(LogChunk{Type: "logs", ID: cmd.ID, Data: line})
	}

	logFunc(LogChunk{Type: "logs", ID: cmd.ID, Done: true})
	return Result{Type: "result", ID: cmd.ID, Success: true}
}

func parseMemory(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	s = strings.ToUpper(s)
	multiplier := int64(1)
	if strings.HasSuffix(s, "G") {
		multiplier = 1024 * 1024 * 1024
		s = strings.TrimSuffix(s, "G")
	} else if strings.HasSuffix(s, "M") {
		multiplier = 1024 * 1024
		s = strings.TrimSuffix(s, "M")
	}
	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return val * multiplier
}
