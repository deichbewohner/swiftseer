package ui

import "github.com/charmbracelet/lipgloss"

var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99"))

	SuccessStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	LabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	InfoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("246"))

	HighlightStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86"))

	progressBarStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("86"))

	statsStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("246"))
)
