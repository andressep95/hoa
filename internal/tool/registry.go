// Package tool defines the Tool interface and a Registry. Tools self-register
// via init() — adding a tool means dropping a file in this directory.
package tool

import (
	"context"
	"fmt"
	"sort"

	"github.com/cloudcentinel/hoa/internal/api"
)

// Tool is the contract every tool implements.
type Tool interface {
	Definition() api.ToolDef
	Execute(ctx context.Context, input string) (result string, isError bool)
}

// Registry holds tools by name.
type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: map[string]Tool{}}
}

func (r *Registry) Register(t Tool) {
	r.tools[t.Definition().Name] = t
}

// Definitions returns all tool schemas sorted by name (deterministic for caching).
func (r *Registry) Definitions() []api.ToolDef {
	names := make([]string, 0, len(r.tools))
	for n := range r.tools {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]api.ToolDef, 0, len(names))
	for _, n := range names {
		out = append(out, r.tools[n].Definition())
	}
	return out
}

// Execute dispatches a tool call by name.
func (r *Registry) Execute(ctx context.Context, name, input string) (string, bool) {
	t, ok := r.tools[name]
	if !ok {
		return fmt.Sprintf("unknown tool: %s", name), true
	}
	return t.Execute(ctx, input)
}

// Default is the package-level registry. Tools self-register here via init().
var Default = NewRegistry()
