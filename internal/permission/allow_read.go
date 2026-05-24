package permission

import "context"

// AllowRead auto-allows read_file and glob, asks for everything else.
type AllowRead struct{}

func (AllowRead) Decide(_ context.Context, name, _ string) (Decision, string) {
	switch name {
	case "read_file", "glob":
		return DecisionAllow, ""
	}
	return DecisionAsk, ""
}
