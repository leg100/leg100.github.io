package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct{}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, func() tea.Msg {
		panic("error")
	}
}

func (m model) View() string { return "" }

func main() {
	p := tea.NewProgram(model{}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("could not start program:", err)
		os.Exit(1)
	}
}
