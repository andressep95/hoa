package command

import "fmt"

func cmdProvider(ctx *Context, _ string) Result {
	current := ctx.GetProvider()
	providers := ctx.GetProviders()

	configured := make(map[string]bool)
	for _, p := range providers {
		configured[p] = true
	}

	items := []MenuItem{}

	// Section 1: Switch to configured providers
	for _, p := range providers {
		prov := p
		hint := ""
		if prov == current {
			hint = "✔ activo"
		}
		items = append(items, MenuItem{
			Label:  prov,
			Hint:   hint,
			Action: func() { ctx.SetProvider(prov) },
		})
	}

	// Separator
	items = append(items, MenuItem{Label: "───────────────────"})

	// Section 2: Add new provider
	allProviders := []string{"anthropic", "openai", "ollama", "google"}
	for _, p := range allProviders {
		if !configured[p] {
			prov := p
			items = append(items, MenuItem{
				Label:  "＋ Agregar " + prov,
				Action: func() {
					ctx.SetupProvider(prov)
					ctx.SetProvider(prov)
				},
			})
		}
	}

	// Section 3: Modify existing (change API key)
	items = append(items, MenuItem{Label: "───────────────────"})
	items = append(items, MenuItem{
		Label:  "🔑 Cambiar API key de " + current,
		Action: func() { ctx.SetupProvider(current) },
	})

	return Result{
		Title: fmt.Sprintf("  🔌 Provider actual: %s", current),
		Menu:  items,
	}
}
