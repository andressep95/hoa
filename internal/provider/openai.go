package provider

import (
	"context"
	"strings"
	"sync"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/packages/param"
	"github.com/openai/openai-go/shared"

	"github.com/cloudcentinel/hoa/internal/api"
)

// OpenAIProvider implements Provider using the OpenAI SDK.
type OpenAIProvider struct {
	client    openai.Client
	model     string
	maxTokens int64
	system    string

	mu    sync.Mutex
	total api.Usage
}

func NewOpenAIProvider(apiKey string, model string, maxTokens int64, system string) *OpenAIProvider {
	opts := []option.RequestOption{}
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	return &OpenAIProvider{
		client:    openai.NewClient(opts...),
		model:     model,
		maxTokens: maxTokens,
		system:    system,
	}
}

func (p *OpenAIProvider) Model() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.model
}

func (p *OpenAIProvider) SetModel(name string) {
	p.mu.Lock()
	p.model = name
	p.mu.Unlock()
}

func (p *OpenAIProvider) TotalUsage() api.Usage {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.total
}

func (p *OpenAIProvider) Send(ctx context.Context, messages []api.Message, tools []api.ToolDef) (api.Response, error) {
	p.mu.Lock()
	model := p.model
	p.mu.Unlock()

	resp, err := p.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:               shared.ChatModel(model),
		Messages:            p.toMessages(messages),
		Tools:               p.toTools(tools),
		MaxCompletionTokens: param.NewOpt(p.maxTokens),
	})
	if err != nil {
		return api.Response{}, err
	}
	if len(resp.Choices) == 0 {
		return api.Response{StopReason: api.StopOther}, nil
	}
	choice := resp.Choices[0]

	out := api.Response{StopReason: fromFinishReason(choice.FinishReason)}
	if choice.Message.Content != "" {
		out.Content = append(out.Content, api.Block{Type: api.BlockText, Text: choice.Message.Content})
	}
	for _, tc := range choice.Message.ToolCalls {
		out.Content = append(out.Content, api.Block{
			Type:      api.BlockToolUse,
			ToolUseID: tc.ID,
			ToolName:  tc.Function.Name,
			ToolInput: tc.Function.Arguments,
		})
	}

	out.Usage = api.Usage{
		InputTokens:     int(resp.Usage.PromptTokens),
		OutputTokens:    int(resp.Usage.CompletionTokens),
		CacheReadTokens: int(resp.Usage.PromptTokensDetails.CachedTokens),
	}
	p.mu.Lock()
	p.total = p.total.Add(out.Usage)
	p.mu.Unlock()

	return out, nil
}

func (p *OpenAIProvider) toMessages(messages []api.Message) []openai.ChatCompletionMessageParamUnion {
	out := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages)+1)
	if p.system != "" {
		out = append(out, openai.SystemMessage(p.system))
	}
	for _, m := range messages {
		switch m.Role {
		case api.RoleUser:
			var textParts []string
			for _, b := range m.Content {
				switch b.Type {
				case api.BlockText:
					textParts = append(textParts, b.Text)
				case api.BlockToolResult:
					out = append(out, openai.ToolMessage(b.ToolResult, b.ToolUseID))
				}
			}
			if len(textParts) > 0 {
				out = append(out, openai.UserMessage(strings.Join(textParts, "\n")))
			}
		case api.RoleAssistant:
			var text strings.Builder
			var toolCalls []openai.ChatCompletionMessageToolCallParam
			for _, b := range m.Content {
				switch b.Type {
				case api.BlockText:
					text.WriteString(b.Text)
				case api.BlockToolUse:
					toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCallParam{
						ID: b.ToolUseID,
						Function: openai.ChatCompletionMessageToolCallFunctionParam{
							Name:      b.ToolName,
							Arguments: b.ToolInput,
						},
					})
				}
			}
			msg := openai.ChatCompletionAssistantMessageParam{}
			if text.Len() > 0 {
				msg.Content.OfString = param.NewOpt(text.String())
			}
			if len(toolCalls) > 0 {
				msg.ToolCalls = toolCalls
			}
			out = append(out, openai.ChatCompletionMessageParamUnion{OfAssistant: &msg})
		}
	}
	return out
}

func (p *OpenAIProvider) toTools(tools []api.ToolDef) []openai.ChatCompletionToolParam {
	if len(tools) == 0 {
		return nil
	}
	out := make([]openai.ChatCompletionToolParam, 0, len(tools))
	for _, t := range tools {
		parameters := map[string]any{
			"type":       "object",
			"properties": t.InputSchema,
		}
		if len(t.Required) > 0 {
			parameters["required"] = t.Required
		}
		out = append(out, openai.ChatCompletionToolParam{
			Function: shared.FunctionDefinitionParam{
				Name:        t.Name,
				Description: param.NewOpt(t.Description),
				Parameters:  shared.FunctionParameters(parameters),
			},
		})
	}
	return out
}

func fromFinishReason(r string) api.StopReason {
	switch r {
	case "stop":
		return api.StopEndTurn
	case "tool_calls":
		return api.StopToolUse
	default:
		return api.StopOther
	}
}
