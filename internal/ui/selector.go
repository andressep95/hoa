package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("33"))
	cursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Bold(true)
)

// SelectorItem is one option in the selector.
type SelectorItem struct {
	Label string
	Hint  string // e.g. "← activo", "← configurado"
}

// SelectorModel is a Bubble Tea model for arrow-key selection.
type SelectorModel struct {
	Title    string
	Items    []SelectorItem
	cursor   int
	selected int
	done     bool
}

func NewSelector(title string, items []SelectorItem) SelectorModel {
	return SelectorModel{Title: title, Items: items, selected: -1}
}

func (m SelectorModel) Init() tea.Cmd { return nil }

func (m SelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.Items)-1 {
				m.cursor++
			}
		case "enter":
			m.selected = m.cursor
			m.done = true
			return m, tea.Quit
		case "esc", "ctrl+c":
			m.selected = -1
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m SelectorModel) View() string {
	if m.done {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(m.Title + "\n\n")
	for i, item := range m.Items {
		cursor := "  "
		style := StyleDim
		if i == m.cursor {
			cursor = cursorStyle.Render("▸ ")
			style = selectedStyle
		}
		line := style.Render(item.Label)
		if item.Hint != "" {
			line += StyleDim.Render("  " + item.Hint)
		}
		sb.WriteString(fmt.Sprintf("%s%s\n", cursor, line))
	}
	sb.WriteString(StyleDim.Render("\n  ↑↓ navegar · enter seleccionar · esc cancelar"))
	return sb.String()
}

// Selected returns the index chosen, or -1 if cancelled.
func (m SelectorModel) Selected() int { return m.selected }

// RunSelector runs the selector and returns the chosen index (-1 if cancelled).
func RunSelector(title string, items []SelectorItem) int {
	m := NewSelector(title, items)
	p := tea.NewProgram(m)
	result, err := p.Run()
	if err != nil {
		return -1
	}
	return result.(SelectorModel).Selected()
}
