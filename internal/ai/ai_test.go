package ai

import (
	"context"
	"errors"
	"testing"
)

func TestStubDispatchesSentiment(t *testing.T) {
	s := NewStub()
	resp, err := s.Infer(context.Background(), &SentimentRequest{
		Subject:     "urgent broken login",
		Description: "cannot access account",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sr, ok := resp.(*SentimentResponse)
	if !ok {
		t.Fatalf("expected *SentimentResponse, got %T", resp)
	}
	if sr.Priority == "" {
		t.Error("expected a derived priority")
	}
	if sr.SuggestedResponse == "" {
		t.Error("expected a suggested response")
	}
}

func TestStubDispatchesContractRisk(t *testing.T) {
	s := NewStub()
	resp, err := s.Infer(context.Background(), &ContractRiskRequest{
		ContractText: "indemnification and penalty clauses",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cr, ok := resp.(*ContractRiskResponse)
	if !ok {
		t.Fatalf("expected *ContractRiskResponse, got %T", resp)
	}
	if len(cr.Clauses) == 0 {
		t.Error("expected flagged clauses for risky contract")
	}
}

func TestStubUnknownCapability(t *testing.T) {
	s := NewStub()
	_, err := s.Infer(context.Background(), &unknownRequest{})
	if !errors.Is(err, ErrUnknownCapability) {
		t.Fatalf("expected ErrUnknownCapability, got %v", err)
	}
}

func TestProviderReturnsNotImplemented(t *testing.T) {
	p := NewProvider()
	_, err := p.Infer(context.Background(), &SentimentRequest{Subject: "x"})
	if !errors.Is(err, ErrInference) {
		t.Fatalf("expected ErrInference, got %v", err)
	}
}

func TestFactorySelectsAdapter(t *testing.T) {
	if _, ok := New(ProviderStub).(*Stub); !ok {
		t.Error("expected stub adapter for ProviderStub")
	}
	if _, ok := New(ProviderInternal).(*Provider); !ok {
		t.Error("expected provider adapter for ProviderInternal")
	}
	if _, ok := New("bogus").(*Stub); !ok {
		t.Error("expected stub fallback for unknown kind")
	}
}

// unknownRequest is a Request type the stub cannot dispatch.
type unknownRequest struct{}

func (*unknownRequest) capability() string { return "test.unknown" }
