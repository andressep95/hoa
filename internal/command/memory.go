package command

func cmdMemory(_ *Context, args string) Result {
	switch args {
	case "status", "":
		// TODO: check actual connection to Oracle when memory client is implemented
		return Result{Lines: []string{
			"  memoria: placeholder (no conectada)",
			"  Usa /memory enable | disable | provider <name>",
		}}
	case "enable":
		// TODO: enable memory and test connection
		return Result{Lines: []string{"  memoria: habilitada (pendiente implementación)"}}
	case "disable":
		// TODO: disable memory
		return Result{Lines: []string{"  memoria: deshabilitada"}}
	default:
		return Result{Lines: []string{
			"  /memory              — Estado actual",
			"  /memory enable       — Activar memoria persistente",
			"  /memory disable      — Desactivar memoria",
			"  /memory provider <x> — Cambiar provider (oracle)",
		}}
	}
}
