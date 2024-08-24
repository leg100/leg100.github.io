package main

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestQuit(t *testing.T) {
	m := model{}
	tm := teatest.NewTestModel(t, m)

	waitForString(t, tm, "Running.")

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})

	waitForString(t, tm, "Quit? (y/N)")

	tm.Type("y")

	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func waitForString(t *testing.T, tm *teatest.TestModel, s string) {
	teatest.WaitFor(
		t,
		tm.Output(),
		func(b []byte) bool {
			return strings.Contains(string(b), s)
		},
		teatest.WithCheckInterval(time.Millisecond*100),
		teatest.WithDuration(time.Second*10),
	)
}
