package command

import (
	"os/exec"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	greenStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	redStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
)

func init() {
	Register("commit", cmdCommit)
}

// cmdCommit handles /commit and /commit <message>.
// Without args: shows status and instructions.
// With args: validates message and executes git commit if valid.
func cmdCommit(_ *Context, args string) Result {
	// Step 1: git status
	status := gitRun("status", "--short")
	if strings.TrimSpace(status) == "" {
		return Result{Lines: []string{"  No hay cambios para commitear."}}
	}

	// If no message provided, show analysis and wait for message
	if args == "" {
		return commitAnalysis(status)
	}

	// Message provided — validate before committing
	return commitExecute(args, status)
}

func commitAnalysis(status string) Result {
	sensitive := checkSensitiveFiles(status)
	diff := gitRun("diff", "--stat", "HEAD")
	log := gitRun("log", "-3", "--oneline")

	lines := []string{"  ── /commit ──", ""}
	lines = append(lines, "  Archivos modificados:")
	for _, l := range strings.Split(status, "\n") {
		if l != "" {
			lines = append(lines, "    "+l)
		}
	}

	if len(sensitive) > 0 {
		lines = append(lines, "", "  ⚠️  Archivos sensibles detectados (NO se commitearán):")
		for _, s := range sensitive {
			lines = append(lines, "    "+s)
		}
	}

	if diff != "" {
		lines = append(lines, "", "  Diff stat:")
		for _, l := range strings.Split(diff, "\n") {
			if l != "" {
				lines = append(lines, "    "+colorizeDiffStat(l))
			}
		}
	}

	if log != "" {
		lines = append(lines, "", "  Últimos commits:")
		for _, l := range strings.Split(log, "\n") {
			if l != "" {
				lines = append(lines, "    "+l)
			}
		}
	}

	lines = append(lines, "")
	lines = append(lines, "  Usa: /commit type(scope): description\\nwhat: ...\\nwhy: ...\\nbreaking: false")
	lines = append(lines, "  O pide al agente: \"haz commit de estos cambios\"")

	return Result{Lines: lines}
}

func commitExecute(msg, status string) Result {
	// Normalize escaped newlines from single-line input
	msg = strings.ReplaceAll(msg, "\\n", "\n")

	// Pre-commit validation — BLOCKS if format is wrong
	if errs := ValidateCommitMsg(msg); len(errs) > 0 {
		lines := []string{"  ❌ Commit BLOQUEADO — no cumple Conventional Commits:", ""}
		for _, e := range errs {
			lines = append(lines, "    • "+e)
		}
		lines = append(lines, "", "  Formato requerido:")
		lines = append(lines, "    type(scope): description")
		lines = append(lines, "    ")
		lines = append(lines, "    what: qué hace el código ahora")
		lines = append(lines, "    why: por qué fue necesario")
		lines = append(lines, "    breaking: false")
		return Result{Lines: lines}
	}

	// Sensitive file guard — exclude them
	sensitive := checkSensitiveFiles(status)
	if len(sensitive) > 0 {
		// Stage all except sensitive
		gitRun("add", "-A")
		for _, s := range sensitive {
			parts := strings.Fields(s)
			if len(parts) >= 2 {
				gitRun("reset", "HEAD", "--", parts[1])
			}
		}
	} else {
		gitRun("add", "-A")
	}

	// Execute git commit
	out, err := exec.Command("git", "commit", "-m", msg).CombinedOutput()
	if err != nil {
		return Result{Lines: []string{
			"  ❌ git commit falló:",
			"  " + strings.TrimSpace(string(out)),
		}}
	}

	lines := []string{"  ✅ Commit exitoso:", ""}
	for _, l := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		lines = append(lines, "    "+l)
	}

	// TODO: Post-commit memory push to Oracle
	// When memory is connected:
	// 1. Extract changes (file-level granularity)
	// 2. INSERT into MEMORY_CHANGES + MEMORY_CHANGE_HUNKS
	// 3. Queue embedding generation via ENRICHMENT_QUEUE

	return Result{Lines: lines}
}

func gitRun(args ...string) string {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func colorizeDiffStat(line string) string {
	// Colorize the +/- portion at the end of diff stat lines
	idx := strings.LastIndex(line, "|")
	if idx == -1 {
		return line
	}
	prefix := line[:idx+1]
	stats := line[idx+1:]
	var result strings.Builder
	result.WriteString(prefix)
	for _, ch := range stats {
		switch ch {
		case '+':
			result.WriteString(greenStyle.Render("+"))
		case '-':
			result.WriteString(redStyle.Render("-"))
		default:
			result.WriteRune(ch)
		}
	}
	return result.String()
}

func checkSensitiveFiles(status string) []string {
	patterns := []string{".env", ".key", ".pem", "secret", "credential"}
	var found []string
	for _, line := range strings.Split(status, "\n") {
		lower := strings.ToLower(line)
		for _, p := range patterns {
			if strings.Contains(lower, p) {
				found = append(found, strings.TrimSpace(line))
				break
			}
		}
	}
	return found
}


