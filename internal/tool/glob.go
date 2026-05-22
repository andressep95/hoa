package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudcentinel/hoa/internal/api"
)

func init() { Default.Register(&GlobTool{}) }

type GlobTool struct{}

func (GlobTool) Definition() api.ToolDef {
	return api.ToolDef{
		Name:        "glob",
		Description: "Find files matching a glob pattern. Returns list of matching paths.",
		InputSchema: map[string]any{
			"pattern": map[string]any{"type": "string", "description": "Glob pattern, e.g. '**/*.go', 'src/**/*.ts'."},
			"path":    map[string]any{"type": "string", "description": "Root directory to search from (default: current dir)."},
		},
		Required: []string{"pattern"},
	}
}

func (GlobTool) Execute(_ context.Context, input string) (string, bool) {
	var in struct {
		Pattern string `json:"pattern"`
		Path    string `json:"path"`
	}
	if err := json.Unmarshal([]byte(input), &in); err != nil {
		return fmt.Sprintf("invalid input: %v", err), true
	}
	root := in.Path
	if root == "" {
		root = "."
	}
	var matches []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			base := filepath.Base(path)
			if base == ".git" || base == "node_modules" || base == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		matched, _ := filepath.Match(in.Pattern, filepath.Base(path))
		if matched {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return err.Error(), true
	}
	if len(matches) == 0 {
		return "no files found", false
	}
	return strings.Join(matches, "\n"), false
}
