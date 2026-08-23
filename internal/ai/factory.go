package ai

// ProviderKind selects which adapter to construct.
type ProviderKind string

const (
	// ProviderStub selects the heuristic stub adapter (default).
	ProviderStub ProviderKind = "stub"
	// ProviderInternal selects the internal provider adapter (placeholder).
	ProviderInternal ProviderKind = "internal"
)

// New constructs the adapter selected by kind. Unknown kinds fall back to the
// stub so a bad config value never takes the service down.
func New(kind ProviderKind) Inferrer {
	switch kind {
	case ProviderInternal:
		return NewProvider()
	default:
		return NewStub()
	}
}
