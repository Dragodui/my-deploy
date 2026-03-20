package cli

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dragodui/my-deploy/internal/agent"
	"github.com/dragodui/my-deploy/internal/shared/models"
)

type TemplatesListMsg struct {
	templates []models.AppTemplate
	err       error
}

type PollResultMsg struct {
	deployment *models.Deployment
	err        error
}

type pollTickMsg struct {
}

type DeployResultMsg struct {
	deployment *models.Deployment
	err        error
}

type DeployModel struct {
	api    *agent.APIClient
	config *agent.LocalConfig
	state  string

	// choose
	cursor int
	items  []string

	// templates
	templates        []models.AppTemplate
	selectedTemplate *models.AppTemplate

	// form
	inputs     []textinput.Model
	focusIndex int
	labels     []string

	// common
	spinner spinner.Model
	loading bool
	err     error
	result  *models.Deployment

	// progress
	deployID string
	progress string
}

func NewDeployModel(api *agent.APIClient, config *agent.LocalConfig) DeployModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = FocusedInput

	return DeployModel{
		api:     api,
		config:  config,
		state:   "choose",
		items:   []string{"From template", "Custom image"},
		spinner: s,
	}
}

func pollDeployCmd(api *agent.APIClient, jwt, deployID string) tea.Cmd {
	return func() tea.Msg {
		deploy, err := api.GetDeployment(jwt, deployID)
		return PollResultMsg{deployment: deploy, err: err}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(1500*time.Millisecond, func(t time.Time) tea.Msg {
		return pollTickMsg{}
	})
}

func listTemplatesCmd(api *agent.APIClient, jwt string) tea.Cmd {
	return func() tea.Msg {
		templates, err := api.ListTemplates(jwt)
		return TemplatesListMsg{templates: templates, err: err}
	}
}

func createDeployCmd(api *agent.APIClient, jwt, agentID string, req models.DeployRequest) tea.Cmd {
	return func() tea.Msg {
		deployment, err := api.CreateDeployment(jwt, agentID, req)
		return DeployResultMsg{deployment: deployment, err: err}
	}
}

func (m DeployModel) Init() tea.Cmd {
	return nil
}

func (m DeployModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "up":
			if (m.state == "choose" || m.state == "templates") && m.cursor > 0 {
				m.cursor--
			}
		case "down":
			if m.state == "choose" && m.cursor < len(m.items)-1 {
				m.cursor++
			}
			if m.state == "templates" && m.cursor < len(m.templates)-1 {
				m.cursor++
			}
		case "tab", "shift+tab":
			if m.state == "form" && !m.loading {
				m.focusIndex = (m.focusIndex + 1) % len(m.inputs)
				return m, m.updateFocus()
			}
		case "enter":
			if m.loading {
				return m, nil
			}
			switch m.state {
			case "choose":
				if m.cursor == 0 {
					m.state = "templates"
					m.loading = true
					m.cursor = 0
					return m, tea.Batch(m.spinner.Tick, listTemplatesCmd(m.api, m.config.JWT))
				}
				m.state = "form"
				m.selectedTemplate = nil
				m.buildCustomInputs()
				return m, textinput.Blink

			case "templates":
				tpl := m.templates[m.cursor]
				m.selectedTemplate = &tpl
				m.state = "form"
				m.buildTemplateInputs()
				return m, textinput.Blink

			case "form":
				if m.focusIndex < len(m.inputs)-1 {
					m.focusIndex++
					return m, m.updateFocus()
				}
				req, err := m.buildDeployRequest()
				if err != nil {
					m.err = err
					return m, nil
				}
				m.loading = true
				m.err = nil
				m.state = "deploying"
				return m, tea.Batch(m.spinner.Tick, createDeployCmd(m.api, m.config.JWT, m.config.AgentID, req))
			}
		}

	case TemplatesListMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.state = "choose"
			return m, nil
		}
		if len(msg.templates) == 0 {
			m.err = fmt.Errorf("no templates available")
			m.state = "choose"
			return m, nil
		}
		m.templates = msg.templates
		return m, nil

	case DeployResultMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.state = "form"
			return m, nil
		}
		m.deployID = msg.deployment.ID
		m.state = "polling"
		m.progress = "Starting deployment..."
		return m, tea.Batch(m.spinner.Tick, tickCmd())

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case pollTickMsg:
		if m.state == "polling" {
			return m, pollDeployCmd(m.api, m.config.JWT, m.deployID)
		}

	case PollResultMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = "form"
			return m, nil
		}

		switch msg.deployment.Status {
		case "running":
			m.result = msg.deployment
			m.state = "done"
			return m, tea.Quit
		case "error":
			m.err = fmt.Errorf("deployment failed")
			m.state = "form"
			return m, nil
		default:
			if msg.deployment.Progress != "" {
				m.progress = msg.deployment.Progress
			}
			return m, tickCmd()
		}
	}

	if m.state == "form" {
		cmd := m.updateInputs(msg)
		return m, cmd
	}

	return m, nil
}

