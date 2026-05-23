// Package cost tracks token usage and estimated costs per model.
package cost

import (
	"fmt"
	"sync"
)

// Rates holds per-million-token pricing.
type Rates struct {
	Input  float64
	Output float64
}

// Known model pricing (USD per million tokens).
var pricing = map[string]Rates{
	// Anthropic
	"claude-opus-4-7":  {Input: 15.00, Output: 75.00},
	"claude-sonnet-4-6": {Input: 3.00, Output: 15.00},
	"claude-haiku-4-5": {Input: 1.00, Output: 5.00},
	// OpenAI
	"gpt-4o":      {Input: 2.50, Output: 10.00},
	"gpt-4o-mini": {Input: 0.15, Output: 0.60},
	"o3":          {Input: 2.50, Output: 10.00},
	"o4-mini":     {Input: 1.10, Output: 4.40},
}

// ModelUsage holds accumulated tokens for one model.
type ModelUsage struct {
	InputTokens  int
	OutputTokens int
}

// Tracker accumulates token usage across the session.
type Tracker struct {
	mu     sync.Mutex
	models map[string]*ModelUsage
}

// New creates a new cost tracker.
func New() *Tracker {
	return &Tracker{models: make(map[string]*ModelUsage)}
}

// Add records token usage for a model.
func (t *Tracker) Add(model string, input, output int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	u, ok := t.models[model]
	if !ok {
		u = &ModelUsage{}
		t.models[model] = u
	}
	u.InputTokens += input
	u.OutputTokens += output
}

// Total returns aggregate input and output tokens.
func (t *Tracker) Total() (int, int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	var in, out int
	for _, u := range t.models {
		in += u.InputTokens
		out += u.OutputTokens
	}
	return in, out
}

// EstimatedCost returns total estimated USD cost.
func (t *Tracker) EstimatedCost() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	var total float64
	for model, u := range t.models {
		if r, ok := pricing[model]; ok {
			total += float64(u.InputTokens) * r.Input / 1_000_000
			total += float64(u.OutputTokens) * r.Output / 1_000_000
		}
	}
	return total
}

// FormatCost formats a USD amount with smart precision.
func FormatCost(usd float64) string {
	if usd >= 0.50 {
		return fmt.Sprintf("$%.2f", usd)
	}
	if usd == 0 {
		return "$0.00"
	}
	return fmt.Sprintf("$%.4f", usd)
}

// EstimateForModel calculates cost for given tokens on a model.
func EstimateForModel(model string, input, output int) float64 {
	r, ok := pricing[model]
	if !ok {
		return 0
	}
	return float64(input)*r.Input/1_000_000 + float64(output)*r.Output/1_000_000
}

// Summary returns a formatted multi-line summary.
func (t *Tracker) Summary() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	in, out := 0, 0
	for _, u := range t.models {
		in += u.InputTokens
		out += u.OutputTokens
	}
	lines := []string{
		fmt.Sprintf("  tokens: %d in · %d out · %d total", in, out, in+out),
		fmt.Sprintf("  costo:  %s (estimado)", FormatCost(t.EstimatedCost())),
	}
	if len(t.models) > 1 {
		lines = append(lines, "")
		for model, u := range t.models {
			r := pricing[model]
			c := float64(u.InputTokens)*r.Input/1_000_000 + float64(u.OutputTokens)*r.Output/1_000_000
			lines = append(lines, fmt.Sprintf("    %s: %d in/%d out %s", model, u.InputTokens, u.OutputTokens, FormatCost(c)))
		}
	}
	return lines
}
