package ui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var promptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212")).Bold(true)

// InputModel is a Bubble Tea model for text input.
type InputModel struct {
	input    textinput.Model
	label    string
	value    string
	done     bool
	canceled bool
}

func NewInput(label, placeholder string, mask bool) InputModel {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Focus()
	if mask {
		ti.EchoMode = textinput.EchoPassword
	}
	return InputModel{input: ti, label: label}
}

func (m InputModel) Init() tea.Cmd { return textinput.Blink }

func (m InputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.value = m.input.Value()
			m.done = true
			return m, tea.Quit
		case "esc", "ctrl+c":
			m.canceled = true
			m.done = true
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m InputModel) View() string {
	if m.done {
		return ""
	}
	return promptStyle.Render(m.label) + "\n" + m.input.View() + "\n"
}

func (m InputModel) Value() string    { return m.value }
func (m InputModel) Canceled() bool   { return m.canceled }

// RunInput runs the input prompt and returns the value (empty if cancelled).
func RunInput(label, placeholder string, mask bool) string {
	m := NewInput(label, placeholder, mask)
	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return ""
	}
	r := result.(InputModel)
	if r.Canceled() {
		return ""
	}
	return r.Value()
}
