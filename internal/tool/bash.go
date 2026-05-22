package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/cloudcentinel/hoa/internal/api"
)

func init() { Default.Register(&BashTool{}) }

type BashTool struct{}

func (BashTool) Definition() api.ToolDef {
	return api.ToolDef{
		Name:        "bash",
		Description: "Run a shell command and return stdout/stderr. Use for running tests, builds, git, etc.",
		InputSchema: map[string]any{
			"command": map[string]any{"type": "string", "description": "The shell command to execute."},
		},
		Required: []string{"command"},
	}
}

func (BashTool) Execute(ctx context.Context, input string) (string, bool) {
	var in struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal([]byte(input), &in); err != nil {
		return fmt.Sprintf("invalid input: %v", err), true
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "sh", "-c", in.Command).CombinedOutput()
	if err != nil {
		return fmt.Sprintf("%s\n[error: %v]", out, err), true
	}
	return string(out), false
}
