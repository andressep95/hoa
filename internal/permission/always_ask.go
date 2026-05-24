package permission

import "context"

// AlwaysAsk asks for confirmation on every tool call.
type AlwaysAsk struct{}

func (AlwaysAsk) Decide(_ context.Context, _, _ string) (Decision, string) {
	return DecisionAsk, ""
}
