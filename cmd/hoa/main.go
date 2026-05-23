package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/cloudcentinel/hoa/internal/agent"
	"github.com/cloudcentinel/hoa/internal/command"
	"github.com/cloudcentinel/hoa/internal/config"
	"github.com/cloudcentinel/hoa/internal/provider"
	"github.com/cloudcentinel/hoa/internal/tool"
	"github.com/cloudcentinel/hoa/internal/ui"
)

const systemPrompt = `You are HOA (Harness Oriented Agent), a coding assistant running in a terminal.
You have tools: bash, read_file, grep, glob. Use them to help the user.
Be concise. Answer in the user's language.`

var knownProvidersList = []struct {
	Name   string
	Models []string
}{
	{"anthropic", []string{"claude-sonnet-4-6", "claude-opus-4-7", "claude-haiku-4-5"}},
	{"openai", []string{"gpt-4o", "o3", "o4-mini", "gpt-4o-mini"}},
	{"ollama", []string{"llama3.1:70b", "deepseek-r1:32b", "codellama:34b"}},
	{"google", []string{"gemini-2.5-pro", "gemini-2.5-flash"}},
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
			os.Exit(1)
		}
		cfg = config.RunWizard()
		if err := config.Save(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "error saving config: %v\n", err)
			os.Exit(1)
		}
	}

	llm := newProvider(cfg)
	a := agent.New(llm, systemPrompt, tool.Default)

	p, _ := cfg.ActiveProviderConfig()
	mode := "execute" // default mode
	banner := buildBanner(cfg.ActiveProvider, p.Models.Base, p.Models.Planning, mode)

	cmdCtx := &command.Context{
		GetModel:    llm.Model,
		SetModel:    llm.SetModel,
		GetPlanModel: func() string { p, _ := cfg.ActiveProviderConfig(); return p.Models.Planning },
		SetPlanModel: func(name string) {
			p, _ := cfg.ActiveProviderConfig()
			p.Models.Planning = name
			cfg.Providers[cfg.ActiveProvider] = p
		},
		GetProvider: func() string { return cfg.ActiveProvider },
		SetProvider: func(name string) { cfg.ActiveProvider = name },
		GetModels: func() []string {
			for _, kp := range knownProvidersList {
				if kp.Name == cfg.ActiveProvider {
					return kp.Models
				}
			}
			return nil
		},
		GetProviders: func() []string {
			names := make([]string, 0, len(cfg.Providers))
			for n := range cfg.Providers {
				names = append(names, n)
			}
			return names
		},
		GetMode: func() string { return mode },
		SetMode: func(m string) { mode = m },
		TokensUsed: func() (int, int) {
			u := llm.TotalUsage()
			return u.InputTokens, u.OutputTokens
		},
		ClearHist:   a.ClearMessages,
		ToolNames: func() []string {
			defs := tool.Default.Definitions()
			names := make([]string, len(defs))
			for i, d := range defs {
				names[i] = d.Name
			}
			return names
		},
		AgentSend: func(prompt string) (string, error) {
			return a.SendOneShot(context.Background(), prompt)
		},
	}

	prog, outputFn := ui.NewProgram(banner, a.Send, cmdCtx)
	a.OnOutput = outputFn

	if _, err := prog.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func newProvider(cfg *config.Config) provider.Provider {
	apiKey := cfg.APIKey()
	model := cfg.ResolveModel()

	switch cfg.ActiveProvider {
	case "openai":
		return provider.NewOpenAIProvider(apiKey, model, 4096, systemPrompt)
	default:
		return provider.NewAnthropicProvider(apiKey, model, 4096, systemPrompt)
	}
}

func buildBanner(providerName, baseModel, planModel, mode string) string {
	banner := `
  ██╗  ██╗ ██████╗  █████╗ 
  ██║  ██║██╔═══██╗██╔══██╗
  ███████║██║   ██║███████║
  ██╔══██║██║   ██║██╔══██║
  ██║  ██║╚██████╔╝██║  ██║
  ╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═╝`

	out := ui.StyleTitle.Render(banner) + "\n"
	out += ui.StyleSubtitle.Render("  Harness-Oriented Agents") + "\n\n"
	out += fmt.Sprintf("  %s %s\n", ui.StyleDim.Render("provider:"), providerName)
	out += fmt.Sprintf("  %s %s\n", ui.StyleDim.Render("base:"), baseModel)
	out += fmt.Sprintf("  %s %s\n", ui.StyleDim.Render("planning:"), planModel)
	out += fmt.Sprintf("  %s %s\n", ui.StyleDim.Render("mode:"), mode)
	out += "\n" + ui.StyleDim.Render("  /help para comandos · /exit para salir")
	return out
}
