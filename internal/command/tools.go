package command

func cmdTools(ctx *Context, _ string) Result {
	names := ctx.ToolNames()
	lines := make([]string, len(names))
	for i, n := range names {
		lines[i] = "  • " + n
	}
	return Result{Lines: lines}
}
