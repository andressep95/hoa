package permission

import "context"

type Decision int

const (
	DecisionAsk   Decision = iota // route through Agent.Confirm
	DecisionAllow                 // execute without asking
	DecisionDeny                  // reject; reason shown to model
)

// ConfirmResult is what the user (via the TUI modal) replied to a 3-button
// approval prompt. Y = approve only this call. A = approve every subsequent
// call to the same tool name in the session. N = deny.
type ConfirmResult int

const (
	ResultYes ConfirmResult = iota
	ResultAlways
	ResultNo
)

type Policy interface {
	Decide(ctx context.Context, name, input string) (Decision, string)
}

// Rememberer is implemented by policies that can persist a "always allow" decision
// for a given tool name during the session. The agent loop type-asserts the
// active Policy against this interface after a ResultAlways reply.
type Rememberer interface {
	Remember(name string)
}
