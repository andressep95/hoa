// Package api defines provider-agnostic message types used across the harness.
// Providers translate to/from their SDK's native shape; the rest of HOA speaks these types.
package api

import (
	"fmt"
	"strings"
)

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type BlockType string

const (
	BlockText       BlockType = "text"
	BlockToolUse    BlockType = "tool_use"
	BlockToolResult BlockType = "tool_result"
)

// Block is one piece of message content. Fields are interpreted based on Type.
type Block struct {
	Type BlockType

	Text string // BlockText

	ToolUseID string // BlockToolUse, BlockToolResult
	ToolName  string // BlockToolUse
	ToolInput string // BlockToolUse — raw JSON

	ToolResult string // BlockToolResult
	IsError    bool   // BlockToolResult
}

// Message is one turn in the conversation.
type Message struct {
	Role    Role
	Content []Block
}

// HasToolResult reports whether the message contains any tool_result blocks.
func (m Message) HasToolResult() bool {
	for _, b := range m.Content {
		if b.Type == BlockToolResult {
			return true
		}
	}
	return false
}

// ToolDef describes a tool the model can call.
type ToolDef struct {
	Name        string
	Description string
	InputSchema map[string]any
	Required    []string
}

type StopReason string

const (
	StopEndTurn StopReason = "end_turn"
	StopToolUse StopReason = "tool_use"
	StopOther   StopReason = "other"
)

// Response is what a provider returns from a single Send call.
type Response struct {
	Content    []Block
	StopReason StopReason
	Usage      Usage
}

// Usage reports token accounting for one API call.
type Usage struct {
	InputTokens         int
	OutputTokens        int
	CacheCreationTokens int
	CacheReadTokens     int
}

// Add returns the per-field sum.
func (u Usage) Add(other Usage) Usage {
	return Usage{
		InputTokens:         u.InputTokens + other.InputTokens,
		OutputTokens:        u.OutputTokens + other.OutputTokens,
		CacheCreationTokens: u.CacheCreationTokens + other.CacheCreationTokens,
		CacheReadTokens:     u.CacheReadTokens + other.CacheReadTokens,
	}
}

// RenderTranscript serializes messages to a human-readable string.
func RenderTranscript(msgs []Message) string {
	var sb strings.Builder
	for _, m := range msgs {
		sb.WriteString(string(m.Role))
		sb.WriteString(": ")
		for _, b := range m.Content {
			switch b.Type {
			case BlockText:
				sb.WriteString(b.Text)
			case BlockToolUse:
				fmt.Fprintf(&sb, "[call %s %s]", b.ToolName, b.ToolInput)
			case BlockToolResult:
				fmt.Fprintf(&sb, "[result: %s]", b.ToolResult)
			}
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
