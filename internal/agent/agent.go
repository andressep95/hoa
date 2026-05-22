// Package agent encapsulates the agent loop — the core of HOA.
package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudcentinel/hoa/internal/api"
	"github.com/cloudcentinel/hoa/internal/provider"
	"github.com/cloudcentinel/hoa/internal/tool"
)

// OutputFunc is called by the agent to emit text or tool events.
type OutputFunc func(kind string, text string)

// Agent owns one conversation: a provider, tools, and a message history.
type Agent struct {
	Provider provider.Provider
	Tools    *tool.Registry
	System   string
	MaxTurns int
	OnOutput OutputFunc

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
	a.messages = append(a.messages, api.Message{
		Role:    api.RoleUser,
		Content: []api.Block{{Type: api.BlockText, Text: prompt}},
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
		for _, b := range resp.Content {
			switch b.Type {
			case api.BlockText:
				if b.Text != "" {
					a.OnOutput("text", b.Text)
					finalText.WriteString(b.Text)
					finalText.WriteString("\n")
				}
			case api.BlockToolUse:
				a.OnOutput("tool", b.ToolName)
				result, isErr := a.Tools.Execute(ctx, b.ToolName, b.ToolInput)
				toolResults = append(toolResults, api.Block{
					Type:       api.BlockToolResult,
					ToolUseID:  b.ToolUseID,
					ToolResult: result,
					IsError:    isErr,
				})
			}
		}

		if resp.StopReason != api.StopToolUse || len(toolResults) == 0 {
			return strings.TrimSpace(finalText.String()), nil
		}

		a.messages = append(a.messages, api.Message{Role: api.RoleUser, Content: toolResults})
	}
	return strings.TrimSpace(finalText.String()), fmt.Errorf("max turns (%d) reached", a.MaxTurns)
}

// Messages returns the conversation history.
func (a *Agent) Messages() []api.Message { return a.messages }

// ClearMessages wipes the conversation.
func (a *Agent) ClearMessages() { a.messages = a.messages[:0] }
