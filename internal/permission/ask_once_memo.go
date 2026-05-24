package permission

import (
	"context"
	"sync"
)

// AskOnceMemo asks every tool by default, but remembers names approved with
// "always" during the session so subsequent calls auto-approve.
type AskOnceMemo struct {
	mu      sync.Mutex
	allowed map[string]bool
}

// NewAskOnceMemo returns an initialized policy.
func NewAskOnceMemo() *AskOnceMemo {
	return &AskOnceMemo{allowed: map[string]bool{}}
}

// Decide checks the memo; returns Allow if remembered, otherwise Ask.
func (p *AskOnceMemo) Decide(_ context.Context, name, _ string) (Decision, string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.allowed[name] {
		return DecisionAllow, ""
	}
	return DecisionAsk, ""
}

// Remember records that the user pressed "always" for this tool name.
func (p *AskOnceMemo) Remember(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.allowed == nil {
		p.allowed = map[string]bool{}
	}
	p.allowed[name] = true
}

// Forget clears the memo (handy for /clear or session reset).
func (p *AskOnceMemo) Forget() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.allowed = map[string]bool{}
}

// Remembered returns the set of currently-remembered tool names (sorted not
// guaranteed). Used by /status.
func (p *AskOnceMemo) Remembered() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, 0, len(p.allowed))
	for k := range p.allowed {
		out = append(out, k)
	}
	return out
}
