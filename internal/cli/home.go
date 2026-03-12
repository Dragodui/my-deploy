package cli

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dragodui/my-deploy/internal/agent"
)

type HomeModel struct {
	userName  string
	agentName string
	items     []string
	cursor    int
	action    string
}

func NewHomeModel(config *agent.LocalConfig) HomeModel {
	return HomeModel{
		userName:  config.UserName,
		agentName: config.AgentName,
		items:     []string{"Change agent", "Deploy", "Deploy list", "Logout"},
	}
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
			switch m.cursor {
			case 0: // Change agent
				m.action = "change_agent"
				return m, tea.Quit
			case 1: // Deploy
				// todo
			case 2: // Deploy list
				// todo
			case 3: // Logout
				_ = agent.Delete()
				return m, tea.Quit
			}
		}
	}
	return m, nil
}

const logo = `   __  ___     ___           __
  /  |/  /_ __/ _ \___ ___  / /__  __ __
 / /|_/ / // / // / -_) _ \/ / _ \/ // /
/_/  /_/\_, /____/\__/ .__/_/\___/\_, /
       /___/        /_/          /___/  `

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

	if m.agentName != "" {
		b.WriteString(Subtle.Render("Agent: " + m.agentName))
	} else {
		b.WriteString(Subtle.Render("Agent: (none)"))
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
