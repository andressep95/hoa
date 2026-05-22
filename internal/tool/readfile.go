package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/cloudcentinel/hoa/internal/api"
)

func init() { Default.Register(&ReadFileTool{}) }

type ReadFileTool struct{}

func (ReadFileTool) Definition() api.ToolDef {
	return api.ToolDef{
		Name:        "read_file",
		Description: "Read the contents of a file. Optionally specify offset and limit (line numbers, 0-indexed).",
		InputSchema: map[string]any{
			"path":   map[string]any{"type": "string", "description": "File path to read."},
			"offset": map[string]any{"type": "integer", "description": "Start line (0-indexed, optional)."},
			"limit":  map[string]any{"type": "integer", "description": "Max lines to return (optional)."},
		},
		Required: []string{"path"},
	}
}

func (ReadFileTool) Execute(_ context.Context, input string) (string, bool) {
	var in struct {
		Path   string `json:"path"`
		Offset int    `json:"offset"`
		Limit  int    `json:"limit"`
	}
	if err := json.Unmarshal([]byte(input), &in); err != nil {
		return fmt.Sprintf("invalid input: %v", err), true
	}
	data, err := os.ReadFile(in.Path)
	if err != nil {
		return err.Error(), true
	}
	if in.Offset == 0 && in.Limit == 0 {
		return string(data), false
	}
	lines := strings.Split(string(data), "\n")
	start := in.Offset
	if start > len(lines) {
		start = len(lines)
	}
	end := len(lines)
	if in.Limit > 0 && start+in.Limit < end {
		end = start + in.Limit
	}
	return strings.Join(lines[start:end], "\n"), false
}
