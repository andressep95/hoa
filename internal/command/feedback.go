package command

import (
	"fmt"

	"github.com/cloudcentinel/hoa/internal/memory"
)

func init() {
	Register("feedback", cmdFeedback)
}

func cmdFeedback(ctx *Context, args string) Result {
	if ctx.MemoryEnabled == nil || !ctx.MemoryEnabled() || ctx.MemoryDSN() == "" {
		return Result{Lines: []string{"  [x] Memoria no configurada. Usa /memory enable primero."}}
	}

	if args == "" {
		return Result{Lines: []string{
			"  /feedback <regla>           — Guardar una regla de feedback",
			"  /feedback list              — Ver reglas activas",
			"",
			"  Ejemplo: /feedback No usar mocks en tests de integracion porque divergen de prod",
		}}
	}

	if args == "list" {
		return feedbackList(ctx)
	}

	// Save new feedback rule
	return Result{
		Lines: []string{"  Guardando feedback..."},
		AsyncFn: func() Result {
			client, err := memory.NewClient(ctx.MemoryDSN(), ctx.MemoryAPIKey())
			if err != nil {
				return Result{Lines: []string{"  [x] " + err.Error()}}
			}
			defer client.Close()

			if err := client.SaveFeedback(args, "", ""); err != nil {
				return Result{Lines: []string{"  [x] Error guardando: " + err.Error()}}
			}
			return Result{Lines: []string{"  [ok] Feedback guardado: " + args}}
		},
	}
}

func feedbackList(ctx *Context) Result {
	return Result{
		Lines: []string{"  Cargando feedback..."},
		AsyncFn: func() Result {
			client, err := memory.NewClient(ctx.MemoryDSN(), ctx.MemoryAPIKey())
			if err != nil {
				return Result{Lines: []string{"  [x] " + err.Error()}}
			}
			defer client.Close()

			rules, err := client.SearchFeedback("", 20)
			if err != nil {
				return Result{Lines: []string{"  [x] " + err.Error()}}
			}
			if len(rules) == 0 {
				return Result{Lines: []string{"  No hay feedback guardado."}}
			}

			lines := []string{fmt.Sprintf("  [list] %d regla(s) activas:", len(rules)), ""}
			for _, r := range rules {
				line := "  - " + r.Rule
				if r.Why != "" {
					line += " — " + r.Why
				}
				if r.Scope != "" {
					line += " [" + r.Scope + "]"
				}
				lines = append(lines, line)
			}
			return Result{Lines: lines}
		},
	}
}
