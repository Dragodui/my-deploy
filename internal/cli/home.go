package cli

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dragodui/my-deploy/internal/agent"
	"github.com/dragodui/my-deploy/internal/daemon"
)

type HomeModel struct {
	userName     string
	agentName    string
	agentMode    string
	agentRunning bool
	agentPID     int
	items        []string
	cursor       int
	action       string
	err          error
}

func NewHomeModel(config *agent.LocalConfig) HomeModel {
	running, pid := daemon.IsRunning()

	m := HomeModel{
		userName:     config.UserName,
		agentName:    config.AgentName,
		agentMode:    config.AgentMode,
		agentRunning: running,
		agentPID:     pid,
	}
	m.buildMenu()
	return m
}

func (m *HomeModel) buildMenu() {
	m.items = []string{"Change agent", "Deploy", "Deploy list"}
	if m.agentMode != "remote" {
		if m.agentRunning {
			m.items = append(m.items, "Stop agent")
		} else {
			m.items = append(m.items, "Start agent")
		}
	}
	m.items = append(m.items, "Logout")
}

func (m HomeModel) Action() string {
	return m.action
}

func (m HomeModel) Init() tea.Cmd {
	return nil
}

type ChangeAgentMsg struct{}

func (m HomeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "down":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "enter":
			selected := m.items[m.cursor]
			switch selected {
			case "Change agent":
				m.action = "change_agent"
				return m, tea.Quit
			case "Deploy":
				m.action = "deploy"
				return m, tea.Quit
			case "Deploy list":
				m.action = "deploy_list"
				return m, tea.Quit
			case "Start agent":
				binary, err := daemon.FindAgentBinary()
				if err != nil {
					m.err = fmt.Errorf("agent binary not found: %w", err)
					return m, nil
				}
				if err := daemon.StartAgent(binary); err != nil {
					m.err = fmt.Errorf("failed to start agent: %w", err)
					return m, nil
				}
				m.agentRunning, m.agentPID = daemon.IsRunning()
				m.err = nil
				m.buildMenu()
				m.cursor = 0
			case "Stop agent":
				if err := daemon.Stop(); err != nil {
					m.err = fmt.Errorf("failed to stop agent: %w", err)
					return m, nil
				}
				m.agentRunning = false
				m.agentPID = 0
				m.err = nil
				m.buildMenu()
				m.cursor = 0
			case "Logout":
				_ = agent.Delete()
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

const logo = `       .
      ":"                 __  ___     ___           __
    ___:____     |"\/"|  /  |/  /_ __/ _ \___ ___  / /__  __ __
  ,'        ` + "`" + `.    \  /  / /|_/ / // / // / -_) _ \/ / _ \/ // /
  |  O        \___/  | /_/  /_/\_, /____/\__/ .__/_/\___/\_, /
~^~^~^~^~^~^~^~^~^~^~^~       /___/        /_/          /___/`

func (m HomeModel) View() string {
	var b strings.Builder

	b.WriteString(FocusedInput.Render(logo))
	b.WriteString("\n\n")

	if m.userName != "" {
		b.WriteString(Title.Render("Welcome, " + m.userName + "!"))
	} else {
		b.WriteString(Title.Render("Welcome!"))
	}
	b.WriteString("\n")

	agentInfo := "Agent: " + m.agentName
	if m.agentMode == "remote" {
		agentInfo += " (remote)"
	} else if m.agentRunning {
		agentInfo += fmt.Sprintf(" (running, PID %d)", m.agentPID)
	} else {
		agentInfo += " (not running)"
	}
	b.WriteString(Subtle.Render(agentInfo))

	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(Error.Render(m.err.Error()))
	}

	b.WriteString("\n\n")

	for i, item := range m.items {
		if i == m.cursor {
			b.WriteString(FocusedInput.Render("> " + item))
		} else {
			b.WriteString("  " + item)
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(Subtle.Render("↑/↓: navigate • enter: select • q: quit"))

	return Container.Render(b.String())
}
