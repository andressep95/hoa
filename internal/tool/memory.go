package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudcentinel/hoa/internal/api"
	"github.com/cloudcentinel/hoa/internal/memory"
)

// MemoryTool exposes Oracle 23ai semantic search as a first-class agent tool.
// Registered explicitly in main (not via init) because it requires runtime config.
type MemoryTool struct {
	dsn    string
	apiKey string
}

// NewMemoryTool constructs a MemoryTool. Call tool.Default.Register(NewMemoryTool(...)) in main.
func NewMemoryTool(dsn, apiKey string) *MemoryTool {
	return &MemoryTool{dsn: dsn, apiKey: apiKey}
}

func (t *MemoryTool) Definition() api.ToolDef {
	return api.ToolDef{
		Name: "search_memory",
		Description: "Search Oracle 23ai project memory using semantic vector similarity. " +
			"Returns structured results with: what changed, why, relevance score (0=perfect, 0.55=cutoff), " +
			"and the actual current file content after each change. " +
			"PRIMARY knowledge source — always call this first. " +
			"If a result includes current file content, do NOT additionally call read_file for that file.",
		InputSchema: map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Natural language description of what you are looking for.",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Maximum number of results (default 5, max 20).",
			},
		},
		Required: []string{"query"},
	}
}

func (t *MemoryTool) Execute(_ context.Context, input string) (string, bool) {
	var in struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal([]byte(input), &in); err != nil {
		return fmt.Sprintf("invalid input: %v", err), true
	}
	if in.Limit <= 0 {
		in.Limit = 5
	}
	if in.Limit > 20 {
		in.Limit = 20
	}

	mc, err := memory.NewClient(t.dsn, t.apiKey)
	if err != nil {
		return fmt.Sprintf("oracle connect: %v", err), true
	}
	defer mc.Close()

	var parts []string

	if rules, err := mc.SearchFeedback(in.Query, 3); err == nil && len(rules) > 0 {
		parts = append(parts, memory.FormatFeedback(rules))
	}

	if results, err := memory.Search(mc, in.Query, in.Limit); err == nil && len(results) > 0 {
		parts = append(parts, memory.FormatContext(results))
	}

	if len(parts) == 0 {
		return "No relevant memory found for this query.", false
	}
	return strings.Join(parts, "\n\n"), false
}
