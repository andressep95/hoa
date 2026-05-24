// Package agent encapsulates the agent loop — the core of HOA.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/cloudcentinel/hoa/internal/api"
	"github.com/cloudcentinel/hoa/internal/memory"
	"github.com/cloudcentinel/hoa/internal/permission"
	"github.com/cloudcentinel/hoa/internal/provider"
	"github.com/cloudcentinel/hoa/internal/tool"
)

// OutputFunc is called by the agent to emit text or tool events.
type OutputFunc func(kind string, text string)

// WorkingContextFunc returns the structured uncommitted-changes context.
type WorkingContextFunc func() memory.WorkingChanges

// MemorySearchFunc searches project memory.
// Returns the formatted context to inject into the LLM prompt and the list of
// human-readable resource labels to display in the UI.
type MemorySearchFunc func(query string) (context string, labels []string)

// Agent owns one conversation: a provider, tools, and a message history.
type Agent struct {
	Provider       provider.Provider
	Tools          *tool.Registry
	System         string
	MaxTurns       int
	OnOutput       OutputFunc
	MemorySearch   MemorySearchFunc
	WorkingContext WorkingContextFunc
	VerifyCmd      string // build command to run after file modifications
	Policy         permission.Policy
	Confirm        func(prompt, detail string) permission.ConfirmResult // nil = auto-approve

	messages []api.Message
}

// New creates an Agent with sensible defaults.
func New(p provider.Provider, system string, tools *tool.Registry) *Agent {
	return &Agent{
		Provider: p,
		Tools:    tools,
		System:   system,
		MaxTurns: 20,
		OnOutput: func(kind, text string) { fmt.Println(text) },
	}
}

// Send appends a user message and runs the loop until the model stops.
func (a *Agent) Send(ctx context.Context, prompt string) (string, error) {
	var contextParts []string

	// 1. Memory context (Oracle — historical relevance, primary source)
	if a.MemorySearch != nil {
		mc, labels := a.MemorySearch(prompt)
		if mc != "" {
			contextParts = append(contextParts, mc)
			a.OnOutput("context", fmt.Sprintf("[mem] %d resultado(s) de Oracle:", len(labels)))
			for _, label := range labels {
				a.OnOutput("memory-item", label)
			}
		} else {
			a.OnOutput("context", "[mem] Oracle: sin resultados relevantes para esta consulta")
		}
	}

	// 2. Working context (uncommitted changes — requires user approval, single modal)
	if a.WorkingContext != nil {
		wc := a.WorkingContext()
		if len(wc.Files) > 0 {
			include := true
			if a.Confirm != nil {
				// Check memo: if user already pressed 'a' for working_context, skip modal.
				decide := permission.DecisionAsk
				if a.Policy != nil {
					decide, _ = a.Policy.Decide(ctx, "working_context", "")
				}
				if decide != permission.DecisionAllow {
					result := a.Confirm(
						fmt.Sprintf("incluir %d archivo(s) sin commitear como contexto?", len(wc.Files)),
						"",
					)
					switch result {
					case permission.ResultYes:
						// include once
					case permission.ResultAlways:
						if r, ok := a.Policy.(permission.Rememberer); ok {
							r.Remember("working_context")
						}
					case permission.ResultNo:
						include = false
					}
				}
			}
			if include && wc.Block != "" {
				contextParts = append(contextParts, wc.Block)
				a.OnOutput("context", fmt.Sprintf("[wc] %d archivo(s) aprobados:", len(wc.Files)))
				for _, f := range wc.Files {
					a.OnOutput("context-item", fmt.Sprintf("%s  %s", f.Path, memory.HumanBytes(f.SizeBytes)))
				}
			}
		}
	}

	fullPrompt := prompt
	if len(contextParts) > 0 {
		fullPrompt = strings.Join(contextParts, "\n\n") + "\n\n" + prompt
	}

	a.messages = append(a.messages, api.Message{
		Role:    api.RoleUser,
		Content: []api.Block{{Type: api.BlockText, Text: fullPrompt}},
	})
	return a.loop(ctx)
}

