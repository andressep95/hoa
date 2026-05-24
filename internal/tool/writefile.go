package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudcentinel/hoa/internal/api"
)

func init() { Default.Register(&WriteFileTool{}) }

type WriteFileTool struct{}

func (WriteFileTool) Definition() api.ToolDef {
	return api.ToolDef{
		Name:        "write_file",
		Description: "Write content to a file. Creates parent directories if needed. Overwrites existing content.",
		InputSchema: map[string]any{
			"path":    map[string]any{"type": "string", "description": "File path to write."},
			"content": map[string]any{"type": "string", "description": "Content to write."},
		},
		Required: []string{"path", "content"},
	}
}

func (WriteFileTool) Execute(_ context.Context, input string) (string, bool) {
	var in struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(input), &in); err != nil {
		return fmt.Sprintf("invalid input: %v", err), true
	}
	if err := os.MkdirAll(filepath.Dir(in.Path), 0o755); err != nil {
		return err.Error(), true
	}
	if err := os.WriteFile(in.Path, []byte(in.Content), 0o644); err != nil {
		return err.Error(), true
	}
	return fmt.Sprintf("wrote %d bytes to %s", len(in.Content), in.Path), false
}
