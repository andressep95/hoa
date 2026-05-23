// Package command implements the slash command dispatch system.
package command

import "strings"

// MenuItem represents one option in an interactive menu.
type MenuItem struct {
	Label  string
	Hint   string
	Action func() string // returns feedback text (empty = no feedback)
}

// Result holds the output of a command execution.
type Result struct {
	Lines   []string
	Menu    []MenuItem // if non-nil, TUI shows interactive menu
	Title   string     // menu title
	Quit    bool
	AsyncFn func() Result // if set, TUI runs this in background with spinner
}

// Context provides commands access to the runtime state.
type Context struct {
	GetModel      func() string
	SetModel      func(string)
	GetPlanModel  func() string
	SetPlanModel  func(string)
	GetProvider   func() string
	SetProvider   func(string)
	SetupProvider func(name string) // configure API key for a new provider
	GetModels     func() []string
	GetProviders  func() []string
	GetMode       func() string
	SetMode       func(string)
	TokensUsed    func() (int, int)
	ClearHist     func()
	ToolNames     func() []string
	AgentSend     func(prompt string) (string, error)
}

// Handler is a function that executes a slash command.
type Handler func(ctx *Context, args string) Result

var registry = map[string]Handler{}

// Register adds a command to the registry.
func Register(name string, h Handler) { registry[name] = h }

// Names returns all registered command names.
func Names() []string {
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	return names
}

// Dispatch parses input and runs the matching command.
func Dispatch(ctx *Context, input string) (Result, bool) {
	if !strings.HasPrefix(input, "/") {
		return Result{}, false
	}
	parts := strings.SplitN(input[1:], " ", 2)
	name := strings.ToLower(parts[0])
	args := ""
	if len(parts) > 1 {
		args = strings.TrimSpace(parts[1])
	}

	if h, ok := registry[name]; ok {
		return h(ctx, args), true
	}

	return Result{Lines: []string{
		"  Comando desconocido: /" + name,
		"  Usa /help para ver comandos disponibles.",
	}}, true
}

func init() {
	Register("help", cmdHelp)
	Register("model", cmdModel)
	Register("mode", cmdMode)
	Register("provider", cmdProvider)
	Register("tokens", cmdTokens)
	Register("clear", cmdClear)
	Register("tools", cmdTools)
	Register("exit", cmdExit)
	Register("memory", cmdMemory)
}
