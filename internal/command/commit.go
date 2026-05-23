package command

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	greenStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	redStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	resultStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).SetString("  ⎿  ")
)

func init() {
	Register("commit", cmdCommit)
}

const commitSystemPrompt = `Analyze the git diff and generate commit message(s).

RESPOND ONLY WITH VALID JSON. No markdown, no explanation.

Schema:
{
  "commits": [
    {
      "type": "feat|fix|refactor|perf|sec|test|docs|chore|ci|style",
      "scope": "module name from top-level dir",
      "description": "imperative, < 50 chars, no period",
      "what": "one sentence, what the code does now (< 72 chars)",
      "why": "one sentence, why it was necessary (< 72 chars)",
      "breaking": false,
      "files": ["path/to/file1.go", "path/to/file2.go"]
    }
  ]
}

Rules:
- Scope from top-level directory or module (agent, ui, command, config, etc.)
- what/why MUST be specific. Bad: "improve things". Good: "Adds /mode command to switch execution modes"
- If changes are CLEARLY unrelated, split into multiple commits
- If changes are related, use ONE commit
- files: list the files that belong to each commit
- ALL strings < 72 chars`

// CommitProposal is what the LLM returns.
type CommitProposal struct {
	Type        string   `json:"type"`
	Scope       string   `json:"scope"`
	Description string   `json:"description"`
	What        string   `json:"what"`
	Why         string   `json:"why"`
	Breaking    bool     `json:"breaking"`
	Files       []string `json:"files"`
}

type commitResponse struct {
	Commits []CommitProposal `json:"commits"`
}

func (c CommitProposal) Message() string {
	header := fmt.Sprintf("%s(%s): %s", c.Type, c.Scope, c.Description)
	return fmt.Sprintf("%s\n\nwhat: %s\nwhy: %s\nbreaking: %v", header, c.What, c.Why, c.Breaking)
}

func cmdCommit(ctx *Context, _ string) Result {
	status := gitRun("status", "--short")
	if strings.TrimSpace(status) == "" {
		return Result{Lines: []string{resultStyle.Render("No hay cambios para commitear.")}}
	}

	return Result{
		Lines:   []string{resultStyle.Render("Analizando cambios...")},
		AsyncFn: func() Result { return generateCommit(ctx, status) },
	}
}

func generateCommit(ctx *Context, status string) Result {
	sensitive := checkSensitiveFiles(status)

	diff := gitRun("diff", "HEAD")
	stat := gitRun("diff", "--stat", "HEAD")
	log := gitRun("log", "-3", "--oneline")

	// Collect new/untracked file previews
	var newFiles strings.Builder
	for _, line := range strings.Split(status, "\n") {
		if strings.HasPrefix(line, "??") || strings.HasPrefix(line, "A ") {
			file := strings.TrimSpace(line[2:])
			raw, _ := exec.Command("head", "-30", file).Output()
			content := strings.TrimSpace(string(raw))
			if len(content) > 1500 {
				content = content[:1500] + "\n..."
			}
			if content != "" {
				newFiles.WriteString("--- " + file + " ---\n" + content + "\n\n")
			}
		}
	}

	if ctx.AgentSend == nil {
		return Result{Lines: []string{resultStyle.Render("AgentSend no disponible.")}}
	}

	// Build full prompt
	var prompt strings.Builder
	prompt.WriteString(commitSystemPrompt)
	prompt.WriteString("\n\nGIT STATUS:\n" + status)
	prompt.WriteString("\n\nDIFF STAT:\n" + stat)
	prompt.WriteString("\n\nRECENT COMMITS (match style):\n" + log)
	if newFiles.Len() > 0 {
		prompt.WriteString("\n\nNEW FILES (preview):\n" + newFiles.String())
	}
	prompt.WriteString("\n\nDIFF:\n")
	if len(diff) > 6000 {
		prompt.WriteString(diff[:6000] + "\n... (truncated)")
	} else {
		prompt.WriteString(diff)
	}

	response, err := ctx.AgentSend(prompt.String())
	if err != nil {
		return Result{Lines: []string{resultStyle.Render("Error: " + err.Error())}}
	}

	response = stripCodeFences(response)
	var parsed commitResponse
	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		return Result{Lines: []string{
			resultStyle.Render("LLM no devolvió JSON válido:"),
			"  " + strings.ReplaceAll(response, "\n", "\n  "),
		}}
	}

	if len(parsed.Commits) == 0 {
		return Result{Lines: []string{resultStyle.Render("No se generaron commits.")}}
	}

	return presentCommits(parsed.Commits, sensitive, ctx)
}
func presentCommits(commits []CommitProposal, sensitive []string, ctx *Context) Result {
	lines := []string{""}

	for i, c := range commits {
		if len(commits) > 1 {
			lines = append(lines, fmt.Sprintf("  %s %s(%s): %s",
				greenStyle.Render(fmt.Sprintf("[%d/%d]", i+1, len(commits))),
				c.Type, c.Scope, c.Description))
		} else {
			lines = append(lines, fmt.Sprintf("  %s(%s): %s", c.Type, c.Scope, c.Description))
		}
		lines = append(lines,
			fmt.Sprintf("    what: %s", c.What),
			fmt.Sprintf("    why:  %s", c.Why),
			fmt.Sprintf("    breaking: %v", c.Breaking),
		)
		if len(c.Files) > 0 {
			lines = append(lines, "    "+resultStyle.Render(strings.Join(c.Files, ", ")))
		}
		lines = append(lines, "")
	}

	// Build menu options
	items := []MenuItem{}

	if len(commits) == 1 {
		msg := commits[0].Message()
		items = append(items, MenuItem{
			Label:  "✓ Confirmar commit",
			Action: func() { executeCommit(msg, commits[0].Files, sensitive) },
		})
	} else {
		items = append(items, MenuItem{
			Label: fmt.Sprintf("✓ Commitear %d commits separados", len(commits)),
			Action: func() {
				for _, c := range commits {
					executeCommit(c.Message(), c.Files, sensitive)
				}
			},
		})
		items = append(items, MenuItem{
			Label: "⊕ Unificar en 1 solo commit",
			Action: func() {
				executeCommit(commits[0].Message(), nil, sensitive)
			},
		})
	}

	items = append(items, MenuItem{
		Label:  "✎ Dar feedback (regenerar)",
		Action: nil, // TODO: prompt user for feedback text, re-run with guidance
	})
	items = append(items, MenuItem{
		Label:  "✗ Cancelar",
		Action: func() {},
	})

	return Result{
		Lines: lines,
		Title: resultStyle.Render(fmt.Sprintf("%d commit(s) propuesto(s)", len(commits))),
		Menu:  items,
	}
}

func executeCommit(msg string, files []string, sensitive []string) {
	if errs := ValidateCommitMsg(msg); len(errs) > 0 {
		return
	}

	if len(files) > 0 {
		for _, f := range files {
			exec.Command("git", "add", "--", f).Run()
		}
	} else {
		exec.Command("git", "add", "-A").Run()
	}

	for _, s := range sensitive {
		parts := strings.Fields(s)
		if len(parts) >= 2 {
			exec.Command("git", "reset", "HEAD", "--", parts[len(parts)-1]).Run()
		}
	}

	exec.Command("git", "commit", "-m", msg).Run()
}

func stripCodeFences(s string) string {
	var lines []string
	for _, l := range strings.Split(s, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			continue
		}
		lines = append(lines, l)
	}
	return strings.Join(lines, "\n")
}

func gitRun(args ...string) string {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func colorizeDiffStat(line string) string {
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


