package memory

import (
	"fmt"
	"os"
	"strings"
)

const maxFilePreview = 2000

// FileEntry describes one file present in the working tree changes.
type FileEntry struct {
	Path      string
	SizeBytes int
}

// WorkingChanges holds the structured view of uncommitted changes plus the
// pre-rendered context block for LLM injection.
type WorkingChanges struct {
	Files []FileEntry
	Block string
}

// WorkingContext builds context from uncommitted changes (git diff).
// This is the "session memory" — what's being built right now.
// Clears naturally after commit (git diff returns empty).
func WorkingContext() WorkingChanges {
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
		return WorkingChanges{}
	}

	var sb strings.Builder
	sb.WriteString("<working_changes>\n")
	sb.WriteString(fmt.Sprintf("%d archivos en progreso:\n\n", len(allFiles)))

	files := make([]FileEntry, 0, len(allFiles))
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
		// Determine displayed size: prefer original file size on disk; fall
		// back to the diff size for tracked-only edits where the file is
		// not directly readable.
		size := 0
		if info, err := os.Stat(f); err == nil {
			size = int(info.Size())
		} else {
			size = len(content)
		}
		files = append(files, FileEntry{Path: f, SizeBytes: size})

		if len(content) > maxFilePreview {
			content = content[:maxFilePreview] + "\n..."
		}
		if content != "" {
			sb.WriteString(fmt.Sprintf("--- %s ---\n%s\n\n", f, content))
		}
	}
	sb.WriteString("</working_changes>")
	return WorkingChanges{Files: files, Block: sb.String()}
}

// BuildBlockForFiles constructs a <working_changes> context block for a
// subset of files. Called by agent.Send after the user approves each file.
func BuildBlockForFiles(files []FileEntry) string {
	if len(files) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("<working_changes>\n")
	sb.WriteString(fmt.Sprintf("%d archivos en progreso:\n\n", len(files)))
	for _, fe := range files {
		content := gitCmd("diff", "HEAD", "--", fe.Path)
		if content == "" {
			data, err := os.ReadFile(fe.Path)
			if err == nil {
				content = string(data)
			}
		}
		if len(content) > maxFilePreview {
			content = content[:maxFilePreview] + "\n..."
		}
		if content != "" {
			sb.WriteString(fmt.Sprintf("--- %s ---\n%s\n\n", fe.Path, content))
		}
	}
	sb.WriteString("</working_changes>")
	return sb.String()
}

// HumanBytes formats a byte count as a compact human-readable string.
// Examples: 0 → "0B", 999 → "999B", 1024 → "1.0KB", 1500 → "1.5KB",
// 1_500_000 → "1.4MB".
func HumanBytes(n int) string {
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