func (m *DeployModel) buildTemplateInputs() {
	nameInput := textinput.New()
	nameInput.Placeholder = "my-minecraft"
	nameInput.CharLimit = 64
	nameInput.PromptStyle = FocusedInput
	nameInput.TextStyle = FocusedInput
	nameInput.Focus()

	m.inputs = []textinput.Model{nameInput}
	m.labels = []string{"Deployment Name"}

	for _, p := range m.selectedTemplate.Ports {
		portInput := textinput.New()
		portInput.Placeholder = strconv.Itoa(p.Container)
		portInput.CharLimit = 10
		portInput.PromptStyle = BlurredInput
		portInput.TextStyle = BlurredInput

		label := fmt.Sprintf("Host Port for %s (%d)", p.Name, p.Container)
		m.inputs = append(m.inputs, portInput)
		m.labels = append(m.labels, label)
	}

	envInput := textinput.New()
	envInput.Placeholder = "KEY=VALUE, KEY2=VALUE2"
	envInput.CharLimit = 256
	envInput.PromptStyle = BlurredInput
	envInput.TextStyle = BlurredInput

	m.inputs = append(m.inputs, envInput)
	m.labels = append(m.labels, "Env overrides (optional)")

	m.focusIndex = 0
}

func (m *DeployModel) buildCustomInputs() {
	nameInput := textinput.New()
	nameInput.Placeholder = "my-app"
	nameInput.CharLimit = 64
	nameInput.PromptStyle = FocusedInput
	nameInput.TextStyle = FocusedInput
	nameInput.Focus()

	imageInput := textinput.New()
	imageInput.Placeholder = "nginx:latest"
	imageInput.CharLimit = 128
	imageInput.PromptStyle = BlurredInput
	imageInput.TextStyle = BlurredInput

	portsInput := textinput.New()
	portsInput.Placeholder = "8080:80, 443:443"
	portsInput.CharLimit = 128
	portsInput.PromptStyle = BlurredInput
	portsInput.TextStyle = BlurredInput

	envInput := textinput.New()
	envInput.Placeholder = "KEY=VALUE, KEY2=VALUE2"
	envInput.CharLimit = 256
	envInput.PromptStyle = BlurredInput
	envInput.TextStyle = BlurredInput

	m.inputs = []textinput.Model{nameInput, imageInput, portsInput, envInput}
	m.labels = []string{"Name", "Image", "Ports (host:container)", "Env variables"}
	m.focusIndex = 0
}

func (m *DeployModel) buildDeployRequest() (models.DeployRequest, error) {
	req := models.DeployRequest{}

	if m.selectedTemplate != nil {
		name := strings.TrimSpace(m.inputs[0].Value())
		if name == "" {
			return req, fmt.Errorf("name is required")
		}
		req.Name = name
		req.AppID = &m.selectedTemplate.ID

		for i, p := range m.selectedTemplate.Ports {
			portStr := strings.TrimSpace(m.inputs[1+i].Value())
			hostPort := p.Container
			if portStr != "" {
				parsed, err := strconv.Atoi(portStr)
				if err != nil {
					return req, fmt.Errorf("invalid port: %s", portStr)
				}
				hostPort = parsed
			}
			req.Ports = append(req.Ports, models.PortBinding{
				HostPort:      hostPort,
				ContainerPort: p.Container,
			})
		}

		envIdx := 1 + len(m.selectedTemplate.Ports)
		req.Env = parseEnvString(m.inputs[envIdx].Value())
	} else {
		name := strings.TrimSpace(m.inputs[0].Value())
		if name == "" {
			return req, fmt.Errorf("name is required")
		}
		image := strings.TrimSpace(m.inputs[1].Value())
		if image == "" {
			return req, fmt.Errorf("image is required")
		}
		req.Name = name
		req.Image = &image

		portsStr := strings.TrimSpace(m.inputs[2].Value())
		if portsStr != "" {
			ports, err := parsePortsString(portsStr)
			if err != nil {
				return req, err
			}
			req.Ports = ports
		}

		req.Env = parseEnvString(m.inputs[3].Value())
	}

	return req, nil
}

