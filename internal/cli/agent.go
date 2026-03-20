package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dragodui/my-deploy/internal/agent"
	"github.com/dragodui/my-deploy/internal/daemon"
	"github.com/dragodui/my-deploy/internal/shared/models"
)

type AgentCreateResultMsg struct {
	agentID    string
	agentToken string
	agentName  string
	machineID  string
	dockerHost string
	err        error
}

type AgentListResultMsg struct {
	agents []models.Agent
	err    error
}

type AgentCreateModel struct {
	api          *agent.APIClient
	inputs       []textinput.Model
	focusIndex   int
	spinner      spinner.Model
	state        string
	agents       []models.Agent
	cursor       int
	loading      bool
	err          error
	done         bool
	selectedName string
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
		state:   "loading",
	}
}

func registerAgentCmd(api *agent.APIClient, name, dockerHost, jwt string) tea.Cmd {
	return func() tea.Msg {
		machineID := agent.GenerateMachineID()

		agentID, agentToken, err := api.RegisterAgent(jwt, name, machineID)

		return AgentCreateResultMsg{agentID: agentID, agentToken: agentToken, agentName: name, machineID: machineID, dockerHost: dockerHost, err: err}
	}
}

func listAgentsCmd(api *agent.APIClient, jwt string) tea.Cmd {
	return func() tea.Msg {
		agents, err := api.ListAgents(jwt)
		return AgentListResultMsg{
			agents, err,
		}
	}
}

func (m AgentCreateModel) Init() tea.Cmd {
	cfg, _ := agent.Load()
	if cfg != nil && cfg.JWT != "" {
		return tea.Batch(m.spinner.Tick, listAgentsCmd(m.api, cfg.JWT))
	}
	return textinput.Blink
}

func (m AgentCreateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "tab", "shift+tab":
			if m.loading || m.state != "create" {
				return m, nil
			}
			m.focusIndex = (m.focusIndex + 1) % len(m.inputs)
			return m, m.updateFocus()

		case "enter":
			if m.loading {
				return m, nil
			}

			if m.state == "list" {
				if m.cursor == len(m.agents) {
					m.state = "create"
					return m, textinput.Blink
				}
				selected := m.agents[m.cursor]
				config, err := agent.Load()
				if err != nil || config == nil {
					m.err = fmt.Errorf("failed to load config: %w", err)
					return m, nil
				}
				config.AgentID = selected.ID
				config.AgentToken = selected.Token
				config.MachineID = selected.MachineID
				config.AgentName = selected.Name
				if err := agent.Save(config); err != nil {
					m.err = fmt.Errorf("failed to save config: %w", err)
					return m, nil
				}
				m.selectedName = selected.Name
				m.done = true
				return m, tea.Quit
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
		case "down":
			if m.state == "list" && m.cursor < len(m.agents) {
				m.cursor++
			}
		case "up":
			if m.state == "list" && m.cursor > 0 {
				m.cursor--
			}
		}
	case AgentListResultMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.state = "create"
			return m, textinput.Blink
		}
		if len(msg.agents) == 0 {
			m.state = "create"
			return m, textinput.Blink
		}
		m.agents = msg.agents
		m.state = "list"
		return m, nil
	case AgentCreateResultMsg:
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
		config.AgentID = msg.agentID
		config.AgentToken = msg.agentToken
		config.AgentName = msg.agentName
		config.MachineID = msg.machineID
		config.DockerHost = msg.dockerHost

		if err := agent.Save(config); err != nil {
			m.err = fmt.Errorf("failed to save config: %w", err)
			return m, nil
		}

		// run agent
		binary, err := daemon.FindAgentBinary()
		if err == nil {
			daemon.StartAgent(binary)
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
		doneMsg := "Agent created!"
		if m.selectedName != "" {
			doneMsg = "Agent selected: " + m.selectedName
		}
		// guide to run agent remotely
		cfg, _ := agent.Load()
		remoteHint := ""
		if m.selectedName == "" && cfg != nil {
			remoteHint = "\n\n" + Subtle.Render(
				"To run on a remote server:\n  mydeploy-agent --url "+cfg.URL+" --token "+cfg.AgentToken)
		}
		return Container.Render(
			Success.Render(doneMsg) + "\n" +
				Subtle.Render("Config saved to ~/.mydeploy/config.json") +
				remoteHint,
		)
	}
	if m.state == "loading" {
		return Container.Render(m.spinner.View() + " Loading agents...")
	}
	if m.state == "list" {
		var b strings.Builder
		b.WriteString(Title.Render("Select Agent"))
		b.WriteString("\n")
		for i, ag := range m.agents {
			if i == m.cursor {
				b.WriteString(FocusedInput.Render("> " + ag.Name))
			} else {
				b.WriteString("  " + ag.Name)
			}
			b.WriteString("\n")
		}

		if m.cursor == len(m.agents) {
			b.WriteString(FocusedInput.Render("> + Create new agent"))
		} else {
			b.WriteString("  + Create new agent")
		}
		b.WriteString("\n\n")
		b.WriteString(Subtle.Render("↑/↓: navigate • enter: select • esc: quit"))
		return Container.Render(b.String())
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
