package agent

import "encoding/json"

type Command struct {
	Type    string          `json:"type"`
	ID      string          `json:"id"`
	Payload json.RawMessage `json:"payload"`
}

type CreatePayload struct {
	Name    string            `json:"name"`
	Image   string            `json:"image"`
	Env     map[string]string `json:"env,omitempty"`
	Ports   []PortBinding     `json:"ports,omitempty"`
	Volumes []VolumeBinding   `json:"volumes,omitempty"`
}

type ContainerPayload struct {
	ContainerID string `json:"container_id"`
}

type Result struct {
	Type        string `json:"type"`
	ID          string `json:"id"`
	Success     bool   `json:"success"`
	ContainerID string `json:"container_id,omitempty"`
	Error       string `json:"error,omitempty"`
}

// reuse from models or duplicate here
type PortBinding struct {
	HostPort      int `json:"host_port"`
	ContainerPort int `json:"container_port"`
}

type VolumeBinding struct {
	HostPath      string `json:"host_path"`
	ContainerPath string `json:"container_path"`
}

type Progress struct {
	Type string `json:"type"`
	ID string `json:"id"`
	Message string `json:"message"`
}