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

func TestStubDispatchesCampaignSegment(t *testing.T) {
	s := NewStub()
	resp, err := s.Infer(context.Background(), &CampaignSegmentRequest{Channel: "email"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cs, ok := resp.(*CampaignSegmentResponse)
	if !ok {
		t.Fatalf("expected *CampaignSegmentResponse, got %T", resp)
	}
	if cs.SegmentCriteria == "" {
		t.Error("expected segment criteria")
	}
}

func TestStubDispatchesCampaignReach(t *testing.T) {
	s := NewStub()
	resp, err := s.Infer(context.Background(), &CampaignReachRequest{Channel: "social"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cr, ok := resp.(*CampaignReachResponse)
	if !ok {
		t.Fatalf("expected *CampaignReachResponse, got %T", resp)
	}
	if cr.EstimatedReach <= 0 {
		t.Error("expected a positive estimated reach")
	}
}

func TestStubDispatchesBottleneckRisk(t *testing.T) {
	s := NewStub()
	resp, err := s.Infer(context.Background(), &BottleneckRiskRequest{Quantity: 120})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	br, ok := resp.(*BottleneckRiskResponse)
	if !ok {
		t.Fatalf("expected *BottleneckRiskResponse, got %T", resp)
	}
	if br.Rating != "High" {
		t.Errorf("expected High rating for large quantity, got %q", br.Rating)
	}
}

func TestStubDispatchesSupplierRiskRating(t *testing.T) {
	s := NewStub()
	resp, err := s.Infer(context.Background(), &SupplierRiskRatingRequest{SupplierName: "Acme"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sr, ok := resp.(*SupplierRiskRatingResponse)
	if !ok {
		t.Fatalf("expected *SupplierRiskRatingResponse, got %T", resp)
	}
	if sr.Rating == "" {
		t.Error("expected a risk rating")
	}
}

func TestStubDispatchesReadingAnomaly(t *testing.T) {
	s := NewStub()
	resp, err := s.Infer(context.Background(), &ReadingAnomalyRequest{MetricName: "temperature", MetricValue: 90})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rr, ok := resp.(*ReadingAnomalyResponse)
	if !ok {
		t.Fatalf("expected *ReadingAnomalyResponse, got %T", resp)
	}
	if !rr.AnomalyDetected {
		t.Error("expected anomaly detected for high temperature")
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
