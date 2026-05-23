package command

import "fmt"

func cmdMode(ctx *Context, _ string) Result {
	current := ctx.GetMode()
	items := []MenuItem{
		{
			Label:  "execute",
			Hint:   modeHint("execute", current),
			Action: func() string { ctx.SetMode("execute"); return "" },
		},
		{
			Label:  "plan+execute",
			Hint:   modeHint("plan+execute", current),
			Action: func() string { ctx.SetMode("plan+execute"); return "" },
		},
	}
	return Result{
		Title: fmt.Sprintf("  ⚡ Modo actual: %s", current),
		Menu:  items,
	}
}

func modeHint(mode, current string) string {
	if mode == current {
		return "✔ activo"
	}
	return ""
}
