package memory

import (
	"fmt"
	"os"
	"strings"
)

const maxFilePreview = 2000

// WorkingContext builds context from uncommitted changes (git diff).
// This is the "session memory" — what's being built right now.
// Clears naturally after commit (git diff returns empty).
func WorkingContext() string {
	modified := splitNonEmpty(gitCmd("diff", "--name-only"))
	staged := splitNonEmpty(gitCmd("diff", "--cached", "--name-only"))
	untracked := splitNonEmpty(gitCmd("ls-files", "--others", "--exclude-standard"))

	seen := make(map[string]bool)
	var allFiles []string
	for _, list := range [][]string{staged, modified, untracked} {
		for _, f := range list {
			if !seen[f] && !shouldIgnore(f) {
				allFiles = append(allFiles, f)
				seen[f] = true
			}
		}
	}

	if len(allFiles) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("<working_changes>\n")
	sb.WriteString(fmt.Sprintf("%d archivos en progreso:\n\n", len(allFiles)))

	for _, f := range allFiles {
		// Try diff against HEAD first
		content := gitCmd("diff", "HEAD", "--", f)
		if content == "" {
			// Untracked — read file directly
			data, err := os.ReadFile(f)
			if err == nil {
				content = string(data)
			}
		}
		if len(content) > maxFilePreview {
			content = content[:maxFilePreview] + "\n..."
		}
		if content != "" {
			sb.WriteString(fmt.Sprintf("--- %s ---\n%s\n\n", f, content))
		}
	}
	sb.WriteString("</working_changes>")
	return sb.String()
}
