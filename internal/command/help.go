package command

func cmdHelp(_ *Context, _ string) Result {
	return Result{Lines: []string{
		"  /mode        — Alterna modo (execute / plan+execute)",
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
