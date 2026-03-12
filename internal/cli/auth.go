package cli

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dragodui/my-deploy/internal/agent"
)

type SwitchToRegisterMsg struct{}
type SwitchToLoginMsg struct{}

type AuthModel struct {
	current tea.Model
	api     *agent.APIClient
}

func NewAuthModel(api *agent.APIClient) AuthModel {
	return AuthModel{
		current: NewLoginModel(api),
		api:     api,
	}
}

func (m AuthModel) Init() tea.Cmd {
	return m.current.Init()
}

func (m AuthModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case SwitchToRegisterMsg:
		m.current = NewRegisterModel(m.api)
		return m, m.current.Init()
	case SwitchToLoginMsg:
		m.current = NewLoginModel(m.api)
		return m, m.current.Init()
	}

	var cmd tea.Cmd
	m.current, cmd = m.current.Update(msg)
	return m, cmd
}

func (m AuthModel) View() string {
	return m.current.View()
}
