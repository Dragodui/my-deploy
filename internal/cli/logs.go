package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/dragodui/my-deploy/internal/agent"
	"github.com/gorilla/websocket"
)

type LogLineMsg struct {
	line string
	done bool
	err  error
}

type logsInitMsg struct {
	conn    *websocket.Conn
	logFile *os.File
}

type LogsModel struct {
	api      *agent.APIClient
	config   *agent.LocalConfig
	deployID string
	lines    []string
	conn     *websocket.Conn
	logFile  *os.File
	err      error
	ready    bool
}

func NewLogsModel(api *agent.APIClient, config *agent.LocalConfig, deployID string) LogsModel {
	return LogsModel{
		api:      api,
		config:   config,
		deployID: deployID,
	}
}

func (m LogsModel) Init() tea.Cmd {
	return func() tea.Msg {
		conn, err := m.api.ConnectLogs(m.config.JWT, m.deployID)
		if err != nil {
			return LogLineMsg{err: err}
		}

		home, _ := os.UserHomeDir()
		logsDir := filepath.Join(home, ".mydeploy", "logs")
		os.MkdirAll(logsDir, 0755)

		logFile, err := os.OpenFile(
			filepath.Join(logsDir, m.deployID+".log"),
			os.O_CREATE|os.O_WRONLY|os.O_APPEND,
			0644,
		)
		if err != nil {
			conn.Close()
			return LogLineMsg{err: err}
		}

		return logsInitMsg{conn: conn, logFile: logFile}
	}
}

func readLogsCmd(conn *websocket.Conn) tea.Cmd {
	return func() tea.Msg {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return LogLineMsg{err: err, done: true}
		}

		var chunk agent.LogChunk
		if err := json.Unmarshal(msg, &chunk); err != nil {
			return LogLineMsg{err: err}
		}

		return LogLineMsg{line: chunk.Data, done: chunk.Done}
	}
}

func (m LogsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			if m.conn != nil {
				m.conn.Close()
			}
			if m.logFile != nil {
				m.logFile.Close()
			}
			return m, tea.Quit
		}

	case logsInitMsg:
		m.conn = msg.conn
		m.logFile = msg.logFile
		m.ready = true
		return m, readLogsCmd(m.conn)

	case LogLineMsg:
		if msg.err != nil {
			m.err = msg.err
			if m.conn != nil {
				m.conn.Close()
			}
			if m.logFile != nil {
				m.logFile.Close()
			}
			return m, nil
		}

		if msg.done {
			if m.conn != nil {
				m.conn.Close()
			}
			if m.logFile != nil {
				m.logFile.Close()
			}
			return m, nil
		}

		if msg.line != "" {
			m.lines = append(m.lines, msg.line)
			if m.logFile != nil {
				fmt.Fprintln(m.logFile, msg.line)
			}
		}

		return m, readLogsCmd(m.conn)
	}

	return m, nil
}

func (m LogsModel) View() string {
	var b strings.Builder

	title := "Logs: " + m.deployID
	if len(m.deployID) > 8 {
		title = "Logs: " + m.deployID[:8]
	}
	b.WriteString(Title.Render(title))
	b.WriteString("\n\n")

	if m.err != nil {
		b.WriteString(Error.Render(m.err.Error()))
		b.WriteString("\n\n")
		b.WriteString(Subtle.Render("esc: back"))
		return Container.Render(b.String())
	}

	if !m.ready {
		b.WriteString(Subtle.Render("Connecting..."))
		return Container.Render(b.String())
	}

	start := 0
	if len(m.lines) > 30 {
		start = len(m.lines) - 30
	}
	for _, line := range m.lines[start:] {
		b.WriteString(line)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(Subtle.Render("esc: back • logs saved to ~/.mydeploy/logs/"))

	return Container.Render(b.String())
}
