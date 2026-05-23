package command

func cmdClear(ctx *Context, _ string) Result {
	ctx.ClearHist()
	return Result{Lines: []string{"  Historial limpiado."}}
}
