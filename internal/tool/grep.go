package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/cloudcentinel/hoa/internal/api"
)

func init() { Default.Register(&GrepTool{}) }

type GrepTool struct{}

func (GrepTool) Definition() api.ToolDef {
	return api.ToolDef{
		Name:        "grep",
		Description: "Search for a regex pattern in files. Returns matching lines with file path and line number.",
		InputSchema: map[string]any{
			"pattern": map[string]any{"type": "string", "description": "Regex pattern to search for."},
			"path":    map[string]any{"type": "string", "description": "Directory or file to search in (default: current dir)."},
			"include": map[string]any{"type": "string", "description": "File glob filter, e.g. '*.go' (optional)."},
		},
		Required: []string{"pattern"},
	}
}

func (GrepTool) Execute(ctx context.Context, input string) (string, bool) {
	var in struct {
		Pattern string `json:"pattern"`
		Path    string `json:"path"`
		Include string `json:"include"`
	}
	if err := json.Unmarshal([]byte(input), &in); err != nil {
		return fmt.Sprintf("invalid input: %v", err), true
	}
	if in.Path == "" {
		in.Path = "."
	}
	args := []string{"-rn", "--color=never"}
	if in.Include != "" {
		args = append(args, "--include="+in.Include)
	}
	args = append(args, in.Pattern, in.Path)

	out, err := exec.CommandContext(ctx, "grep", args...).CombinedOutput()
	if err != nil {
		if len(out) == 0 {
			return "no matches found", false
		}
		return string(out), false
	}
	return string(out), false
}
