package command

func cmdClear(ctx *Context, _ string) Result {
	ctx.ClearHist()
	return Result{ClearScreen: true}
}
