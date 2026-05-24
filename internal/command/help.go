package command

func cmdHelp(_ *Context, _ string) Result {
	return Result{Lines: []string{
		"  /mode        — Alterna modo (execute / plan+execute)",
		"  /model       — Selecciona modelo (menu interactivo)",
		"  /provider    — Cambia provider (menu interactivo)",
		"  /tokens      — Muestra tokens acumulados",
		"  /status      — Estado detallado de subsistemas",
		"  /memory      — Gestiona memoria persistente",
		"  /commit      — Commit interactivo (Conventional Commits)",
		"  /tools       — Lista herramientas disponibles",
		"  /clear       — Limpia historial de conversacion",
		"  /exit        — Salir",
	}}
}
