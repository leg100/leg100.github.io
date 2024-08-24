package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	width, height int
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.String() == "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m model) View() string {
	header := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Width(m.width).
		Render("header")
	footer := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Width(m.width).
		Render("footer")

	content := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height-1-1).
		Align(lipgloss.Center, lipgloss.Center).
		Render("content")

	return lipgloss.JoinVertical(lipgloss.Top, header, content, footer)
}

func main() {
	p := tea.NewProgram(model{}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("could not start program:", err)
		os.Exit(1)
	}
}
