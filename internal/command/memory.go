package command

import (
	"fmt"

	"github.com/cloudcentinel/hoa/internal/memory"
)

func cmdMemory(ctx *Context, args string) Result {
	switch args {
	case "status", "":
		return memoryStatus(ctx)
	case "enable":
		return memorySetup(ctx)
	case "disable":
		return memoryDisable(ctx)
	case "sync":
		return memorySync(ctx)
	default:
		return memoryStatus(ctx)
	}
}

func memoryStatus(ctx *Context) Result {
	enabled := ctx.MemoryEnabled != nil && ctx.MemoryEnabled()

	if !enabled {
		return Result{
			Title: "  🧠 Memoria: deshabilitada",
			Lines: []string{
				"",
				"  provider: oracle 23ai",
				"  estado:   ○ desconectada",
				"",
			},
			Menu: []MenuItem{
				{
					Label:  "configurar",
					Hint:   "conectar a Oracle",
					Action: func() string { return doMemorySetup(ctx) },
				},
			},
		}
	}

	dsn := ctx.MemoryDSN()
	apiKey := ctx.MemoryAPIKey()
	if dsn == "" || apiKey == "" {
		return Result{
			Title: "  🧠 Memoria: ⚠️  sin configurar",
			Lines: []string{
				"",
				"  provider: oracle 23ai",
				"  estado:   ⚠️  falta DSN / API Key",
				"",
			},
			Menu: []MenuItem{
				{
					Label:  "configurar",
					Hint:   "ingresar DSN y API Key",
					Action: func() string { return doMemorySetup(ctx) },
				},
				{
					Label:  "disable",
					Action: func() string { ctx.SetMemory(false); return "deshabilitada" },
				},
			},
		}
	}

	client, err := memory.NewClient(dsn, apiKey)
	if err != nil {
		return Result{
			Title: "  🧠 Memoria: ✗ error de conexión",
			Lines: []string{
				"",
				"  provider: oracle 23ai",
				"  estado:   ✗ " + err.Error(),
				"",
			},
			Menu: []MenuItem{
				{
					Label:  "reconfigurar",
					Action: func() string { return doMemorySetup(ctx) },
				},
				{
					Label:  "disable",
					Action: func() string { ctx.SetMemory(false); return "deshabilitada" },
				},
			},
		}
	}
	defer client.Close()

	indexed, pending, err := client.CountIndexed()
	if err != nil {
		return Result{Lines: []string{"  ❌ Error consultando: " + err.Error()}}
	}

	lines := []string{
		"",
		"  provider:    oracle 23ai",
		"  estado:      ● conectada",
		fmt.Sprintf("  indexados:   %d archivos", indexed),
	}
	if pending > 0 {
		lines = append(lines, fmt.Sprintf("  enrichment:  %d pendientes", pending))
	}
	lines = append(lines, "")

	return Result{
		Title: "  🧠 Memoria: ✓ conectada",
		Lines: lines,
		Menu: []MenuItem{
			{
				Label: "sync",
				Hint:  "sincronizar historial completo",
				AsyncAction: func() Result {
					client2, err := memory.NewClient(ctx.MemoryDSN(), ctx.MemoryAPIKey())
					if err != nil {
						return Result{Lines: []string{"  ❌ " + err.Error()}}
					}
					defer client2.Close()

					res, err := memory.Sync(client2, "HEAD", ctx.AgentSend)
					if err != nil {
						return Result{Lines: []string{"  ❌ Sync falló: " + err.Error()}}
					}
					out := []string{
						fmt.Sprintf("  ✓ Sync completo: %d commits procesados", res.Total),
						fmt.Sprintf("    insertados:  %d", res.Inserted),
						fmt.Sprintf("    skipped:     %d", res.Skipped),
					}
					if res.Enriched > 0 {
						out = append(out, fmt.Sprintf("    enrichment:  %d enriquecidos via LLM", res.Enriched))
					}
					return Result{Lines: out}
				},
			},
			{
				Label:  "disable",
				Action: func() string { ctx.SetMemory(false); return "deshabilitada" },
			},
		},
	}
}

// doMemorySetup runs the interactive wizard (prompts for DSN + API key).
// Called from MenuItem.Action — same pattern as SetupProvider.
func doMemorySetup(ctx *Context) string {
	host := ctx.PromptInput("  Host Oracle:", "localhost:1521/FREEPDB1", false)
	if host == "" {
		return "cancelled"
	}
	user := ctx.PromptInput("  Usuario:", "hoa", false)
	if user == "" {
		return "cancelled"
	}
	pass := ctx.PromptInput("  Password:", "••••", true)
	if pass == "" {
		return "cancelled"
	}

	dsn := "oracle://" + user + ":" + pass + "@" + host
	ctx.SetMemoryDSN(dsn)

	// Test connection
	client, err := memory.ConnectDSN(dsn)
	if err != nil {
		return "✗ No se pudo conectar: " + err.Error()
	}
	defer client.Close()

	apiKey := ctx.PromptInput("  API Key (vacío = crear proyecto nuevo):", "", false)

	if apiKey == "" {
		// Create project automatically
		name := memory.RepoName()
		newKey, err := client.CreateProject(name)
		if err != nil {
			return "✗ Error creando proyecto: " + err.Error()
		}
		apiKey = newKey
	} else {
		if err := client.ResolveProject(apiKey); err != nil {
			return "✗ " + err.Error()
		}
	}

	ctx.SetMemoryKey(apiKey)
	ctx.SetMemory(true)

	return fmt.Sprintf("✓ conectada · proyecto: %s", apiKey)
}

func memorySetup(ctx *Context) Result {
	result := doMemorySetup(ctx)
	return Result{Lines: []string{"  🧠 " + result}}
}

func memoryDisable(ctx *Context) Result {
	if ctx.SetMemory != nil {
		ctx.SetMemory(false)
	}
	return Result{Lines: []string{"  🧠 memoria: deshabilitada"}}
}

func memorySync(ctx *Context) Result {
	if ctx.MemoryEnabled == nil || !ctx.MemoryEnabled() || ctx.MemoryDSN() == "" {
		return Result{Lines: []string{"  ❌ Memoria no configurada. Usa /memory enable primero."}}
	}

	dsn := ctx.MemoryDSN()
	apiKey := ctx.MemoryAPIKey()
	llm := ctx.AgentSend

	return Result{
		Lines: []string{"  🧠 Sincronizando historial con Oracle..."},
		AsyncFn: func() Result {
			client, err := memory.NewClient(dsn, apiKey)
			if err != nil {
				return Result{Lines: []string{"  ❌ " + err.Error()}}
			}
			defer client.Close()

			res, err := memory.Sync(client, "HEAD", llm)
			if err != nil {
				return Result{Lines: []string{"  ❌ Sync falló: " + err.Error()}}
			}

			lines := []string{
				fmt.Sprintf("  ✓ Sync completo: %d commits procesados", res.Total),
				fmt.Sprintf("    insertados:  %d", res.Inserted),
				fmt.Sprintf("    skipped:     %d", res.Skipped),
			}
			if res.Enriched > 0 {
				lines = append(lines, fmt.Sprintf("    enrichment:  %d enriquecidos via LLM", res.Enriched))
			}
			return Result{Lines: lines}
		},
	}
}
