package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	content string
}

func (m *model) Init() tea.Cmd {
	go func() {
		<-time.After(time.Second)
		m.content = "initialized\n"
	}()
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case msg.String() == "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *model) View() string { return m.content }

func main() {
	p := tea.NewProgram(&model{content: "uninitalized"})
	if _, err := p.Run(); err != nil {
		fmt.Println("could not start program:", err)
		os.Exit(1)
	}
}
