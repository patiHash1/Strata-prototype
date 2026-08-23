package ai

import (
	"context"
	"fmt"
)

// Provider is the placeholder adapter for the internal AI provider that is
// coming. It satisfies the Inferrer seam so the seam is real (two adapters),
// but returns ErrInference until the real provider is implemented.
type Provider struct{}

// NewProvider creates a Provider adapter.
func NewProvider() *Provider { return &Provider{} }

// Infer returns ErrInference for every request until the real provider lands.
func (p *Provider) Infer(ctx context.Context, req Request) (Response, error) {
	return nil, fmt.Errorf("%w: internal provider not yet implemented (capability %s)", ErrInference, req.capability())
}
