package command

func cmdExit(_ *Context, _ string) Result {
	return Result{Quit: true}
}
