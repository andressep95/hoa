package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/cloudcentinel/hoa/internal/agent"
	"github.com/cloudcentinel/hoa/internal/command"
	"github.com/cloudcentinel/hoa/internal/config"
	"github.com/cloudcentinel/hoa/internal/cost"
	"github.com/cloudcentinel/hoa/internal/health"
	"github.com/cloudcentinel/hoa/internal/memory"
	"github.com/cloudcentinel/hoa/internal/permission"
	"github.com/cloudcentinel/hoa/internal/provider"
	"github.com/cloudcentinel/hoa/internal/stack"
	"github.com/cloudcentinel/hoa/internal/tool"
	"github.com/cloudcentinel/hoa/internal/ui"
)

const systemPrompt = `You are HOA (Harness Oriented Agent), a coding assistant with persistent vector memory backed by Oracle 23ai.

MEMORY ARCHITECTURE:
- Oracle 23ai holds the authoritative semantic history: every commit, file change, intent, decision, and feedback rule.
- <project_knowledge> lists every file Oracle has indexed with its latest description. This is your routing map.
- search_memory returns structured results with actual file content (complete or labelled truncated), relevance score, what changed, and why.
- Score: 0.0 = perfect match → 0.55 = cutoff. Results above 0.55 are filtered. Trust scores below 0.3 fully.
- <working_changes> are uncommitted diffs the user approved — use as a patch on top of Oracle results.

TOOL DECISION GUIDE:
- search_memory(query)    → "find commits/decisions RELATED TO a concept" (vector/semantic)
- oracle_query(type,...)  → "what HAPPENED to X", "WHO changed Y", "WHEN was Z introduced" (structured SQL)
- read_file(path)         → file content (Oracle-backed: returns indexed version if available, disk otherwise)
- bash/grep/glob          → live filesystem: new files, running tests, anything not in Oracle

TWO-STAGE ROUTING:
1. Check <project_knowledge>. If the file/topic is listed → use search_memory or oracle_query first.
2. If search_memory returns "complete" content for a file → do NOT additionally call read_file.
3. If no Oracle results → fall through to filesystem tools.
4. <working_changes> supplements Oracle for uncommitted edits.

Never explain routing decisions — just act and answer.
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

	// Detect project stack for write-verify loop
	proj := stack.Detect()
	if proj.BuildCmd != "" {
		a.VerifyCmd = proj.BuildCmd
	}

	// Wire Oracle memory if enabled.
	// - Registers search_memory as an explicit tool the LLM can invoke.
	// - Caches a high-level project knowledge block in the system prompt (one-shot at startup).
	if cfg.Memory.Enabled && cfg.Memory.DSN != "" && cfg.Memory.APIKey != "" {
		tool.Default.Register(tool.NewMemoryTool(cfg.Memory.DSN, cfg.Memory.APIKey))
		tool.Default.Register(tool.NewOracleReadFileTool(cfg.Memory.DSN, cfg.Memory.APIKey))
		tool.Default.Register(tool.NewOracleQueryTool(cfg.Memory.DSN, cfg.Memory.APIKey))

		// Startup: fetch high-level project knowledge and cache it in the system prompt.
		// Anthropic caches this block across turns within the 5-min TTL window.
		if mc, err := memory.NewClient(cfg.Memory.DSN, cfg.Memory.APIKey); err == nil {
			if knowledge, err := memory.FetchProjectKnowledge(mc); err == nil && knowledge != "" {
				a.Provider.SetKnowledgeContext(knowledge)
			}
			mc.Close()
		}
	}

	// Working context — always active (uses git directly). Cached for the
	// footer / /status to avoid running git every render.
	var (
		workingMu       sync.Mutex
		workingSnapshot memory.WorkingChanges
	)
	a.WorkingContext = func() memory.WorkingChanges {
		wc := memory.WorkingContext()
		workingMu.Lock()
		workingSnapshot = wc
		workingMu.Unlock()
		return wc
	}

	mode := cfg.Harness.Mode
	if mode == "" {
		mode = "execute"
	}

	bannerFn := func() string {
		pc, _ := cfg.ActiveProviderConfig()
		return buildBanner(cfg.ActiveProvider, a.Provider.Model(), pc.Models.Planning, mode)
	}

	// Declared up front so cmdCtx closures can capture them; they're
	// assigned below after `prog` exists.
	policy := permission.NewAskOnceMemo()
	a.Policy = policy
	var monitor *health.Monitor

	cmdCtx := &command.Context{
		GetModel: func() string { return a.Provider.Model() },
		SetModel: func(name string) {
			a.Provider.SetModel(name)
			p, _ := cfg.ActiveProviderConfig()
			p.Models.Base = name
			cfg.Providers[cfg.ActiveProvider] = p
			config.Save(cfg)
		},
		GetPlanModel: func() string { p, _ := cfg.ActiveProviderConfig(); return p.Models.Planning },
		SetPlanModel: func(name string) {
			p, _ := cfg.ActiveProviderConfig()
			p.Models.Planning = name
			cfg.Providers[cfg.ActiveProvider] = p
			config.Save(cfg)
		},
		GetProvider: func() string { return cfg.ActiveProvider },
		SetProvider: func(name string) {
			cfg.ActiveProvider = name
			newLLM := newProvider(cfg)
			a.Provider = newLLM
			config.Save(cfg)
		},
		SetupProvider: func(name string) {
			// Prompt for API key using TUI input (runs outside alt-screen)
			apiKey := ui.RunInput(fmt.Sprintf("API Key para %s:", name), "sk-...", true)
			if apiKey == "" {
				return
			}
			// Find default model for this provider
			model := "claude-sonnet-4-6"
			for _, kp := range knownProvidersList {
				if kp.Name == name && len(kp.Models) > 0 {
					model = kp.Models[0]
				}
			}
			cfg.Providers[name] = config.ProviderConfig{
				APIKey: apiKey,
				Models: config.ModelsConfig{Base: model, Planning: model},
			}
			config.Save(cfg)
		},
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
		SetMode: func(m string) {
			mode = m
			cfg.Harness.Mode = m
			config.Save(cfg)
		},
		TokensUsed: func() (int, int) {
			u := a.Provider.TotalUsage()
			return u.InputTokens, u.OutputTokens
		},
		CostTotal: func() float64 {
			u := a.Provider.TotalUsage()
			return cost.EstimateForModel(a.Provider.Model(), u.InputTokens, u.OutputTokens)
		},
		WorkingCount: func() int {
			workingMu.Lock()
			defer workingMu.Unlock()
			return len(workingSnapshot.Files)
		},
		WorkingSnapshot: func() command.WorkingSnapshotData {
			workingMu.Lock()
			defer workingMu.Unlock()
			out := command.WorkingSnapshotData{Files: make([]command.FileSnapshot, 0, len(workingSnapshot.Files))}
			for _, f := range workingSnapshot.Files {
				out.Files = append(out.Files, command.FileSnapshot{Path: f.Path, SizeBytes: f.SizeBytes})
			}
			return out
		},
		OracleStatus: func() (bool, error, time.Time) {
			if monitor == nil {
				return false, nil, time.Time{}
			}
			return monitor.Status()
		},
		RememberedTools: func() []string {
			return policy.Remembered()
		},
		ClearHist: a.ClearMessages,
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

		// Memory
		MemoryEnabled: func() bool { return cfg.Memory.Enabled },
		MemoryDSN:     func() string { return cfg.Memory.DSN },
		MemoryAPIKey:  func() string { return cfg.Memory.APIKey },
		SetMemory: func(enabled bool) {
			cfg.Memory.Enabled = enabled
			config.Save(cfg)
		},
		SetMemoryDSN: func(dsn string) {
			cfg.Memory.DSN = dsn
			config.Save(cfg)
		},
		SetMemoryKey: func(apiKey string) {
			cfg.Memory.APIKey = apiKey
			config.Save(cfg)
		},
		PromptInput: ui.RunInput,
	}

	prog, outputFn := ui.NewProgram(bannerFn, a.Send, cmdCtx)
	a.OnOutput = outputFn

	// Confirm callback uses prog (declared above) to send approval requests.
	a.Confirm = func(prompt, detail string) permission.ConfirmResult {
		reply := make(chan permission.ConfirmResult, 1)
		prog.Send(ui.ApprovalRequest{Prompt: prompt, Detail: detail, Reply: reply})
		return <-reply
	}

	// Oracle health monitor: heartbeat every 30s; pushes status to footer.
	if cfg.Memory.Enabled && cfg.Memory.DSN != "" {
		monitor = health.NewMonitor()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		monitor.Start(ctx, cfg.Memory.DSN, 30*time.Second)
		go func() {
			// Initial push (the immediate first tick will fire shortly).
			tick := time.NewTicker(200 * time.Millisecond)
			defer tick.Stop()
			pushed := false
			for !pushed {
				select {
				case <-ctx.Done():
					return
				case <-tick.C:
					ok, oerr, since := monitor.Status()
					if !since.IsZero() {
						configured := true
						prog.Send(ui.FooterUpdate{
							OracleConfigured: &configured,
							OracleOK:         &ok,
							OracleErr:        oerr,
							ResetOracleErr:   oerr == nil,
						})
						pushed = true
					}
				}
			}
			// Subsequent updates from the channel.
			for {
				select {
				case <-ctx.Done():
					return
				case <-monitor.Updates():
					ok, oerr, _ := monitor.Status()
					configured := true
					prog.Send(ui.FooterUpdate{
						OracleConfigured: &configured,
						OracleOK:         &ok,
						OracleErr:        oerr,
						ResetOracleErr:   oerr == nil,
					})
				}
			}
		}()
	}

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
	out += ui.StyleSubtitle.Render("  Harness Oriented Agents") + "\n\n"
	out += fmt.Sprintf("  %s %s\n", ui.StyleDim.Render("provider:"), providerName)
	out += fmt.Sprintf("  %s %s\n", ui.StyleDim.Render("base:"), baseModel)
	out += fmt.Sprintf("  %s %s\n", ui.StyleDim.Render("planning:"), planModel)
	out += fmt.Sprintf("  %s %s\n", ui.StyleDim.Render("mode:"), mode)
	return out
}
