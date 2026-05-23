// Package provider defines the LLM-backend interface. The harness only talks
// to providers through this interface — swap implementations to swap models.
package provider

import (
	"context"

	"github.com/cloudcentinel/hoa/internal/api"
)

// Provider is the contract every LLM backend implements.
type Provider interface {
	Send(ctx context.Context, messages []api.Message, tools []api.ToolDef) (api.Response, error)
	Model() string
	SetModel(name string)
	TotalUsage() api.Usage
}
