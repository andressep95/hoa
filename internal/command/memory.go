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
			Title: "  [mem] Memoria: deshabilitada",
			Lines: []string{
				"",
				"  provider: oracle 23ai",
				"  estado:   ( ) desconectada",
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
			Title: "  [mem] Memoria: [!] sin configurar",
			Lines: []string{
				"",
				"  provider: oracle 23ai",
				"  estado:   [!] falta DSN / API Key",
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
			Title: "  [mem] Memoria: [x] error de conexion",
			Lines: []string{
				"",
				"  provider: oracle 23ai",
				"  estado:   [x] " + err.Error(),
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
		return Result{Lines: []string{"  [x] Error consultando: " + err.Error()}}
	}

	lines := []string{
		"",
		"  provider:    oracle 23ai",
		"  estado:      (*) conectada",
		fmt.Sprintf("  indexados:   %d archivos", indexed),
	}
	if pending > 0 {
		lines = append(lines, fmt.Sprintf("  enrichment:  %d pendientes", pending))
	}
	lines = append(lines, "")

	return Result{
		Title: "  [mem] Memoria: [ok] conectada",
		Lines: lines,
		Menu: []MenuItem{
			{
				Label: "sync",
				Hint:  "sincronizar historial completo",
				AsyncAction: func() Result {
					client2, err := memory.NewClient(ctx.MemoryDSN(), ctx.MemoryAPIKey())
					if err != nil {
						return Result{Lines: []string{"  [x] " + err.Error()}}
					}
					defer client2.Close()

					res, err := memory.Sync(client2, "HEAD", ctx.AgentSend)
					if err != nil {
						return Result{Lines: []string{"  [x] Sync fallo: " + err.Error()}}
					}
					out := []string{
						fmt.Sprintf("  [ok] Sync completo: %d commits procesados", res.Total),
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
		return "[x] No se pudo conectar: " + err.Error()
	}
	defer client.Close()

	apiKey := ctx.PromptInput("  API Key (vacio = crear proyecto nuevo):", "", false)

	if apiKey == "" {
		// Create project automatically
		name := memory.RepoName()
		newKey, err := client.CreateProject(name)
		if err != nil {
			return "[x] Error creando proyecto: " + err.Error()
		}
		apiKey = newKey
	} else {
		if err := client.ResolveProject(apiKey); err != nil {
			return "[x] " + err.Error()
		}
	}

	ctx.SetMemoryKey(apiKey)
	ctx.SetMemory(true)

	return fmt.Sprintf("[ok] conectada · proyecto: %s", apiKey)
}

func memorySetup(ctx *Context) Result {
	result := doMemorySetup(ctx)
	return Result{Lines: []string{"  [mem] " + result}}
}

func memoryDisable(ctx *Context) Result {
	if ctx.SetMemory != nil {
		ctx.SetMemory(false)
	}
	return Result{Lines: []string{"  [mem] memoria: deshabilitada"}}
}

func memorySync(ctx *Context) Result {
	if ctx.MemoryEnabled == nil || !ctx.MemoryEnabled() || ctx.MemoryDSN() == "" {
		return Result{Lines: []string{"  [x] Memoria no configurada. Usa /memory enable primero."}}
	}

	dsn := ctx.MemoryDSN()
	apiKey := ctx.MemoryAPIKey()
	llm := ctx.AgentSend

	return Result{
		Lines: []string{"  [mem] Sincronizando historial con Oracle..."},
		AsyncFn: func() Result {
			client, err := memory.NewClient(dsn, apiKey)
			if err != nil {
				return Result{Lines: []string{"  [x] " + err.Error()}}
			}
			defer client.Close()

			res, err := memory.Sync(client, "HEAD", llm)
			if err != nil {
				return Result{Lines: []string{"  [x] Sync fallo: " + err.Error()}}
			}

			lines := []string{
				fmt.Sprintf("  [ok] Sync completo: %d commits procesados", res.Total),
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
