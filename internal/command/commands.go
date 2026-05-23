package command

import (
	"fmt"

	"github.com/cloudcentinel/hoa/internal/cost"
)

func cmdHelp(_ *Context, _ string) Result {
	return Result{Lines: []string{
		"  /model       — Selecciona modelo (menú interactivo)",
		"  /provider    — Cambia provider (menú interactivo)",
		"  /tokens      — Muestra tokens acumulados",
		"  /memory      — Gestiona memoria persistente",
		"  /commit      — Commit interactivo (Conventional Commits)",
		"  /tools       — Lista herramientas disponibles",
		"  /clear       — Limpia historial de conversación",
		"  /exit        — Salir",
	}}
}

func cmdModel(ctx *Context, _ string) Result {
	current := ctx.GetModel()
	plan := ctx.GetPlanModel()
	models := ctx.GetModels()
	provider := ctx.GetProvider()

	items := make([]MenuItem, 0, len(models)+3)

	for i, m := range models {
		model := m
		hint := ""
		if model == current && model == plan {
			hint = "✔ base + planning"
		} else if model == current {
			hint = "✔ base"
		} else if model == plan {
			hint = "✔ planning"
		}
		items = append(items, MenuItem{
			Label: fmt.Sprintf("%d. %s", i+1, model),
			Hint:  hint,
			Action: func() {
				ctx.SetModel(model)
			},
		})
	}

	// Separator + planning section
	items = append(items, MenuItem{Label: "─── planning ───"})

	for i, m := range models {
		model := m
		hint := ""
		if model == plan {
			hint = "✔"
		}
		items = append(items, MenuItem{
			Label: fmt.Sprintf("%d. %s (planning)", i+1, model),
			Hint:  hint,
			Action: func() {
				ctx.SetPlanModel(model)
			},
		})
	}

	return Result{
		Title: fmt.Sprintf("  Modelo · %s · base: %s · plan: %s", provider, current, plan),
		Menu:  items,
	}
}

func cmdProvider(ctx *Context, _ string) Result {
	current := ctx.GetProvider()
	providers := ctx.GetProviders()

	items := make([]MenuItem, 0, len(providers))
	for _, p := range providers {
		prov := p
		hint := ""
		if prov == current {
			hint = "✔ activo"
		}
		items = append(items, MenuItem{
			Label: prov,
			Hint:  hint,
			Action: func() {
				ctx.SetProvider(prov)
			},
		})
	}

	return Result{
		Title: "  Provider",
		Menu:  items,
	}
}

func cmdTokens(ctx *Context, _ string) Result {
	in, out := ctx.TokensUsed()
	model := ctx.GetModel()
	usd := cost.EstimateForModel(model, in, out)
	return Result{Lines: []string{
		fmt.Sprintf("  tokens: %d in · %d out · %d total", in, out, in+out),
		fmt.Sprintf("  costo:  %s (estimado)", cost.FormatCost(usd)),
		fmt.Sprintf("  modelo: %s", model),
	}}
}

func cmdClear(ctx *Context, _ string) Result {
	ctx.ClearHist()
	return Result{Lines: []string{"  Historial limpiado."}}
}

func cmdTools(ctx *Context, _ string) Result {
	names := ctx.ToolNames()
	lines := make([]string, len(names))
	for i, n := range names {
		lines[i] = "  • " + n
	}
	return Result{Lines: lines}
}

func cmdExit(_ *Context, _ string) Result {
	return Result{Quit: true}
}
