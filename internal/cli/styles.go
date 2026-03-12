package cli

import "github.com/charmbracelet/lipgloss"

var (
	ColorAccent  = lipgloss.Color("99")
	ColorSuccess = lipgloss.Color("78")
	ColorError   = lipgloss.Color("196")
	ColorSubtle  = lipgloss.Color("241")
	ColorText    = lipgloss.Color("252")

	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorAccent).
		MarginBottom(1)

	Error = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorError)

	Success = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorSuccess)

	Subtle = lipgloss.NewStyle().
		Foreground(ColorSubtle)

	InputLabel = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorText).
			MarginTop(1)

	FocusedInput = lipgloss.NewStyle().
			Foreground(ColorAccent)

	BlurredInput = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	Button = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("0")).
		Background(ColorAccent).
		Padding(0, 2)

	BlurredButton = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("238")).
			Padding(0, 2)

	Container = lipgloss.NewStyle().
			Padding(1, 2)
)
