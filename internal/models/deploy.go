package models

import "time"

// - resource
type ResourceTemplate struct {
	Memory string `json:"memory,omitempty" yaml:"memory,omitempty"`
	CPU    int    `json:"cpu,omitempty" yaml:"cpu,omitempty"`
}

// - volume
type VolumeTemplate struct {
	Name          string `json:"name,omitempty" yaml:"name,omitempty"`
	ContainerPath string `json:"container_path" yaml:"container_path"`
}

// - port
type PortTemplate struct {
	Name      string `json:"name,omitempty" yaml:"name,omitempty"`
	Container int    `json:"container" yaml:"container"`
	Protocol  string `json:"protocol,omitempty" yaml:"protocol,omitempty"`
}

// app template will be given from /templates/*.yaml files
type AppTemplate struct {
	ID          string `json:"id" yaml:"id" db:"id"`
	Name        string `json:"name" yaml:"name" db:"name"`
	Description string `json:"description" yaml:"description" db:"description"`

	Image string `json:"image" yaml:"image" db:"image"`

	Ports   []PortTemplate    `json:"ports" yaml:"ports" db:"ports"`
	Volumes []VolumeTemplate  `json:"volumes" yaml:"volumes" db:"volumes"`
	Env     map[string]string `json:"env" yaml:"env" db:"env"`

	Resources *ResourceTemplate `json:"resources,omitempty" yaml:"resources,omitempty" db:"resources"`

	Restart string `json:"restart,omitempty" yaml:"restart,omitempty" db:"restart"`
}

// - volumes data
type VolumeBinding struct {
	HostPath      string `json:"host_path"`
	ContainerPath string `json:"container_path"`
}

// - ports data
type PortBinding struct {
	HostPort      int `json:"host_port"`
	ContainerPort int `json:"container_port"`
}

// data from user request to create deploy
type DeployRequest struct {
	Name string `json:"name"`

	AppID *string `json:"app_id,omitempty"`
	Image *string `json:"image,omitempty"`

	Ports   []PortBinding     `json:"ports,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	Volumes []VolumeBinding   `json:"volumes,omitempty"`

	Memory string `json:"memory,omitempty"`
	CPU    int    `json:"cpu,omitempty"`
}

// full deployment info
type Deployment struct {
	ID   string `json:"id" db:"id"`
	Name string `json:"name" db:"name"`

	AppID *string `json:"app_id,omitempty" db:"app_id"`
	Image string  `json:"image" db:"image"`

	ContainerID *string `json:"container_id,omitempty" db:"container_id"`

	Ports   []PortBinding   `json:"ports" db:"ports"`
	Volumes []VolumeBinding `json:"volumes" db:"volumes"`
	Env     []string        `json:"env" db:"env"`
	AgentID string          `json:"agent_id" db:"agent_id"`

	Status    string    `json:"status" db:"status"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}
