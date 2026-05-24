package ui

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"

	"github.com/cloudcentinel/hoa/internal/command"
	"github.com/cloudcentinel/hoa/internal/permission"
)

// AgentSendFunc sends a message to the agent.
type AgentSendFunc func(ctx context.Context, input string) (string, error)

type outputMsg struct{ kind, text string }
type agentDoneMsg struct{ err error }
type asyncCmdDoneMsg struct{ result command.Result }

// ApprovalRequest is sent when the agent needs y/a/n from the user.
type ApprovalRequest struct {
	Prompt string
	Detail string
	Reply  chan permission.ConfirmResult
}

// FooterUpdate is a message sent to refresh footer state in-place.
// Only non-zero / non-nil fields are applied so callers can update
// individual segments (cost-only, oracle-only, etc).
type FooterUpdate struct {
	CWD              *string
	WorkingCount     *int
	InTok, OutTok    *int
	CostUSD          *float64
	OracleConfigured *bool
	OracleOK         *bool
	OracleErr        error // ignored when nil — set ResetOracleErr to clear
	ResetOracleErr   bool
}

// Model is the main Bubble Tea TUI model.
type Model struct {
	input    textinput.Model
	spinner  spinner.Model
	renderer *glamour.TermRenderer
	vp       viewport.Model

	lines    []string
	cmdHist  []string
	histIdx  int
	atBottom bool
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

	// Approval modal state
	approvalActive bool
	approvalPrompt string
	approvalDetail string
	approvalReply  chan permission.ConfirmResult
	approvalVP     viewport.Model

	// Autocomplete state
	acActive bool
	acItems  []string
	acCursor int

	// Footer (always-visible status line)
	footer Footer
}

// NewProgram creates the Bubble Tea program and returns an output function for the agent.
func NewProgram(banner func() string, agentFn AgentSendFunc, cmdCtx *command.Context) (*tea.Program, func(string, string)) {
	ch := make(chan outputMsg, 64)
	m := newModel(banner, agentFn, cmdCtx, ch)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

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

	r, _ := glamour.NewTermRenderer(
		glamour.WithStandardStyle(glamourStyle()),
		glamour.WithWordWrap(100),
	)

	vp := viewport.New(80, 20)
	vp.MouseWheelEnabled = true

	cwd, _ := os.Getwd()

	return Model{
		input:    ti,
		spinner:  sp,
		renderer: r,
		vp:       vp,
		lines:    []string{},
		cmdHist:  []string{},
		histIdx:  -1,
		atBottom: true,
		banner:   banner,
		agentFn:  agentFn,
		cmdCtx:   cmdCtx,
		outputCh: ch,
		footer:   Footer{CWD: cwd, OracleConfigured: cmdCtx != nil && cmdCtx.MemoryEnabled != nil && cmdCtx.MemoryEnabled(), OracleOK: true},
	}
}

func glamourStyle() string {
	if s := os.Getenv("HOA_THEME"); s != "" {
		return s
	}
	return "dark"
}

func newRenderer(width int) *glamour.TermRenderer {
	wrap := width - 4
	if wrap < 40 {
		wrap = 40
	}
	r, _ := glamour.NewTermRenderer(
		glamour.WithStandardStyle(glamourStyle()),
		glamour.WithWordWrap(wrap),
	)
	return r
}

func (m *Model) calcViewportHeight() int {
	if m.height == 0 {
		return 20
	}
	ban := m.banner()
	banH := strings.Count(ban, "\n") + 1
	// Reserve: 1 separator after banner + 1 footer + 1 spinner + 1 input + 1 padding
	h := m.height - banH - 5
	if h < 5 {
		h = 5
	}
	return h
}

