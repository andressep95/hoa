package provider

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/cloudcentinel/hoa/internal/api"
)

// AnthropicProvider implements Provider using the Anthropic SDK.
type AnthropicProvider struct {
	client    anthropic.Client
	model     anthropic.Model
	maxTokens int64
	system    string

	mu             sync.Mutex
	total          api.Usage
	knowledgeBlock string
}

func NewAnthropicProvider(apiKey string, model string, maxTokens int64, system string) *AnthropicProvider {
	opts := []option.RequestOption{}
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	return &AnthropicProvider{
		client:    anthropic.NewClient(opts...),
		model:     anthropic.Model(model),
		maxTokens: maxTokens,
		system:    system,
	}
}

func (p *AnthropicProvider) Model() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return string(p.model)
}

func (p *AnthropicProvider) SetModel(name string) {
	p.mu.Lock()
	p.model = anthropic.Model(name)
	p.mu.Unlock()
}

func (p *AnthropicProvider) TotalUsage() api.Usage {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.total
}

func (p *AnthropicProvider) SetKnowledgeContext(text string) {
	p.mu.Lock()
	p.knowledgeBlock = text
	p.mu.Unlock()
}

func (p *AnthropicProvider) Send(ctx context.Context, messages []api.Message, tools []api.ToolDef) (api.Response, error) {
	p.mu.Lock()
	kb := p.knowledgeBlock
	p.mu.Unlock()

	system := []anthropic.TextBlockParam{{
		Text:         p.system,
		CacheControl: anthropic.NewCacheControlEphemeralParam(),
	}}
	if kb != "" {
		system = append(system, anthropic.TextBlockParam{
			Text:         kb,
			CacheControl: anthropic.NewCacheControlEphemeralParam(),
		})
	}

	resp, err := p.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     p.model,
		MaxTokens: p.maxTokens,
		System:    system,
		Messages:  toAnthropicMessages(messages),
		Tools:     toAnthropicTools(tools),
	})
	if err != nil {
		return api.Response{}, err
	}

	out := api.Response{StopReason: fromStopReason(resp.StopReason)}
	for _, block := range resp.Content {
		switch v := block.AsAny().(type) {
		case anthropic.TextBlock:
			out.Content = append(out.Content, api.Block{Type: api.BlockText, Text: v.Text})
		case anthropic.ToolUseBlock:
			out.Content = append(out.Content, api.Block{
				Type:      api.BlockToolUse,
				ToolUseID: v.ID,
				ToolName:  v.Name,
				ToolInput: v.JSON.Input.Raw(),
			})
		}
	}

	out.Usage = api.Usage{
		InputTokens:         int(resp.Usage.InputTokens),
		OutputTokens:        int(resp.Usage.OutputTokens),
		CacheCreationTokens: int(resp.Usage.CacheCreationInputTokens),
		CacheReadTokens:     int(resp.Usage.CacheReadInputTokens),
	}
	p.mu.Lock()
	p.total = p.total.Add(out.Usage)
	p.mu.Unlock()

	return out, nil
}

func toAnthropicMessages(messages []api.Message) []anthropic.MessageParam {
	out := make([]anthropic.MessageParam, 0, len(messages))
	for _, m := range messages {
		blocks := make([]anthropic.ContentBlockParamUnion, 0, len(m.Content))
		for _, b := range m.Content {
			switch b.Type {
			case api.BlockText:
				blocks = append(blocks, anthropic.NewTextBlock(b.Text))
			case api.BlockToolUse:
				blocks = append(blocks, anthropic.ContentBlockParamUnion{
					OfToolUse: &anthropic.ToolUseBlockParam{
						ID:    b.ToolUseID,
						Name:  b.ToolName,
						Input: json.RawMessage(b.ToolInput),
					},
				})
			case api.BlockToolResult:
				blocks = append(blocks, anthropic.NewToolResultBlock(b.ToolUseID, b.ToolResult, b.IsError))
			}
		}
		switch m.Role {
		case api.RoleUser:
			out = append(out, anthropic.NewUserMessage(blocks...))
		case api.RoleAssistant:
			out = append(out, anthropic.NewAssistantMessage(blocks...))
		}
	}
	return out
}

func toAnthropicTools(tools []api.ToolDef) []anthropic.ToolUnionParam {
	if len(tools) == 0 {
		return nil
	}
	out := make([]anthropic.ToolUnionParam, 0, len(tools))
	for _, t := range tools {
		out = append(out, anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        t.Name,
				Description: anthropic.String(t.Description),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: t.InputSchema,
					Required:   t.Required,
				},
			},
		})
	}
	return out
}

func fromStopReason(s anthropic.StopReason) api.StopReason {
	switch s {
	case anthropic.StopReasonEndTurn:
		return api.StopEndTurn
	case anthropic.StopReasonToolUse:
		return api.StopToolUse
	default:
		return api.StopOther
	}
}
