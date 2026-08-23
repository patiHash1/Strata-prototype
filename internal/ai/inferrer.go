package ai

import (
	"context"
	"errors"
)

// ErrInference indicates that an inference request failed (provider error,
// stub failure, etc.). Services map this to an HTTP 500 at the handler.
var ErrInference = errors.New("ai inference failed")

// ErrUnknownCapability indicates the adapter received a request type it
// cannot dispatch.
var ErrUnknownCapability = errors.New("unknown ai capability")

// Inferrer is the seam behind which all AI inference lives. It has a single,
// maximally-deep method: callers pass a typed Request and receive a typed
// Response, and the adapter (stub or provider) handles dispatch internally.
type Inferrer interface {
	// Infer performs an inference for the given typed request.
	Infer(ctx context.Context, req Request) (Response, error)
}
