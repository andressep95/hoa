package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Footer renders a single status line displayed between the conversation
// viewport and the input prompt. It deliberately does NOT duplicate fields
// shown in the banner (provider, model, mode); only live session state.
type Footer struct {
	CWD              string
	WorkingCount     int
	InTok, OutTok    int
	CostUSD          float64
	OracleConfigured bool
	OracleOK         bool
	OracleErr        error
}

var (
	footerOK   = lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
	footerErr  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	footerSep  = " · "
)

// Render returns the footer line, fitted to width. Segments are joined with " · ".
// If width is too small, segments are truncated from the right.
func (f Footer) Render(width int) string {
	var segs []string

	if f.OracleConfigured {
		if f.OracleOK {
			segs = append(segs, footerOK.Render("Oracle ok"))
		} else {
			segs = append(segs, footerErr.Render("Oracle err"))
		}
	}

	cwd := f.CWD
	if cwd == "" {
		cwd = "?"
	}
	segs = append(segs, "cwd: "+filepath.Base(cwd))

	if f.WorkingCount > 0 {
		segs = append(segs, fmt.Sprintf("%d wc", f.WorkingCount))
	}

	segs = append(segs, fmt.Sprintf("%s tok", humanTokens(f.InTok+f.OutTok)))

	if f.CostUSD > 0 || f.InTok > 0 || f.OutTok > 0 {
		segs = append(segs, fmt.Sprintf("$%.4f", f.CostUSD))
	}

	line := "  " + strings.Join(segs, footerSep)

	// Truncate if needed (very narrow terminals)
	if width > 0 && lipgloss.Width(line) > width {
		// Strip from the right: drop segments until it fits
		for len(segs) > 1 && lipgloss.Width("  "+strings.Join(segs, footerSep)) > width {
			segs = segs[:len(segs)-1]
		}
		line = "  " + strings.Join(segs, footerSep)
	}

	return StyleDim.Render(line)
}

// humanTokens formats a token count compactly (e.g. 1234 → "1.2K", 12 → "12").
func humanTokens(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1_000_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
}
