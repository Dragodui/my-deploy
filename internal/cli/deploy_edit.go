package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dragodui/my-deploy/internal/agent"
	"github.com/dragodui/my-deploy/internal/shared/models"
)

type DeployUpdateMsg struct {
	deployment *models.Deployment
	err        error
}

type DeployEditModel struct {
	api    *agent.APIClient
	config *agent.LocalConfig
	deploy models.Deployment

	inputs     []textinput.Model
	labels     []string
	focusIndex int

	loading bool
	err     error
	done    bool
}

func NewDeployEditModel(api *agent.APIClient, config *agent.LocalConfig, deploy models.Deployment) DeployEditModel {
	// Name
	nameInput := textinput.New()
	nameInput.SetValue(deploy.Name)
	nameInput.CharLimit = 64
	nameInput.PromptStyle = FocusedInput
	nameInput.TextStyle = FocusedInput
	nameInput.Focus()

	// Image
	imageInput := textinput.New()
	imageInput.SetValue(deploy.Image)
	imageInput.CharLimit = 128
	imageInput.PromptStyle = BlurredInput
	imageInput.TextStyle = BlurredInput

	// Ports
	var portParts []string
	for _, p := range deploy.Ports {
		portParts = append(portParts, fmt.Sprintf("%d:%d", p.HostPort, p.ContainerPort))
	}
	portsInput := textinput.New()
	portsInput.SetValue(strings.Join(portParts, ", "))
	portsInput.CharLimit = 128
	portsInput.PromptStyle = BlurredInput
	portsInput.TextStyle = BlurredInput

	// Env
	envInput := textinput.New()
	envInput.SetValue(strings.Join(deploy.Env, ", "))
	envInput.CharLimit = 256
	envInput.PromptStyle = BlurredInput
	envInput.TextStyle = BlurredInput

	return DeployEditModel{
		api:    api,
		config: config,
		deploy: deploy,
		inputs: []textinput.Model{nameInput, imageInput, portsInput, envInput},
		labels: []string{"Name", "Image", "Ports (host:container)", "Env (KEY=VALUE, ...)"},
	}
}

func updateDeployCmd(api *agent.APIClient, jwt, id string, req models.UpdateDeploymentReq) tea.Cmd {
	return func() tea.Msg {
		deploy, err := api.UpdateDeployment(jwt, id, req)
		return DeployUpdateMsg{deployment: deploy, err: err}
	}
}

func (m DeployEditModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m DeployEditModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "tab", "shift+tab":
			if !m.loading {
				if msg.String() == "shift+tab" {
					m.focusIndex = (m.focusIndex - 1 + len(m.inputs)) % len(m.inputs)
				} else {
					m.focusIndex = (m.focusIndex + 1) % len(m.inputs)
				}
				return m, m.updateFocus()
			}
		case "enter":
			if m.loading {
				return m, nil
			}
			if m.focusIndex < len(m.inputs)-1 {
				m.focusIndex++
				return m, m.updateFocus()
			}
			req, err := m.buildUpdateRequest()
			if err != nil {
				m.err = err
				return m, nil
			}
			m.loading = true
			m.err = nil
			return m, updateDeployCmd(m.api, m.config.JWT, m.deploy.ID, req)
		}

	case DeployUpdateMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.done = true
		return m, tea.Quit
	}

	cmd := m.updateInputs(msg)
	return m, cmd
}

func (m *DeployEditModel) buildUpdateRequest() (models.UpdateDeploymentReq, error) {
	var req models.UpdateDeploymentReq

	name := strings.TrimSpace(m.inputs[0].Value())
	if name == "" {
		return req, fmt.Errorf("name is required")
	}
	req.Name = &name

	image := strings.TrimSpace(m.inputs[1].Value())
	if image == "" {
		return req, fmt.Errorf("image is required")
	}
	req.Image = &image

	portsStr := strings.TrimSpace(m.inputs[2].Value())
	if portsStr != "" {
		ports, err := parsePortsString(portsStr)
		if err != nil {
			return req, err
		}
		req.Ports = ports
	} else {
		req.Ports = []models.PortBinding{}
	}

	envStr := strings.TrimSpace(m.inputs[3].Value())
	if envStr != "" {
		var envList []string
		for _, part := range strings.Split(envStr, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				envList = append(envList, part)
			}
		}
		req.Env = envList
	} else {
		req.Env = []string{}
	}

	return req, nil
}

func (m *DeployEditModel) updateFocus() tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		if i == m.focusIndex {
			cmds[i] = m.inputs[i].Focus()
			m.inputs[i].PromptStyle = FocusedInput
			m.inputs[i].TextStyle = FocusedInput
		} else {
			m.inputs[i].Blur()
			m.inputs[i].PromptStyle = BlurredInput
			m.inputs[i].TextStyle = BlurredInput
		}
	}
	return tea.Batch(cmds...)
}

func (m *DeployEditModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m DeployEditModel) View() string {
	var b strings.Builder

	b.WriteString(Title.Render("Edit: " + m.deploy.Name))
	b.WriteString("\n")

	for i, input := range m.inputs {
		b.WriteString(InputLabel.Render(m.labels[i]))
		b.WriteString("\n")
		b.WriteString(input.View())
		b.WriteString("\n")
	}

	if m.loading {
		b.WriteString("\n")
		b.WriteString(Subtle.Render("Saving..."))
	}

	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(Error.Render(m.err.Error()))
	}

	b.WriteString("\n")
	b.WriteString(Subtle.Render("tab: switch field • enter: submit • esc: back"))

	return Container.Render(b.String())
}