func parsePortsString(s string) ([]models.PortBinding, error) {
	var ports []models.PortBinding
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		sides := strings.SplitN(part, ":", 2)
		if len(sides) != 2 {
			return nil, fmt.Errorf("invalid port format: %s (expected host:container)", part)
		}
		host, err := strconv.Atoi(strings.TrimSpace(sides[0]))
		if err != nil {
			return nil, fmt.Errorf("invalid host port: %s", sides[0])
		}
		container, err := strconv.Atoi(strings.TrimSpace(sides[1]))
		if err != nil {
			return nil, fmt.Errorf("invalid container port: %s", sides[1])
		}
		ports = append(ports, models.PortBinding{HostPort: host, ContainerPort: container})
	}
	return ports, nil
}

func parseEnvString(s string) map[string]string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	env := make(map[string]string)
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) == 2 {
			env[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}
	return env
}

func (m *DeployModel) updateFocus() tea.Cmd {
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

func (m *DeployModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m DeployModel) View() string {
	var b strings.Builder

	switch m.state {
	case "done":
		b.WriteString(Success.Render("Deployed!"))
		b.WriteString("\n\n")
		if m.result != nil {
			b.WriteString(Subtle.Render("Name:      " + m.result.Name))
			b.WriteString("\n")
			b.WriteString(Subtle.Render("Image:     " + m.result.Image))
			b.WriteString("\n")
			b.WriteString(Subtle.Render("Status:    " + m.result.Status))
			if m.result.ContainerID != nil {
				b.WriteString("\n")
				b.WriteString(Subtle.Render("Container: " + *m.result.ContainerID))
			}
		}
		return Container.Render(b.String())

	case "choose":
		b.WriteString(Title.Render("Create Deployment"))
		b.WriteString("\n")
		for i, item := range m.items {
			if i == m.cursor {
				b.WriteString(FocusedInput.Render("> " + item))
			} else {
				b.WriteString("  " + item)
			}
			b.WriteString("\n")
		}
		if m.err != nil {
			b.WriteString("\n")
			b.WriteString(Error.Render(m.err.Error()))
		}
		b.WriteString("\n")
		b.WriteString(Subtle.Render("↑/↓: navigate • enter: select • esc: back"))

	case "templates":
		if m.loading {
			b.WriteString(m.spinner.View() + " Loading templates...")
			return Container.Render(b.String())
		}
		b.WriteString(Title.Render("Select Template"))
		b.WriteString("\n")
		for i, tpl := range m.templates {
			line := tpl.Name + " — " + tpl.Description
			if i == m.cursor {
				b.WriteString(FocusedInput.Render("> " + line))
			} else {
				b.WriteString("  " + line)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(Subtle.Render("↑/↓: navigate • enter: select • esc: back"))

	case "form":
		if m.selectedTemplate != nil {
			b.WriteString(Title.Render("Deploy: " + m.selectedTemplate.Name))
		} else {
			b.WriteString(Title.Render("Deploy: Custom Image"))
		}
		b.WriteString("\n")

		for i, input := range m.inputs {
			b.WriteString(InputLabel.Render(m.labels[i]))
			b.WriteString("\n")
			b.WriteString(input.View())
			b.WriteString("\n")
		}

		if m.err != nil {
			b.WriteString("\n")
			b.WriteString(Error.Render(m.err.Error()))
		}
		b.WriteString("\n")
		b.WriteString(Subtle.Render("tab: switch field • enter: submit • esc: back"))

	case "deploying", "polling":
		b.WriteString(m.spinner.View() + " " + m.progress)
	}

	return Container.Render(b.String())
}
