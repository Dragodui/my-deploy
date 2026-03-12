package cli

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/dragodui/my-deploy/internal/agent"
)

func signUpCmd(api *agent.APIClient, email, name, password string) tea.Cmd {
	return func() tea.Msg {
		token, userName, err := api.SignUp(email, name, password)
		return AuthResultMsg{token: token, name: userName, err: err}
	}
}

type RegisterModel struct {
	api        *agent.APIClient
	inputs     []textinput.Model
	focusIndex int
	spinner    spinner.Model
	loading    bool
	err        error
	done       bool
}

func NewRegisterModel(api *agent.APIClient) *RegisterModel {
	emailInput := textinput.New()
	emailInput.Placeholder = "email@example.com"
	emailInput.CharLimit = 64
	emailInput.PromptStyle = FocusedInput
	emailInput.TextStyle = FocusedInput
	emailInput.Focus()

	passwordInput := textinput.New()
	passwordInput.Placeholder = "password"
	passwordInput.CharLimit = 64
	passwordInput.EchoMode = textinput.EchoPassword
	passwordInput.EchoCharacter = '•'
	passwordInput.PromptStyle = BlurredInput
	passwordInput.TextStyle = BlurredInput

	nameInput := textinput.New()
	nameInput.Placeholder = "Alex"
	nameInput.CharLimit = 64
	nameInput.PromptStyle = BlurredInput
	nameInput.TextStyle = BlurredInput

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = FocusedInput

	return &RegisterModel{
		api:     api,
		inputs:  []textinput.Model{emailInput, passwordInput, nameInput},
		spinner: s,
	}
}

func (m RegisterModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m RegisterModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "ctrl+l":
			return m, func() tea.Msg { return SwitchToLoginMsg{} }

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
			email := strings.TrimSpace(m.inputs[0].Value())
			password := m.inputs[1].Value()
			name := strings.TrimSpace(m.inputs[2].Value())
			if email == "" || password == "" || name == "" {
				m.err = fmt.Errorf("email, password and name are required")
				return m, nil
			}
			m.loading = true
			m.err = nil
			return m, tea.Batch(m.spinner.Tick, signUpCmd(m.api, email, name, password))
		}
	case AuthResultMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		config := &agent.LocalConfig{
			URL:      agent.DefaultServerURL,
			JWT:      msg.token,
			UserName: msg.name,
		}
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

func (m *RegisterModel) updateFocus() tea.Cmd {
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

func (m *RegisterModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m RegisterModel) View() string {
	if m.done {
		return Container.Render(
			Success.Render("Authenticated!") + "\n" +
				Subtle.Render("Token saved to ~/.mydeploy/config.json"),
		)
	}

	var b strings.Builder

	b.WriteString(Title.Render("MyDeploy Sign Up"))
	b.WriteString("\n")

	labels := []string{"Email", "Password", "Name"}
	for i, input := range m.inputs {
		b.WriteString(InputLabel.Render(labels[i]))
		b.WriteString("\n")
		b.WriteString(input.View())
		b.WriteString("\n")
	}

	if m.loading {
		b.WriteString("\n")
		b.WriteString(m.spinner.View() + " Signing up...")
	}

	if m.err != nil {
		b.WriteString("\n")
		b.WriteString(Error.Render(m.err.Error()))
	}

	b.WriteString("\n\n")
	b.WriteString(Subtle.Render("tab: switch field • enter: submit • ctrl+l: sign in • esc: quit"))

	return Container.Render(b.String())
}
