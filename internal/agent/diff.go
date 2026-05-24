package agent

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pmezard/go-difflib/difflib"
)

// buildWriteDiff returns a unified-diff string describing what a write_file
// tool call would change on disk. Returns "" on malformed input.
func buildWriteDiff(rawInput string) string {
	var in struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(rawInput), &in); err != nil {
		return ""
	}

	existing, err := os.ReadFile(in.Path)
	if err != nil {
		return synthesizeNewFileDiff(in.Path, in.Content)
	}

	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(existing)),
		B:        difflib.SplitLines(in.Content),
		FromFile: in.Path + " (current)",
		ToFile:   in.Path + " (proposed)",
		Context:  3,
	}
	text, _ := difflib.GetUnifiedDiffString(diff)
	if text == "" {
		return "(no changes)\n"
	}
	return text
}

func synthesizeNewFileDiff(path, content string) string {
	lines := difflib.SplitLines(content)
	out := "--- /dev/null\n"
	out += "+++ " + path + " (new file)\n"
	out += fmt.Sprintf("@@ -0,0 +1,%d @@\n", len(lines))
	for _, l := range lines {
		out += "+" + l
	}
	if len(lines) > 0 && !endsWithNewline(lines[len(lines)-1]) {
		out += "\n"
	}
	return out
}

func endsWithNewline(s string) bool {
	return len(s) > 0 && s[len(s)-1] == '\n'
}

// buildApprovalPrompt returns (title, detail) for the permission modal.
// title is always the tool name + primary arg so the user knows what's being called.
// detail is the scrollable body: a diff for write_file, the full relevant arg for others.
func buildApprovalPrompt(name, rawInput string) (title, detail string) {
	var args map[string]json.RawMessage
	json.Unmarshal([]byte(rawInput), &args)

	strArg := func(key string) string {
		if v, ok := args[key]; ok {
			var s string
			if json.Unmarshal(v, &s) == nil {
				return s
			}
		}
		return ""
	}

	truncate := func(s string, n int) string {
		if len(s) <= n {
			return s
		}
		return s[:n-1] + "…"
	}

	switch name {
	case "write_file", "edit_file":
		path := strArg("path")
		diff := buildWriteDiff(rawInput)
		if path != "" {
			title = name + ": " + path
		} else {
			title = name
		}
		detail = diff

	case "bash":
		cmd := strArg("command")
		title = "bash"
		detail = cmd

	case "read_file":
		path := strArg("path")
		if path != "" {
			title = "read_file: " + path
		} else {
			title = "read_file"
		}

	case "search_memory":
		query := strArg("query")
		title = "search_memory: " + truncate(query, 60)
		detail = query

	case "grep":
		pattern := strArg("pattern")
		path := strArg("path")
		title = "grep: " + truncate(pattern, 40)
		if path != "" {
			detail = "pattern: " + pattern + "\npath:    " + path
		} else {
			detail = "pattern: " + pattern
		}

	case "glob":
		pattern := strArg("pattern")
		title = "glob: " + truncate(pattern, 60)

	default:
		title = name
		detail = rawInput
	}

	if title == "" {
		title = name
	}
	return title, detail
}
