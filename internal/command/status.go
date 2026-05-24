package command

import (
	"fmt"
	"strings"
	"time"

	"github.com/cloudcentinel/hoa/internal/cost"
)

func init() {
	Register("status", cmdStatus)
}

func cmdStatus(ctx *Context, _ string) Result {
	var lines []string
	lines = append(lines, "")

	// ── Provider section
	lines = append(lines, "  Provider")
	if ctx.GetProvider != nil {
		lines = append(lines, "    active:    "+ctx.GetProvider())
	}
	if ctx.GetModel != nil {
		lines = append(lines, "    model:     "+ctx.GetModel())
	}
	if ctx.GetPlanModel != nil {
		lines = append(lines, "    planning:  "+ctx.GetPlanModel())
	}
	if ctx.GetMode != nil {
		lines = append(lines, "    mode:      "+ctx.GetMode())
	}
	lines = append(lines, "")

	// ── Memory section
	lines = append(lines, "  Memory (Oracle 23ai)")
	if ctx.MemoryEnabled != nil && ctx.MemoryEnabled() {
		dsn := ""
		if ctx.MemoryDSN != nil {
			dsn = sanitizeDSN(ctx.MemoryDSN())
		}
		lines = append(lines, "    enabled:   yes")
		lines = append(lines, "    dsn:       "+dsn)

		if ctx.OracleStatus != nil {
			ok, oerr, since := ctx.OracleStatus()
			switch {
			case since.IsZero():
				lines = append(lines, "    health:    (no check yet)")
			case ok:
				lines = append(lines, fmt.Sprintf("    health:    ok  (last check %s ago)", humanAge(since)))
			default:
				msg := "err"
				if oerr != nil {
					msg = oerr.Error()
				}
				lines = append(lines, fmt.Sprintf("    health:    err: %s", msg))
				lines = append(lines, fmt.Sprintf("    last check: %s ago", humanAge(since)))
			}
		}
	} else {
		lines = append(lines, "    enabled:   no")
	}
	lines = append(lines, "")

	// ── Working changes
	lines = append(lines, "  Working changes")
	if ctx.WorkingSnapshot != nil {
		ws := ctx.WorkingSnapshot()
		if len(ws.Files) == 0 {
			lines = append(lines, "    (no files)")
		} else {
			lines = append(lines, fmt.Sprintf("    %d archivo(s):", len(ws.Files)))
			for _, f := range ws.Files {
				lines = append(lines, fmt.Sprintf("      %s  %s", f.Path, humanBytes(f.SizeBytes)))
			}
		}
	} else {
		lines = append(lines, "    (snapshot unavailable)")
	}
	lines = append(lines, "")

	// ── Session usage
	lines = append(lines, "  Session")
	if ctx.TokensUsed != nil {
		in, out := ctx.TokensUsed()
		lines = append(lines, fmt.Sprintf("    tokens:    %d in · %d out · %d total", in, out, in+out))
	}
	if ctx.CostTotal != nil {
		lines = append(lines, "    cost:      "+cost.FormatCost(ctx.CostTotal()))
	}
	lines = append(lines, "")

	// ── Permissions remembered
	if ctx.RememberedTools != nil {
		remembered := ctx.RememberedTools()
		lines = append(lines, "  Permissions")
		if len(remembered) == 0 {
			lines = append(lines, "    remembered: (none yet)")
		} else {
			lines = append(lines, "    remembered: "+strings.Join(remembered, ", "))
		}
		lines = append(lines, "")
	}

	return Result{Lines: lines}
}

// sanitizeDSN strips passwords from oracle:// connection strings.
//
//	oracle://user:pass@host:1521/SID -> oracle://user:***@host:1521/SID
func sanitizeDSN(dsn string) string {
	idx := strings.Index(dsn, "://")
	if idx < 0 {
		return dsn
	}
	scheme := dsn[:idx+3]
	rest := dsn[idx+3:]
	at := strings.Index(rest, "@")
	if at < 0 {
		return dsn
	}
	creds := rest[:at]
	host := rest[at:]
	colon := strings.Index(creds, ":")
	if colon < 0 {
		return dsn
	}
	return scheme + creds[:colon] + ":***" + host
}

// humanAge returns a compact "Ns/Nm/Nh ago" given a past time.
func humanAge(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	default:
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
}

// humanBytes is a local copy to avoid pulling in memory pkg here. Mirrors
// memory.HumanBytes; kept inline to reduce import surface.
func humanBytes(n int) string {
	if n < 1024 {
		return fmt.Sprintf("%dB", n)
	}
	const kb = 1024.0
	const mb = kb * 1024
	const gb = mb * 1024
	f := float64(n)
	switch {
	case f < mb:
		return fmt.Sprintf("%.1fKB", f/kb)
	case f < gb:
		return fmt.Sprintf("%.1fMB", f/mb)
	default:
		return fmt.Sprintf("%.1fGB", f/gb)
	}
}
