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
	api          *agent.APIClient
	config       *agent.LocalConfig
	deployments  []models.Deployment
	state        string // list or actions
	selected     *models.Deployment
	action       string
	actions      []string
	actionCursor int
	cursor       int
	spinner      spinner.Model
	loading      bool
	err          error
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
		state:   "list",
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

func (m DeployListModel) Action() string {
	return m.action
}
func (m DeployListModel) SelectedDeploy() *models.Deployment {
	return m.selected
}

func (m DeployListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if m.loading {
				return m, nil
			}
			if m.state == "list" {
				if len(m.deployments) == 0 {
					return m, nil
				}
				m.selected = &m.deployments[m.cursor]
				actions := make([]string, 3)
				status := m.deployments[m.cursor].Status

				// update available statuses
				if status == "running" {
					actions = []string{"Logs", "Stop", "Delete", "Back"}
				} else if status == "exited" {
					actions = []string{"Start", "Delete", "Back"}
				} else {
					actions = []string{"Delete", "Back"}
				}
				m.actions = actions

				// reset cursor
				m.state = "actions"
				m.actionCursor = 0
			} else {
				selected := m.actions[m.actionCursor]
				switch selected {
				case "Start":
					err := m.api.StartDeployment(m.config.JWT, m.selected.ID)
					if err != nil {
						m.err = err
						return m, nil
					}
					m.state = "list"
					m.loading = true
					return m, tea.Batch(m.spinner.Tick, listDeploymentsCmd(m.api, m.config.JWT, m.config.AgentID))
				case "Stop":
					err := m.api.StopDeployment(m.config.JWT, m.selected.ID)
					if err != nil {
						m.err = err
						return m, nil
					}
					m.state = "list"
					m.loading = true
					return m, tea.Batch(m.spinner.Tick, listDeploymentsCmd(m.api, m.config.JWT, m.config.AgentID))
				case "Delete":
					err := m.api.DeleteDeployment(m.config.JWT, m.selected.ID)
					if err != nil {
						m.err = err
						return m, nil
					}
					m.state = "list"
					m.loading = true
					return m, tea.Batch(m.spinner.Tick, listDeploymentsCmd(m.api, m.config.JWT, m.config.AgentID))
				case "Logs":
					m.action = "logs"
					m.state = "list"
					return m, tea.Quit
				case "Back":
					m.state = "list"
				}
			}
		case "ctrl+c", "esc", "q":
			if m.state == "actions" {
				m.state = "list"
			} else {
				return m, tea.Quit
			}
		case "up":
			if m.state == "list" && m.cursor > 0 {
				m.cursor--
			} else if m.state == "actions" && m.actionCursor > 0 {
				m.actionCursor--
			}
		case "down":
			if m.state == "list" && m.cursor < len(m.deployments)-1 {
				m.cursor++
			} else if m.state == "actions" && m.actionCursor < len(m.actions)-1 {
				m.actionCursor++
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

	if m.state == "actions" && m.selected != nil {
		b.WriteString(Title.Render(m.selected.Name))
		b.WriteString("\n")
		for i, action := range m.actions {
			if i == m.actionCursor {
				b.WriteString(FocusedInput.Render("> " + action))
			} else {
				b.WriteString("  " + action)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(Subtle.Render("↑/↓: navigate • enter: select • esc: back"))
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
