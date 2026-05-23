# HOA — Project Index

## Structure

```
cmd/hoa/main.go              Entry point, wiring, banner
internal/
├── agent/agent.go           Agent loop + SendOneShot
├── api/types.go             Message, Block, Usage, ToolDef, Response
├── command/                  Slash command system
│   ├── registry.go          Dispatch, Context, MenuItem, Result
│   ├── clear.go             /clear
│   ├── commit.go            /commit (LLM-powered, JSON, validation)
│   ├── exit.go              /exit
│   ├── help.go              /help
│   ├── memory.go            /memory (placeholder)
│   ├── mode.go              /mode (execute / plan+execute)
│   ├── model.go             /model (interactive selector)
│   ├── provider.go          /provider (switch, add, modify key)
│   ├── tokens.go            /tokens (usage + cost)
│   ├── tools.go             /tools
│   └── validate.go          Conventional Commits validator
├── config/
│   ├── config.go            Load/Save, MemoryConfig, ResolveModel
│   ├── crypto.go            AES-256-GCM encryption
│   └── wizard.go            First-run TUI wizard
├── cost/tracker.go          Per-model pricing + FormatCost
├── provider/
│   ├── provider.go          Provider interface
│   ├── anthropic.go         Anthropic SDK implementation
│   └── openai.go            OpenAI SDK implementation
├── tool/
│   ├── registry.go          Tool registry + definitions
│   ├── bash.go              Shell execution
│   ├── readfile.go          File reading
│   ├── grep.go              Regex search
│   └── glob.go              File pattern matching
└── ui/
    ├── program.go           Bubble Tea main model (TUI)
    ├── styles.go            Centralized lipgloss styles
    ├── textinput.go         Input component (wizard)
    └── selector.go          Selector component (wizard)
```

## Key Patterns

- **Provider interface**: `Send()`, `Model()`, `SetModel()`, `TotalUsage()`
- **Command dispatch**: `command.Dispatch(ctx, input)` → `Result{Lines, Menu, AsyncFn, Quit}`
- **Async commands**: `Result.AsyncFn` runs in background with spinner
- **Menu system**: `Result.Menu` shows interactive selector, `Action() string` returns feedback
- **Config persistence**: Every SetModel/SetProvider/SetMode saves to `~/.hoa/config.json`
- **Model resolution**: env var (`ANTHROPIC_MODEL`) → config → default
- **Dynamic banner**: `func() string` re-evaluated on each render

## Docs

- [Fase 1 Stories](docs/phases/fase-1/stories/) — 16 user stories
- [README](README.md) — Full feature documentation
