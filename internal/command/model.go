package command

import "fmt"

func cmdModel(ctx *Context, _ string) Result {
	base := ctx.GetModel()
	plan := ctx.GetPlanModel()
	models := ctx.GetModels()
	mode := ctx.GetMode()

	items := make([]MenuItem, 0)

	// Execute section
	items = append(items, MenuItem{Label: "── execute ──"})
	for _, m := range models {
		model := m
		hint := ""
		if model == base {
			hint = "✔"
		}
		items = append(items, MenuItem{
			Label:  "  " + model,
			Hint:   hint,
			Action: func() { ctx.SetModel(model) },
		})
	}

	// Planning section (only if plan+execute mode)
	if mode == "plan+execute" {
		items = append(items, MenuItem{Label: ""})
		items = append(items, MenuItem{Label: "── planning ──"})
		for _, m := range models {
			model := m
			hint := ""
			if model == plan {
				hint = "✔"
			}
			items = append(items, MenuItem{
				Label:  "  " + model,
				Hint:   hint,
				Action: func() { ctx.SetPlanModel(model) },
			})
		}
	}

	title := fmt.Sprintf("  %s · execute: %s", ctx.GetProvider(), base)
	if mode == "plan+execute" {
		title = fmt.Sprintf("  %s · execute: %s · plan: %s", ctx.GetProvider(), base, plan)
	}

	return Result{Title: title, Menu: items}
}
