package ui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/cloudcentinel/hoa/internal/command"
)

// AgentSendFunc sends a message to the agent.
type AgentSendFunc func(ctx context.Context, input string) (string, error)

type outputMsg struct{ kind, text string }
type agentDoneMsg struct{ err error }
type asyncCmdDoneMsg struct{ result command.Result }

// Model is the main Bubble Tea TUI model.
type Model struct {
	input   textinput.Model
	spinner spinner.Model

	history  []string
	cmdHist  []string
	histIdx  int
	scroll   int // scroll offset from bottom (0 = latest)
	thinking bool
	width    int
	height   int
	banner   func() string

	agentFn  AgentSendFunc
	cmdCtx   *command.Context
	outputCh chan outputMsg
	quitting bool

	// Interactive menu state
	menuActive bool
	menuTitle  string
	menuItems  []command.MenuItem
	menuCursor int

	// Autocomplete state
	acActive bool
	acItems  []string
	acCursor int
}

// NewProgram creates the Bubble Tea program and returns an output function for the agent.
func NewProgram(banner func() string, agentFn AgentSendFunc, cmdCtx *command.Context) (*tea.Program, func(string, string)) {
	ch := make(chan outputMsg, 64)
	m := newModel(banner, agentFn, cmdCtx, ch)
	p := tea.NewProgram(m, tea.WithAltScreen())

	outputFn := func(kind, text string) {
		ch <- outputMsg{kind, text}
		p.Send(nil)
	}
	return p, outputFn
}

func newModel(banner func() string, agentFn AgentSendFunc, cmdCtx *command.Context, ch chan outputMsg) Model {
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
		cmdCtx:   cmdCtx,
		outputCh: ch,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.drainOutput()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Menu mode
		if m.menuActive {
			return m.updateMenu(msg)
		}
		// Autocomplete mode — intercept navigation keys only
		if m.acActive {
			switch msg.String() {
			case "down":
				if m.acCursor < len(m.acItems)-1 {
					m.acCursor++
				}
				return m, nil
			case "up":
				if m.acCursor > 0 {
					m.acCursor--
				}
				return m, nil
			case "tab":
				m.input.SetValue("/" + m.acItems[m.acCursor])
				m.input.CursorEnd()
				m.acActive = false
				return m, nil
			case "enter":
				// If autocomplete has a suggestion, execute it instead of the partial
				if len(m.acItems) > 0 {
					val := "/" + m.acItems[m.acCursor]
					m.input.SetValue("")
					m.acActive = false
					m.histIdx = -1
					m.cmdHist = append(m.cmdHist, val)
					m.history = append(m.history, StylePrompt.Render("❯ ")+val)
					return m.executeCommand(val)
				}
				m.acActive = false
				// Fall through to normal enter handling
			case "esc":
				m.acActive = false
				return m, nil
			default:
				// Pass to input, then update suggestions
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				m.updateAutocompleteState()
				return m, cmd
			}
		}
		// Thinking — only allow ctrl+c
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
			m.acActive = false
			m.cmdHist = append(m.cmdHist, val)
			m.history = append(m.history, StylePrompt.Render("❯ ")+val)

			return m.executeCommand(val)

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

		case "pgup":
			m.scroll += 10
			max := len(m.history) - 5
			if m.scroll > max {
				m.scroll = max
			}
			if m.scroll < 0 {
				m.scroll = 0
			}
			return m, nil

		case "pgdown":
			m.scroll -= 10
			if m.scroll < 0 {
				m.scroll = 0
			}
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

	case asyncCmdDoneMsg:
		m.thinking = false
		result := msg.result
		if len(result.Menu) > 0 {
			if len(result.Lines) > 0 {
				m.history = append(m.history, result.Lines...)
			}
			m.menuActive = true
			m.menuTitle = result.Title
			m.menuItems = result.Menu
			m.menuCursor = 0
			return m, nil
		}
		m.history = append(m.history, result.Lines...)
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	// Update text input
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)

	// Autocomplete trigger
	m.updateAutocompleteState()

	return m, cmd
}

// ── Menu handling ───────────────────────────────────────────────────────────

func (m Model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		for i := m.menuCursor - 1; i >= 0; i-- {
			if m.menuItems[i].Action != nil {
				m.menuCursor = i
				break
			}
		}
	case "down", "j":
		for i := m.menuCursor + 1; i < len(m.menuItems); i++ {
			if m.menuItems[i].Action != nil {
				m.menuCursor = i
				break
			}
		}
	case "enter":
		item := m.menuItems[m.menuCursor]
		if item.Action != nil {
			item.Action()
			m.history = append(m.history, StyleDim.Render("  ⎿  "+item.Label), "")
		}
		m.menuActive = false
	case "esc", "ctrl+c", "q":
		m.menuActive = false
		return m, nil
	}
	return m, nil
}

