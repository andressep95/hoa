package config

import (
	"fmt"

	"github.com/cloudcentinel/hoa/internal/ui"
)

var knownProviders = []struct {
	Name   string
	Label  string
	Models []string
}{
	{"anthropic", "Anthropic (Claude)", []string{"claude-sonnet-4-6", "claude-opus-4-7", "claude-haiku-4-5"}},
	{"openai", "OpenAI", []string{"gpt-4o", "o3", "o4-mini", "gpt-4o-mini"}},
	{"ollama", "Ollama (local)", []string{"llama3.1:70b", "deepseek-r1:32b", "codellama:34b"}},
	{"google", "Google (Gemini)", []string{"gemini-2.5-pro", "gemini-2.5-flash"}},
}

// RunWizard guides the user through first-time setup with TUI selectors.
func RunWizard() *Config {
	fmt.Println("\n🎛️  HOA — Primera configuración\n")

	// 1. Select provider
	providerItems := make([]ui.SelectorItem, len(knownProviders))
	for i, p := range knownProviders {
		providerItems[i] = ui.SelectorItem{Label: p.Label}
	}
	idx := ui.RunSelector("Provider principal:", providerItems)
	if idx < 0 {
		fmt.Println("Cancelado.")
		return defaultConfig()
	}
	provider := knownProviders[idx]

	// 2. API Key
	var apiKey string
	if provider.Name != "ollama" {
		apiKey = ui.RunInput(
			fmt.Sprintf("API Key para %s:", provider.Label),
			"sk-...", true,
		)
	}

	// 3. Base URL (ollama only)
	var baseURL string
	if provider.Name == "ollama" {
		baseURL = ui.RunInput("Base URL:", "http://localhost:11434", false)
		if baseURL == "" {
			baseURL = "http://localhost:11434"
		}
	}

	// 4. Base model
	modelItems := make([]ui.SelectorItem, len(provider.Models))
	for i, m := range provider.Models {
		modelItems[i] = ui.SelectorItem{Label: m}
	}
	baseIdx := ui.RunSelector("Modelo base (ejecución):", modelItems)
	if baseIdx < 0 {
		baseIdx = 0
	}
	baseModel := provider.Models[baseIdx]

	// 5. Planning model
	planIdx := ui.RunSelector("Modelo planning (razonamiento):", modelItems)
	if planIdx < 0 {
		planIdx = 0
	}
	planModel := provider.Models[planIdx]

	cfg := &Config{
		ActiveProvider: provider.Name,
		Providers: map[string]ProviderConfig{
			provider.Name: {
				APIKey:  apiKey,
				BaseURL: baseURL,
				Models: ModelsConfig{
					Base:     baseModel,
					Planning: planModel,
				},
			},
		},
		Memory: runMemorySetup(),
		Harness: HarnessConfig{
			VerifyAfterWrite: true,
			SDDEnforced:      true,
			MaxRetries:       3,
			CompactThreshold: 0.7,
		},
	}

	fmt.Printf("\n✅ Configuración lista (provider: %s, base: %s, planning: %s)\n\n",
		provider.Name, baseModel, planModel)
	return cfg
}

func defaultConfig() *Config {
	return &Config{
		ActiveProvider: "anthropic",
		Providers: map[string]ProviderConfig{
			"anthropic": {Models: ModelsConfig{Base: "claude-sonnet-4-6", Planning: "claude-opus-4-7"}},
		},
		Harness: HarnessConfig{VerifyAfterWrite: true, SDDEnforced: true, MaxRetries: 3, CompactThreshold: 0.7},
	}
}

func runMemorySetup() MemoryConfig {
	fmt.Println()
	idx := ui.RunSelector("¿Configurar memoria persistente (Oracle Vector DB)?", []ui.SelectorItem{
		{Label: "No, saltar por ahora"},
		{Label: "Sí, configurar conexión Oracle"},
	})
	if idx != 1 {
		return MemoryConfig{Enabled: false}
	}

	dsn := ui.RunInput("DSN Oracle (host:port/service):", "localhost:1521/FREEPDB1", false)
	user := ui.RunInput("Usuario:", "MCP_USER", false)
	pass := ui.RunInput("Password:", "", true)
	projectKey := ui.RunInput("API Key del proyecto:", "", false)

	return MemoryConfig{
		Enabled:  true,
		Provider: "oracle",
		DSN:      dsn,
		User:     user,
		Password: pass,
		APIKey:   projectKey,
	}
}