func (m *Model) refreshViewport() {
	content := strings.Join(m.lines, "\n")
	m.vp.SetContent(content)

	// Grow viewport with content up to the calculated max height.
	// This keeps the input prompt near the content instead of pushing it
	// to the very bottom of an empty full-height viewport.
	maxH := m.calcViewportHeight()
	h := len(m.lines)
	if h > maxH {
		h = maxH
	}
	m.vp.Height = h

	if m.atBottom && h >= maxH {
		m.vp.GotoBottom()
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
		m.renderer = newRenderer(msg.Width)
		m.vp.Width = msg.Width
		m.vp.Height = m.calcViewportHeight()
		m.refreshViewport()
		return m, nil

	case tea.MouseMsg:
		var cmd tea.Cmd
		m.vp, cmd = m.vp.Update(msg)
		m.atBottom = m.vp.AtBottom()
		return m, cmd

	case tea.KeyMsg:
		if m.menuActive {
			return m.updateMenu(msg)
		}
		if m.approvalActive {
			return m.updateApproval(msg)
		}
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
				if len(m.acItems) > 0 {
					val := "/" + m.acItems[m.acCursor]
					m.input.SetValue("")
					m.acActive = false
					m.histIdx = -1
					m.cmdHist = append(m.cmdHist, val)
					m.appendLine(StylePrompt.Render("❯ ") + val)
					m.appendLine("")
					m.refreshViewport()
					return m.executeCommand(val)
				}
				m.acActive = false
			case "esc":
				m.acActive = false
				return m, nil
			default:
				var cmd tea.Cmd
				m.input, cmd = m.input.Update(msg)
				m.updateAutocompleteState()
				return m, cmd
			}
		}
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
			m.appendLine(StylePrompt.Render("❯ ") + val)
			m.appendLine("")
			m.atBottom = true
			m.refreshViewport()
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

		case "pgup", "ctrl+u":
			m.vp.HalfPageUp()
			m.atBottom = m.vp.AtBottom()
			return m, nil

		case "pgdown", "ctrl+d":
			m.vp.HalfPageDown()
			m.atBottom = m.vp.AtBottom()
			return m, nil
		}

	case agentDoneMsg:
		m.drainOutput()
		m.thinking = false
		if msg.err != nil {
			m.appendLine(StyleError.Render("  error: " + msg.err.Error()))
		}
		m.appendLine("")
		m.atBottom = true
		m.refreshFooterFromCtx()
		m.refreshViewport()
		return m, nil

	case FooterUpdate:
		if msg.CWD != nil {
			m.footer.CWD = *msg.CWD
		}
		if msg.WorkingCount != nil {
			m.footer.WorkingCount = *msg.WorkingCount
		}
		if msg.InTok != nil {
			m.footer.InTok = *msg.InTok
		}
		if msg.OutTok != nil {
			m.footer.OutTok = *msg.OutTok
		}
		if msg.CostUSD != nil {
			m.footer.CostUSD = *msg.CostUSD
		}
		if msg.OracleConfigured != nil {
			m.footer.OracleConfigured = *msg.OracleConfigured
		}
		if msg.OracleOK != nil {
			m.footer.OracleOK = *msg.OracleOK
		}
		if msg.ResetOracleErr {
			m.footer.OracleErr = nil
		} else if msg.OracleErr != nil {
			m.footer.OracleErr = msg.OracleErr
		}
		return m, nil

	case ApprovalRequest:
		m.approvalActive = true
		m.approvalPrompt = msg.Prompt
		m.approvalDetail = msg.Detail
		m.approvalReply = msg.Reply
		ban := m.banner()
		banH := strings.Count(ban, "\n") + 1
		// innerW: border(2) + padding(2) = 4 overhead; additional margin of 4
		innerW := m.width - 8
		if innerW < 40 {
			innerW = 40
		}
		// height: total - banner - contextLines(3) - title(1) - sep(1) - border(2) - hint(2)
		vpH := m.height - banH - 9
		if vpH < 3 {
			vpH = 3
		}
		m.approvalVP = viewport.New(innerW, vpH)
		m.approvalVP.MouseWheelEnabled = true
		if msg.Detail != "" {
			m.approvalVP.SetContent(msg.Detail)
		} else {
			m.approvalVP.SetContent(msg.Prompt)
		}
		return m, nil

	case asyncCmdDoneMsg:
		m.thinking = false
		result := msg.result
		if len(result.Menu) > 0 {
			if len(result.Lines) > 0 {
				m.lines = append(m.lines, result.Lines...)
				m.refreshViewport()
			}
			m.menuActive = true
			m.menuTitle = result.Title
			m.menuItems = result.Menu
			m.menuCursor = 0
			return m, nil
		}
		m.lines = append(m.lines, result.Lines...)
		m.atBottom = true
		m.refreshFooterFromCtx()
		m.refreshViewport()
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.updateAutocompleteState()

	return m, cmd
}

// ── Menu handling ───────────────────────────────────────────────────────────

func (m Model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		for i := m.menuCursor - 1; i >= 0; i-- {
			if m.menuItems[i].Action != nil || m.menuItems[i].AsyncAction != nil {
				m.menuCursor = i
				break
			}
		}
	case "down", "j":
		for i := m.menuCursor + 1; i < len(m.menuItems); i++ {
			if m.menuItems[i].Action != nil || m.menuItems[i].AsyncAction != nil {
				m.menuCursor = i
				break
			}
		}
	case "enter":
		item := m.menuItems[m.menuCursor]
		m.menuActive = false
		if item.AsyncAction != nil {
			m.thinking = true
			fn := item.AsyncAction
			return m, func() tea.Msg {
				return asyncCmdDoneMsg{result: fn()}
			}
		}
		if item.Action != nil {
			result := item.Action()
			if result != "" {
				for _, line := range strings.Split(result, "\n") {
					m.appendLine(StyleDim.Render("  ⎿  " + line))
				}
			} else {
				m.appendLine(StyleDim.Render("  ⎿  " + item.Label))
			}
			m.appendLine("")
			m.atBottom = true
			m.refreshViewport()
		}
	case "esc", "ctrl+c", "q":
		m.menuActive = false
		m.appendLine(StyleDim.Render("  ⎿  cancelled"))
		m.appendLine("")
		m.atBottom = true
		m.refreshViewport()
		return m, nil
	}
	return m, nil
}

