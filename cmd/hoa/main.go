package main

import (
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
	banner := buildBanner(cfg.ActiveProvider, p.Models.Base, p.Models.Planning)

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
		TokensUsed:  func() (int, int) { return 0, 0 }, // TODO: track tokens
		ClearHist:   a.ClearMessages,
		ToolNames: func() []string {
			defs := tool.Default.Definitions()
			names := make([]string, len(defs))
			for i, d := range defs {
				names[i] = d.Name
			}
			return names
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
	p, _ := cfg.ActiveProviderConfig()
	apiKey := cfg.APIKey()

	switch cfg.ActiveProvider {
	case "openai":
		return provider.NewOpenAIProvider(apiKey, p.Models.Base, 4096, systemPrompt)
	default:
		return provider.NewAnthropicProvider(apiKey, p.Models.Base, 4096, systemPrompt)
	}
}

func buildBanner(providerName, baseModel, planModel string) string {
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
	out += "\n" + ui.StyleDim.Render("  /help para comandos · /exit para salir")
	return out
}