func (a *Agent) loop(ctx context.Context) (string, error) {
	var finalText strings.Builder

	for turn := 0; turn < a.MaxTurns; turn++ {
		resp, err := a.Provider.Send(ctx, a.messages, a.Tools.Definitions())
		if err != nil {
			return "", err
		}

		a.messages = append(a.messages, api.Message{Role: api.RoleAssistant, Content: resp.Content})

		var toolResults []api.Block
		var wroteFiles bool
		var turnText strings.Builder
		for _, b := range resp.Content {
			switch b.Type {
			case api.BlockText:
				if b.Text != "" {
					turnText.WriteString(b.Text)
					finalText.WriteString(b.Text)
					finalText.WriteString("\n")
				}
			case api.BlockToolUse:
				a.OnOutput("tool", formatToolCall(b.ToolName, b.ToolInput))
				result, isErr := a.executeTool(ctx, b.ToolName, b.ToolInput)
				if isWriteTool(b.ToolName, b.ToolInput) {
					wroteFiles = true
				}
				toolResults = append(toolResults, api.Block{
					Type:       api.BlockToolResult,
					ToolUseID:  b.ToolUseID,
					ToolResult: result,
					IsError:    isErr,
				})
			}
		}

		// Emit accumulated text as one block (for proper markdown rendering)
		if turnText.Len() > 0 {
			a.OnOutput("text", turnText.String())
		}

		if resp.StopReason != api.StopToolUse || len(toolResults) == 0 {
			return strings.TrimSpace(finalText.String()), nil
		}

		// Write-verify loop: run build only if a write-capable tool was used
		if a.VerifyCmd != "" && wroteFiles {
			if verifyErr := a.runVerify(); verifyErr != "" {
				a.OnOutput("verify", "[!] build failed")
				toolResults = append(toolResults, api.Block{
					Type:       api.BlockText,
					Text:       "BUILD FAILED. Fix the errors before continuing:\n" + verifyErr,
				})
			}
		}

		a.messages = append(a.messages, api.Message{Role: api.RoleUser, Content: toolResults})
	}
	return strings.TrimSpace(finalText.String()), fmt.Errorf("max turns (%d) reached", a.MaxTurns)
}

// Messages returns the conversation history.
func (a *Agent) Messages() []api.Message { return a.messages }

// ClearMessages wipes the conversation.
func (a *Agent) ClearMessages() { a.messages = a.messages[:0] }

// SendOneShot sends a prompt to the LLM without affecting conversation history.
// Used for internal tasks like commit message generation.
func (a *Agent) SendOneShot(ctx context.Context, prompt string) (string, error) {
	msgs := []api.Message{{
		Role:    api.RoleUser,
		Content: []api.Block{{Type: api.BlockText, Text: prompt}},
	}}
	resp, err := a.Provider.Send(ctx, msgs, nil)
	if err != nil {
		return "", err
	}
	var text strings.Builder
	for _, b := range resp.Content {
		if b.Type == api.BlockText {
			text.WriteString(b.Text)
		}
	}
	return strings.TrimSpace(text.String()), nil
}

// executeTool applies the permission policy, optionally asks for confirmation
// (with diff detail for write_file), then dispatches to the registry.
func (a *Agent) executeTool(ctx context.Context, name, rawInput string) (string, bool) {
	decision := permission.DecisionAsk
	reason := ""
	if a.Policy != nil {
		decision, reason = a.Policy.Decide(ctx, name, rawInput)
	}

	switch decision {
	case permission.DecisionAllow:
		// proceed
	case permission.DecisionDeny:
		if reason == "" {
			reason = "permission policy denied this tool call"
		}
		return reason, true
	case permission.DecisionAsk:
		prompt, detail := buildApprovalPrompt(name, rawInput)
		if a.Confirm != nil {
			result := a.Confirm(prompt, detail)
			switch result {
			case permission.ResultYes:
				// proceed once
			case permission.ResultAlways:
				if r, ok := a.Policy.(permission.Rememberer); ok {
					r.Remember(name)
				}
			case permission.ResultNo:
				return "user denied this tool call", true
			}
		}
	}
	return a.Tools.Execute(ctx, name, rawInput)
}

func (a *Agent) runVerify() string {
	out, err := exec.Command("sh", "-c", a.VerifyCmd).CombinedOutput()
	if err != nil {
		result := strings.TrimSpace(string(out))
		if len(result) > 3000 {
			result = result[:3000]
		}
		return result
	}
	return ""
}

// formatToolCall builds a human-readable label for a tool invocation.
// It extracts the most relevant argument so the user can see what the tool is doing.
func formatToolCall(name, input string) string {
	var args map[string]any
	if err := json.Unmarshal([]byte(input), &args); err != nil {
		return name
	}
	// Ordered by relevance per tool type
	for _, field := range []string{"command", "pattern", "path", "query", "glob", "content"} {
		if v, ok := args[field]; ok {
			if s, ok := v.(string); ok && s != "" {
				if len(s) > 70 {
					s = s[:67] + "..."
				}
				return name + " " + s
			}
		}
	}
	return name
}

func isWriteTool(name string, input string) bool {
	switch name {
	case "write_file", "edit_file":
		return true
	case "bash":
		return strings.Contains(input, ">") || strings.Contains(input, "tee ") ||
			strings.Contains(input, "mv ") || strings.Contains(input, "cp ") ||
			strings.Contains(input, "sed ") || strings.Contains(input, "mkdir ")
	}
	return false
}
