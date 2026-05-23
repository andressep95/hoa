# HOA — Project Index

## Structure

```
cmd/hoa/main.go              Entry point, wiring, banner, memory + cache setup
internal/
├── agent/agent.go           Agent loop + WorkingContext + MemorySearch injection
├── api/types.go             Message, Block, Usage (with cache tokens), ToolDef, Response
├── command/                  Slash command system
│   ├── registry.go          Dispatch, Context (incl. Memory fields), MenuItem (AsyncAction)
│   ├── clear.go             /clear
│   ├── commit.go            /commit (LLM-powered + post-commit memory push)
│   ├── exit.go              /exit
│   ├── feedback.go          /feedback (save, list)
│   ├── help.go              /help
│   ├── memory.go            /memory (status, enable, disable, sync) + wizard
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
├── memory/                   Memory Provider (Oracle 23ai)
│   ├── client.go            Oracle connection, BatchInsert, CreateProject, CountIndexed
│   ├── extractor.go         Extract() — git log/diff → entries (deterministic, no LLM)
│   ├── sync.go              Sync() + SyncOne() with concurrent enrichment
│   ├── enrichment.go        EnrichmentProcessor (async goroutine, drains queue via LLM)
│   ├── search.go            Search() — VECTOR_DISTANCE semantic search
│   ├── working.go           WorkingContext() — git diff as session memory
│   └── feedback.go          SaveFeedback, SearchFeedback, FormatFeedback
├── provider/
│   ├── provider.go          Provider interface
│   ├── anthropic.go         Anthropic SDK + prompt caching (system + automatic)
│   └── openai.go            OpenAI SDK implementation
├── tool/
│   ├── registry.go          Tool registry + definitions
│   ├── bash.go              Shell execution
│   ├── readfile.go          File reading
│   ├── grep.go              Regex search
│   └── glob.go              File pattern matching
└── ui/
    ├── program.go           Bubble Tea main model (TUI + menu + async + cancel feedback)
    ├── styles.go            Centralized lipgloss styles
    ├── textinput.go         Input component (wizard)
    └── selector.go          Selector component (wizard)

docker/
├── docker-compose.yml       Oracle 23ai (gvenzl/oracle-free:23.7-slim)
├── setup-model.sh           Downloads Oracle pre-converted ONNX model
└── oracle/
    ├── init/01-schema.sql   All tables (projects, memory_changes, hunks, enrichment, feedback_rules)
    ├── init/02-embedding-model.sql  ONNX model load + auto-embedding triggers
    └── models/              all_MiniLM_L12_v2.onnx (384 dims, gitignored)
```

## Key Patterns

- **Provider interface**: `Send()`, `Model()`, `SetModel()`, `TotalUsage()`
- **Prompt caching**: system prompt + automatic caching (90% savings on turns 2+)
- **Command dispatch**: `command.Dispatch(ctx, input)` → `Result{Lines, Menu, AsyncFn, Quit}`
- **Async commands**: `Result.AsyncFn` runs in background with spinner
- **Menu system**: `Result.Menu` with `Action` (sync) and `AsyncAction` (with spinner)
- **Cancel feedback**: Esc in menu shows "⎿ cancelled"
- **Config persistence**: Every SetModel/SetProvider/SetMode saves to `~/.hoa/config.json`
- **Model resolution**: env var → config → default

## Memory Architecture

```
User prompt → [WorkingContext + FeedbackRules + ProjectMemory] → LLM (cached)
                    │                  │                │
                    │                  │                └─ VECTOR_DISTANCE on memory_changes
                    │                  └─ VECTOR_DISTANCE on feedback_rules
                    └─ git diff (auto-clears on /commit)

/commit → Extract → BatchInsert → Oracle (trigger generates embedding)
                                      └─ NeedsEnrichment? → async LLM enrichment
```

## Docs

- [Fase 1 README](docs/phases/fase-1/README.md) — Architecture + implementation order
- [Fase 1 Stories](docs/phases/fase-1/stories/) — User stories with status
- [HARNESS.md](docs/HARNESS.md) — Harness engineering bible (improvement roadmap)
