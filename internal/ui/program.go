package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// AgentSendFunc sends a message to the agent. Output arrives via the OutputFunc set on the agent.
type AgentSendFunc func(ctx context.Context, input string) (string, error)

// outputMsg carries agent output into the Bubble Tea update loop.
type outputMsg struct {
	kind string
	text string
}

// agentDoneMsg signals the agent finished.
type agentDoneMsg struct{ err error }

// Model is the main Bubble Tea TUI model.
type Model struct {
	input    textinput.Model
	spinner  spinner.Model
	history  []string
	cmdHist  []string
	histIdx  int
	thinking bool
	width    int
	height   int
	banner   string
	agentFn  AgentSendFunc
	outputCh chan outputMsg
	quitting bool
}

// NewProgram creates and returns the Bubble Tea program and an OutputFunc
// that the agent should use to emit text.
func NewProgram(banner string, agentFn AgentSendFunc) (*tea.Program, func(string, string)) {
	ch := make(chan outputMsg, 64)
	m := newModel(banner, agentFn, ch)
	p := tea.NewProgram(m, tea.WithAltScreen())

	outputFn := func(kind, text string) {
		ch <- outputMsg{kind: kind, text: text}
		// Send a nil message to wake up the program
		p.Send(nil)
	}

	return p, outputFn
}

func newModel(banner string, agentFn AgentSendFunc, ch chan outputMsg) Model {
	ti := textinput.New()
	ti.Prompt = StylePrompt.Render("❯ ")
	ti.Focus()
	ti.CharLimit = 4096

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("75"))

	return Model{
		input:    ti,
		spinner:  sp,
		history:  []string{},
		cmdHist:  []string{},
		histIdx:  -1,
		banner:   banner,
		agentFn:  agentFn,
		outputCh: ch,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Drain output channel
	m.drainOutput()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if m.thinking {
			if msg.String() == "ctrl+c" {
				m.quitting = true
				return m, tea.Quit
			}
			return m, nil
		}
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			val := strings.TrimSpace(m.input.Value())
			if val == "" {
				return m, nil
			}
			m.input.SetValue("")
			m.histIdx = -1

			if val == "/exit" {
				m.quitting = true
				return m, tea.Quit
			}

			m.cmdHist = append(m.cmdHist, val)
			m.history = append(m.history, StylePrompt.Render("❯ ")+val)

			if m.handleCommand(val) {
				return m, nil
			}

			m.thinking = true
			return m, m.runAgent(val)

		case "up":
			if len(m.cmdHist) == 0 {
				return m, nil
			}
			if m.histIdx == -1 {
				m.histIdx = len(m.cmdHist) - 1
			} else if m.histIdx > 0 {
				m.histIdx--
			}
			m.input.SetValue(m.cmdHist[m.histIdx])
			m.input.CursorEnd()
			return m, nil

		case "down":
			if m.histIdx == -1 {
				return m, nil
			}
			if m.histIdx < len(m.cmdHist)-1 {
				m.histIdx++
				m.input.SetValue(m.cmdHist[m.histIdx])
			} else {
				m.histIdx = -1
				m.input.SetValue("")
			}
			m.input.CursorEnd()
			return m, nil
		}

	case agentDoneMsg:
		m.drainOutput()
		m.thinking = false
		if msg.err != nil {
			m.history = append(m.history, StyleError.Render("  error: "+msg.err.Error()))
		}
		m.history = append(m.history, "")
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *Model) drainOutput() {
	for {
		select {
		case o := <-m.outputCh:
			switch o.kind {
			case "tool":
				m.history = append(m.history, StyleTool.Render("  [tool] "+o.text))
			default:
				m.history = append(m.history, "  "+o.text)
			}
		default:
			return
		}
	}
}

func (m *Model) handleCommand(input string) bool {
	switch input {
	case "/clear":
		m.history = m.history[:0]
		m.history = append(m.history, StyleDim.Render("  Historial limpiado."))
		return true
	case "/help":
		m.history = append(m.history,
			"  /clear   — Limpia historial",
			"  /tools   — Lista herramientas",
			"  /exit    — Salir",
		)
		return true
	}
	return false
}

func (m Model) runAgent(input string) tea.Cmd {
	return func() tea.Msg {
		_, err := m.agentFn(context.Background(), input)
		return agentDoneMsg{err: err}
	}
}

func (m Model) View() string {
	if m.quitting {
		return StyleDim.Render("¡Hasta luego!") + "\n"
	}

	var sb strings.Builder
	sb.WriteString(m.banner)
	sb.WriteString("\n")

	// Viewport: show last N lines
	viewportH := m.height - strings.Count(m.banner, "\n") - 4
	if viewportH < 10 {
		viewportH = 20
	}
	start := 0
	if len(m.history) > viewportH {
		start = len(m.history) - viewportH
	}
	for _, line := range m.history[start:] {
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	if m.thinking {
		sb.WriteString(fmt.Sprintf("  %s pensando...\n", m.spinner.View()))
	}

	sb.WriteString(m.input.View())
	return sb.String()
}
