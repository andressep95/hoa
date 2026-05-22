package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/cloudcentinel/hoa/internal/agent"
	"github.com/cloudcentinel/hoa/internal/config"
	"github.com/cloudcentinel/hoa/internal/provider"
	"github.com/cloudcentinel/hoa/internal/tool"
)

const systemPrompt = `You are HOA (Harness Oriented Agent), a coding assistant running in a terminal.
You have tools: bash, read_file, grep, glob. Use them to help the user.
Be concise. Answer in the user's language.`

var (
	promptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Bold(true)
	dimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	toolStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("75"))
)

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
	printBanner(cfg.ActiveProvider, p.Models.Base, p.Models.Planning)

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	ctx := context.Background()

	for {
		fmt.Print(promptStyle.Render("❯ "))
		if !scanner.Scan() {
			fmt.Println()
			return
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		if handleCommand(input, a) {
			continue
		}
		if input == "/exit" {
			return
		}

		fmt.Println(dimStyle.Render("  pensando..."))
		if _, err := a.Send(ctx, input); err != nil {
			fmt.Println(errorStyle.Render(fmt.Sprintf("  error: %v", err)))
		}
		fmt.Println()
	}
}

func handleCommand(input string, a *agent.Agent) bool {
	switch {
	case input == "/clear":
		a.ClearMessages()
		fmt.Println(dimStyle.Render("  Historial limpiado."))
		return true
	case input == "/tools":
		for _, t := range tool.Default.Definitions() {
			fmt.Printf("  %s — %s\n", toolStyle.Render(t.Name), t.Description)
		}
		return true
	case input == "/help":
		fmt.Println("  /tools   — Lista herramientas disponibles")
		fmt.Println("  /clear   — Limpia historial de conversación")
		fmt.Println("  /exit    — Salir")
		return true
	}
	return false
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

func printBanner(providerName, baseModel, planModel string) {
	banner := `
  ██╗  ██╗ ██████╗  █████╗ 
  ██║  ██║██╔═══██╗██╔══██╗
  ███████║██║   ██║███████║
  ██╔══██║██║   ██║██╔══██║
  ██║  ██║╚██████╔╝██║  ██║
  ╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═╝`

	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("33")).Bold(true)
	subtitleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("75"))

	fmt.Println(titleStyle.Render(banner))
	fmt.Println(subtitleStyle.Render("  Harness-Oriented Agents"))
	fmt.Println()
	fmt.Printf("  %s %s\n", dimStyle.Render("provider:"), providerName)
	fmt.Printf("  %s %s\n", dimStyle.Render("base:"), baseModel)
	fmt.Printf("  %s %s\n", dimStyle.Render("planning:"), planModel)
	fmt.Println()
	fmt.Println(dimStyle.Render("  /help para comandos · /exit para salir"))
	fmt.Println()
}