// ── Autocomplete handling (ghost text) ──────────────────────────────────────

func (m *Model) updateAutocompleteState() {
	val := m.input.Value()
	if strings.HasPrefix(val, "/") && !strings.Contains(val, " ") {
		prefix := strings.ToLower(val[1:])
		names := command.Names()
		sort.Strings(names)

		var filtered []string
		if prefix == "" {
			// Show all commands on bare /
			filtered = names
		} else {
			// Fuzzy: prefix match first, then substring
			for _, n := range names {
				if strings.HasPrefix(n, prefix) {
					filtered = append(filtered, n)
				}
			}
			if len(filtered) == 0 {
				for _, n := range names {
					if strings.Contains(n, prefix) {
						filtered = append(filtered, n)
					}
				}
			}
			// Exact match — hide autocomplete
			if len(filtered) == 1 && filtered[0] == prefix {
				m.acActive = false
				m.acItems = nil
				return
			}
		}

		if len(filtered) > 0 {
			m.acActive = true
			m.acItems = filtered
			if m.acCursor >= len(filtered) {
				m.acCursor = 0
			}
			return
		}
	}
	m.acActive = false
	m.acItems = nil
	m.acCursor = 0
}


// ── Other ───────────────────────────────────────────────────────────────────

// executeCommand dispatches a slash command and handles the result.
func (m Model) executeCommand(val string) (tea.Model, tea.Cmd) {
	if result, handled := command.Dispatch(m.cmdCtx, val); handled {
		if result.Quit {
			m.quitting = true
			return m, tea.Quit
		}
		if result.AsyncFn != nil {
			if len(result.Lines) > 0 {
				m.history = append(m.history, result.Lines...)
			}
			m.thinking = true
			fn := result.AsyncFn
			return m, func() tea.Msg {
				return asyncCmdDoneMsg{result: fn()}
			}
		}
		if len(result.Menu) > 0 {
			if len(result.Lines) > 0 {
				m.history = append(m.history, result.Lines...)
			}
			m.menuActive = true
			m.menuTitle = result.Title
			m.menuItems = result.Menu
			m.menuCursor = 0
			return m, nil
		}
		m.history = append(m.history, result.Lines...)
		m.history = append(m.history, "") // spacing
		return m, nil
	}
	// Not a command — send to agent
	m.thinking = true
	return m, m.runAgent(val)
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

func (m Model) runAgent(input string) tea.Cmd {
	return func() tea.Msg {
		_, err := m.agentFn(context.Background(), input)
		return agentDoneMsg{err: err}
	}
}

// ── View ────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.quitting {
		return StyleDim.Render("¡Hasta luego!") + "\n"
	}

	var sb strings.Builder
	ban := m.banner()
	sb.WriteString(ban)
	sb.WriteString("\n")

	// History viewport
	viewportH := m.height - strings.Count(ban, "\n") - 6
	if viewportH < 10 {
		viewportH = 20
	}
	end := len(m.history) - m.scroll
	if end < 0 {
		end = 0
	}
	start := end - viewportH
	if start < 0 {
		start = 0
	}
	for _, line := range m.history[start:end] {
		sb.WriteString(line)
		sb.WriteString("\n")
	}

	// Menu overlay
	if m.menuActive {
		sb.WriteString(m.renderMenu())
	}

	// Spinner
	if m.thinking {
		sb.WriteString(fmt.Sprintf("  %s pensando...\n", m.spinner.View()))
	}

	// Input
	sb.WriteString(m.input.View())

	// Autocomplete dropdown
	if m.acActive && !m.menuActive {
		sb.WriteString("\n")
		for i, item := range m.acItems {
			if i == m.acCursor {
				sb.WriteString(StylePrompt.Render("  ❯ /"+item) + "\n")
			} else {
				sb.WriteString(StyleDim.Render("    /"+item) + "\n")
			}
		}
	}

	return sb.String()
}

func (m Model) renderMenu() string {
	var sb strings.Builder
	sb.WriteString("\n" + StyleSubtitle.Render(m.menuTitle) + "\n\n")
	for i, item := range m.menuItems {
		if item.Label == "───────────────────" {
			sb.WriteString("  " + StyleDim.Render(item.Label) + "\n")
			continue
		}
		cursor := "  "
		style := StyleDim
		if i == m.menuCursor {
			cursor = StylePrompt.Render("❯ ")
			style = StyleSubtitle
		}
		line := style.Render(item.Label)
		if item.Hint != "" {
			line += "  " + StyleDim.Render(item.Hint)
		}
		sb.WriteString(cursor + line + "\n")
	}
	sb.WriteString("\n" + StyleDim.Render("  Enter confirmar · Esc cancelar") + "\n")
	return sb.String()
}


