package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	quitting bool
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.String() == "ctrl+c":
			m.quitting = true
			return m, nil
		}
		if m.quitting {
			switch {
			case msg.String() == "y":
				return m, tea.Quit
			default:
				m.quitting = false
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return "Quit? (y/N)"
	} else {
		return "Running."
	}
}

func main() {
	p := tea.NewProgram(model{})
	if _, err := p.Run(); err != nil {
		fmt.Println("could not start program:", err)
		os.Exit(1)
	}
}
