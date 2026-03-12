package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dragodui/my-deploy/internal/agent"
)

type AgentResultMsg struct {
	agentToken string
	machineID  string
	dockerHost string
	err        error
}

type AgentCreateModel struct {
	api        *agent.APIClient
	inputs     []textinput.Model
	focusIndex int
	spinner    spinner.Model
	loading    bool
	err        error
	done       bool
}

func NewAgentCreateModel(api *agent.APIClient) AgentCreateModel {
	agentNameInput := textinput.New()
	agentNameInput.Placeholder = "Agent 1"
	agentNameInput.CharLimit = 64
	agentNameInput.PromptStyle = FocusedInput
	agentNameInput.TextStyle = FocusedInput
	agentNameInput.Focus()

	dockerHostInput := textinput.New()
	dockerHostInput.Placeholder = "Docker Host URL"
	dockerHostInput.CharLimit = 64
	dockerHostInput.PromptStyle = BlurredInput
	dockerHostInput.TextStyle = BlurredInput

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = FocusedInput
	return AgentCreateModel{
		api:     api,
		inputs:  []textinput.Model{agentNameInput, dockerHostInput},
		spinner: s,
	}
}

func registerAgentCmd(api *agent.APIClient, name, dockerHost, jwt string) tea.Cmd {
	return func() tea.Msg {
		machineID := agent.GenerateMachineID()

		agentToken, err := api.RegisterAgent(jwt, name, machineID)

		return AgentResultMsg{agentToken: agentToken, machineID: machineID, dockerHost: dockerHost, err: err}
	}
}

func (m AgentCreateModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m AgentCreateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "tab", "shift+tab":
			if m.loading {
				return m, nil
			}
			m.focusIndex = (m.focusIndex + 1) % len(m.inputs)
			return m, m.updateFocus()

		case "enter":
			if m.loading {
				return m, nil
			}
			if m.focusIndex < len(m.inputs)-1 {
				m.focusIndex++
				return m, m.updateFocus()
			}

			// load config for jwt
			cfg, err := agent.Load()
			if err != nil || cfg == nil {
				m.err = fmt.Errorf("failed to load config: %w", err)
				return m, nil
			}

			if cfg.JWT == "" {
				m.err = fmt.Errorf("unauthorized: please login first")
				return m, nil
			}

			agentName := strings.TrimSpace(m.inputs[0].Value())
			dockerHost := strings.TrimSpace(m.inputs[1].Value())
			if agentName == "" {
				hostname, err := os.Hostname()
				if err != nil {
					agentName = "agent"
				} else {
					agentName = hostname
				}
			}

			m.loading = true
			m.err = nil
			return m, tea.Batch(m.spinner.Tick, registerAgentCmd(m.api, agentName, dockerHost, cfg.JWT))
		}
	case AgentResultMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		// update config with new agent files
		config, err := agent.Load()
		if err != nil || config == nil {
			m.err = fmt.Errorf("failed to load config: %w", err)
			return m, nil
		}
		config.AgentToken = msg.agentToken
		config.MachineID = msg.machineID
		config.DockerHost = msg.dockerHost

		if err := agent.Save(config); err != nil {
			m.err = fmt.Errorf("failed to save config: %w", err)
			return m, nil
		}
		m.done = true
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	cmd := m.updateInputs(msg)
	return m, cmd
}

func (m *AgentCreateModel) updateFocus() tea.Cmd {
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

func (m *AgentCreateModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m AgentCreateModel) View() string {
	if m.done {
		return Container.Render(
			Success.Render("Agent Created Successfully!") + "\n" +
				Subtle.Render("Config saved to ~/.mydeploy/config.json"),
		)
	}
	var b strings.Builder
	b.WriteString(Title.Render("MyDeploy Create Agent"))
	b.WriteString("\n")

	labels := []string{"Agent Name(optional)", "Docker Host(optional)"}
	for i, input := range m.inputs {
		b.WriteString(InputLabel.Render(labels[i]))
		b.WriteString("\n")
		b.WriteString(input.View())
		b.WriteString("\n")
	}

	if m.loading {
		b.WriteString("\n")
		b.WriteString(m.spinner.View() + "Creating Agent...")
	}

	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(Error.Render(m.err.Error()))
	}

	b.WriteString("\n\n")
	b.WriteString(Subtle.Render("tab: switch field • enter: submit • esc: quit"))

	return Container.Render(b.String())
}
