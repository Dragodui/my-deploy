package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dragodui/my-deploy/internal/agent"
	"github.com/dragodui/my-deploy/internal/models"
)

type DeployListMsg struct {
	deployments []models.Deployment
	err         error
}

type DeployDeleteMsg struct {
	err error
}

type DeployListModel struct {
	api         *agent.APIClient
	config      *agent.LocalConfig
	deployments []models.Deployment
	cursor      int
	spinner     spinner.Model
	loading     bool
	err         error
}

func NewDeployListModel(api *agent.APIClient, config *agent.LocalConfig) DeployListModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = FocusedInput

	return DeployListModel{
		api:     api,
		config:  config,
		spinner: s,
		loading: true,
	}
}

func listDeploymentsCmd(api *agent.APIClient, jwt, agentID string) tea.Cmd {
	return func() tea.Msg {
		deployments, err := api.ListDeployments(jwt, agentID)
		return DeployListMsg{deployments: deployments, err: err}
	}
}

func (m DeployListModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, listDeploymentsCmd(m.api, m.config.JWT, m.config.AgentID))
}

func (m DeployListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			return m, tea.Quit
		case "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down":
			if m.cursor < len(m.deployments)-1 {
				m.cursor++
			}
		}

	case DeployListMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.deployments = msg.deployments
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m DeployListModel) View() string {
	var b strings.Builder

	b.WriteString(Title.Render("Deployments"))
	b.WriteString("\n")

	if m.loading {
		b.WriteString(m.spinner.View() + " Loading deployments...")
		return Container.Render(b.String())
	}

	if m.err != nil {
		b.WriteString(Error.Render(m.err.Error()))
		b.WriteString("\n\n")
		b.WriteString(Subtle.Render("esc: back"))
		return Container.Render(b.String())
	}

	if len(m.deployments) == 0 {
		b.WriteString(Subtle.Render("No deployments yet."))
		b.WriteString("\n\n")
		b.WriteString(Subtle.Render("esc: back"))
		return Container.Render(b.String())
	}

	for i, d := range m.deployments {
		status := d.Status
		var statusStyle func(strs ...string) string
		switch status {
		case "running":
			statusStyle = Success.Render
		case "error", "failed":
			statusStyle = Error.Render
		default:
			statusStyle = Subtle.Render
		}

		line := fmt.Sprintf("%-20s %-25s %s", d.Name, d.Image, statusStyle(status))
		if i == m.cursor {
			b.WriteString(FocusedInput.Render("> ") + line)
		} else {
			b.WriteString("  " + line)
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(Subtle.Render("↑/↓: navigate • esc: back"))

	return Container.Render(b.String())
}