// ── Approval handling ────────────────────────────────────────────────────────

func (m Model) updateApproval(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	reply := func(r permission.ConfirmResult, echo string) (tea.Model, tea.Cmd) {
		m.appendLine(StyleDim.Render(echo))
		if m.approvalReply != nil {
			m.approvalReply <- r
			m.approvalReply = nil
		}
		m.approvalActive = false
		m.atBottom = true
		m.refreshViewport()
		return m, nil
	}
	switch msg.String() {
	case "y", "Y":
		return reply(permission.ResultYes, "  > approved (this time)")
	case "a", "A":
		return reply(permission.ResultAlways, "  > approved (always)")
	case "n", "N", "esc", "ctrl+c":
		return reply(permission.ResultNo, "  > denied")
	case "up", "pgup", "ctrl+u":
		m.approvalVP.HalfPageUp()
		return m, nil
	case "down", "pgdown", "ctrl+d":
		m.approvalVP.HalfPageDown()
		return m, nil
	}
	return m, nil
}

// ── Autocomplete handling ───────────────────────────────────────────────────

func (m *Model) updateAutocompleteState() {
	val := m.input.Value()
	if strings.HasPrefix(val, "/") && !strings.Contains(val, " ") {
		prefix := strings.ToLower(val[1:])
		names := command.Names()
		sort.Strings(names)

		var filtered []string
		if prefix == "" {
			filtered = names
		} else {
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

// ── Content helpers ──────────────────────────────────────────────────────────

func (m *Model) refreshFooterFromCtx() {
	if m.cmdCtx == nil {
		return
	}
	if m.cmdCtx.TokensUsed != nil {
		in, out := m.cmdCtx.TokensUsed()
		m.footer.InTok = in
		m.footer.OutTok = out
	}
	if m.cmdCtx.CostTotal != nil {
		m.footer.CostUSD = m.cmdCtx.CostTotal()
	}
	if m.cmdCtx.WorkingCount != nil {
		m.footer.WorkingCount = m.cmdCtx.WorkingCount()
	}
}

func (m *Model) appendLine(line string) {
	m.lines = append(m.lines, line)
}

// ── Command dispatch ─────────────────────────────────────────────────────────

func (m Model) executeCommand(val string) (tea.Model, tea.Cmd) {
	if result, handled := command.Dispatch(m.cmdCtx, val); handled {
		if result.Quit {
			m.quitting = true
			return m, tea.Quit
		}
		if result.ClearScreen {
			m.lines = m.lines[:0]
			m.vp.SetContent("")
			return m, tea.ClearScreen
		}
		if result.AsyncFn != nil {
			if len(result.Lines) > 0 {
				m.lines = append(m.lines, result.Lines...)
				m.refreshViewport()
			}
			m.thinking = true
			fn := result.AsyncFn
			return m, func() tea.Msg {
				return asyncCmdDoneMsg{result: fn()}
			}
		}
		if len(result.Menu) > 0 {
			if len(result.Lines) > 0 {
				m.lines = append(m.lines, result.Lines...)
				m.refreshViewport()
			}
			m.menuActive = true
			m.menuTitle = result.Title
			m.menuItems = result.Menu
			m.menuCursor = 0
			return m, nil
		}
		m.lines = append(m.lines, result.Lines...)
		m.appendLine("")
		m.atBottom = true
		m.refreshViewport()
		return m, nil
	}
	m.thinking = true
	return m, m.runAgent(val)
}

func (m *Model) drainOutput() {
	for {
		select {
		case o := <-m.outputCh:
			switch o.kind {
			case "tool":
				m.appendLine(StyleTool.Render("  [tool] " + o.text))
			case "memory-item":
				m.appendLine(StyleDim.Render("  ⎿  " + o.text))
			case "text":
				m.appendMarkdown(o.text)
			default:
				for _, line := range strings.Split(o.text, "\n") {
					m.appendLine("  " + line)
				}
			}
			m.refreshViewport()
		default:
			return
		}
	}
}

func (m *Model) appendMarkdown(text string) {
	rendered := text
	if m.renderer != nil {
		if out, err := m.renderer.Render(text); err == nil {
			rendered = out
		}
	}
	rendered = strings.TrimRight(rendered, "\n")
	for _, line := range strings.Split(rendered, "\n") {
		m.appendLine(line)
	}
	m.appendLine("")
}

func (m Model) runAgent(input string) tea.Cmd {
	return func() tea.Msg {
		_, err := m.agentFn(context.Background(), input)
		return agentDoneMsg{err: err}
	}
}

// ── View ─────────────────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.quitting {
		return StyleDim.Render("¡Hasta luego!") + "\n"
	}

	if m.approvalActive {
		return m.viewApprovalModal()
	}

	var sb strings.Builder
	ban := m.banner()
	sb.WriteString(ban)
	sb.WriteString("\n")

	if m.vp.Height > 0 {
		sb.WriteString(m.vp.View())
		sb.WriteString("\n")
	}

	if m.menuActive {
		sb.WriteString(m.renderMenu())
	}

	if m.thinking {
		sb.WriteString(fmt.Sprintf("  %s pensando...\n", m.spinner.View()))
	}

	// Footer: always-visible status line above the prompt.
	sb.WriteString(m.footer.Render(m.width))
	sb.WriteString("\n")

	sb.WriteString(m.input.View())

	if m.acActive && !m.menuActive {
		sb.WriteString("\n")
		for i, item := range m.acItems {
			if i == m.acCursor {
				sb.WriteString(StylePrompt.Render("  ❯ /" + item))
				sb.WriteString("\n")
			} else {
				sb.WriteString(StyleDim.Render("    /" + item))
				sb.WriteString("\n")
			}
		}
	}

	return sb.String()
}

func (m Model) renderMenu() string {
	var body strings.Builder

	// Title as first line inside the border.
	body.WriteString(StyleSubtitle.Render(m.menuTitle))
	body.WriteString("\n\n")

	for i, item := range m.menuItems {
		if item.Label == "───────────────────" {
			body.WriteString(StyleDim.Render(item.Label))
			body.WriteString("\n")
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
		body.WriteString(cursor)
		body.WriteString(line)
		body.WriteString("\n")
	}

	body.WriteString("\n")
	body.WriteString(StyleDim.Render("Enter confirmar · Esc cancelar"))

	outerW := m.width - 4
	if outerW < 30 {
		outerW = 30
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("75")).
		Padding(0, 1).
		Width(outerW)

	return "\n" + box.Render(body.String()) + "\n"
}

func toolIcon(prompt string) string {
	lp := strings.ToLower(prompt)
	switch {
	case strings.Contains(lp, "bash"):
		return "⚡"
	case strings.Contains(lp, "write"):
		return "✏ "
	case strings.Contains(lp, "read"):
		return "📄"
	case strings.Contains(lp, "memory"):
		return "🧠"
	default:
		return "🔑"
	}
}

func scrollIndicator(vp viewport.Model) string {
	if vp.TotalLineCount() <= vp.Height {
		return ""
	}
	pct := int(vp.ScrollPercent() * 100)
	return StyleDim.Render(fmt.Sprintf(" %d%% ↕", pct))
}

func (m Model) viewApprovalModal() string {
	var sb strings.Builder

	// Banner stays visible for context.
	ban := m.banner()
	sb.WriteString(ban)
	sb.WriteString("\n")

	// Show a small slice of the conversation so the user remembers what triggered this.
	if m.vp.Height > 0 {
		contextLines := 3
		savedH := m.vp.Height
		if savedH > contextLines {
			m.vp.Height = contextLines
		}
		sb.WriteString(m.vp.View())
		m.vp.Height = savedH
		sb.WriteString("\n")
	}

	// Modal dimensions: border(2) + padding(2) = 4 overhead; margin of 4.
	outerW := m.width - 4
	if outerW < 44 {
		outerW = 44
	}
	innerW := outerW - 4 // padding(1 each side) + border(1 each side)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("220")).
		Padding(0, 1).
		Width(outerW)

	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Bold(true)
	icon := toolIcon(m.approvalPrompt)
	title := titleStyle.Render(icon + " " + m.approvalPrompt)

	sep := StyleBorder.Render(strings.Repeat("─", innerW))

	// Scroll indicator aligned to the right of the detail viewport.
	scrollHint := scrollIndicator(m.approvalVP)
	detailView := m.approvalVP.View()
	if scrollHint != "" {
		// Place scroll hint on its own line after the viewport.
		detailView = detailView + "\n" + lipgloss.NewStyle().
			Width(innerW).Align(lipgloss.Right).Render(scrollHint)
	}

	body := title + "\n" + sep + "\n" + detailView
	modal := borderStyle.Render(body)

	sb.WriteString("\n")
	sb.WriteString(modal)
	sb.WriteString("\n")
	sb.WriteString(StyleDim.Render("  y aprobar · a siempre · n denegar · esc cancelar"))
	sb.WriteString("\n")

	return sb.String()
}
