package command

import (
	"fmt"

	"github.com/cloudcentinel/hoa/internal/cost"
)

func cmdTokens(ctx *Context, _ string) Result {
	in, out := ctx.TokensUsed()
	model := ctx.GetModel()
	usd := cost.EstimateForModel(model, in, out)
	return Result{Lines: []string{
		fmt.Sprintf("  tokens: %d in · %d out · %d total", in, out, in+out),
		fmt.Sprintf("  costo:  %s (estimado)", cost.FormatCost(usd)),
		fmt.Sprintf("  modelo: %s", model),
	}}
}
