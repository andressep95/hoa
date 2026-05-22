package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/cloudcentinel/hoa/internal/agent"
	"github.com/cloudcentinel/hoa/internal/config"
	"github.com/cloudcentinel/hoa/internal/provider"
	"github.com/cloudcentinel/hoa/internal/tool"
	"github.com/cloudcentinel/hoa/internal/ui"
)

const systemPrompt = `You are HOA (Harness Oriented Agent), a coding assistant running in a terminal.
You have tools: bash, read_file, grep, glob. Use them to help the user.
Be concise. Answer in the user's language.`

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

	prog, outputFn := ui.NewProgram(banner, a.Send)
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
	out += ui.StyleSubtitle.Render("  Harness Oriented Agents") + "\n\n"
	out += fmt.Sprintf("  %s %s\n", ui.StyleDim.Render("provider:"), providerName)
	out += fmt.Sprintf("  %s %s\n", ui.StyleDim.Render("base:"), baseModel)
	out += fmt.Sprintf("  %s %s\n", ui.StyleDim.Render("planning:"), planModel)
	out += "\n" + ui.StyleDim.Render("  /help para comandos · /exit para salir")
	return out
}
