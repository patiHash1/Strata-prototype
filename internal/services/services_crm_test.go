package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/patiHash1/Strata-prototype/internal/ai"
	"github.com/patiHash1/Strata-prototype/internal/services"
	"github.com/patiHash1/Strata-prototype/internal/services/crmmem"
)

// scriptedInferrer is a test adapter for ai.Inferrer returning canned
// responses keyed by request type.
type scriptedInferrer struct {
	contractRisk *ai.ContractRiskResponse
	sentiment    *ai.SentimentResponse
	segment      *ai.CampaignSegmentResponse
	reach        *ai.CampaignReachResponse
}

func (s *scriptedInferrer) Infer(_ context.Context, req ai.Request) (ai.Response, error) {
	switch r := req.(type) {
	case *ai.ContractRiskRequest:
		if s.contractRisk == nil {
			return nil, errors.New("unexpected ContractRiskRequest")
		}
		_ = r
		return s.contractRisk, nil
	case *ai.SentimentRequest:
		if s.sentiment == nil {
			return nil, errors.New("unexpected SentimentRequest")
		}
		return s.sentiment, nil
	case *ai.CampaignSegmentRequest:
		if s.segment == nil {
			return nil, errors.New("unexpected CampaignSegmentRequest")
		}
		return s.segment, nil
	case *ai.CampaignReachRequest:
		if s.reach == nil {
			return nil, errors.New("unexpected CampaignReachRequest")
		}
		return s.reach, nil
	default:
		return nil, errors.New("unexpected request type")
	}
}

func newTestCRM(t *testing.T, inf *scriptedInferrer, opts ...services.CRMOption) (*services.CRMService, *crmmem.Repo) {
	t.Helper()
	repo := crmmem.New()
	if inf == nil {
		inf = &scriptedInferrer{} // nil-safe; tests that don't hit AI paths won't need fields
	}
	opts = append(opts, services.WithCRMRepo(repo))
	svc := services.NewCRMService(nil, inf, opts...)
	return svc, repo
}

// newInf builds a *scriptedInferrer from field values, convenience for tests
// that only care about one capability.
func newInf(fields ...any) *scriptedInferrer {
	inf := &scriptedInferrer{}
	for _, f := range fields {
		switch v := f.(type) {
		case *ai.ContractRiskResponse:
			inf.contractRisk = v
		case *ai.SentimentResponse:
			inf.sentiment = v
		case *ai.CampaignSegmentResponse:
			inf.segment = v
		case *ai.CampaignReachResponse:
			inf.reach = v
		}
	}
	return inf
}

func TestCreateLeadPinsWinProbability(t *testing.T) {
	inf := &scriptedInferrer{}
	repo := crmmem.New()
	svc := services.NewCRMService(nil, inf,
		services.WithCRMRepo(repo),
		services.WithWinProbability(func(*float64) int { return 73 }),
	)

	orgID := uuid.New()
	contact, assignedTo, winProb, err := svc.CreateLead(context.Background(), orgID, "Ada", nil, "ada@example.com", nil, nil)
	if err != nil {
		t.Fatalf("CreateLead: %v", err)
	}
	if contact.ID == uuid.Nil || assignedTo == uuid.Nil {
		t.Fatal("expected contact ID and assignee to be populated")
	}
	if winProb != 73 {
		t.Errorf("win probability = %d, want 73", winProb)
	}
}

func TestAnalyzeContractRiskHappyPathAndErrorModes(t *testing.T) {
	orgID := uuid.New()
	quoteID := uuid.New()

	inf := newInf(&ai.ContractRiskResponse{
		RiskScore: 0.75,
		Clauses:   []ai.FlaggedClause{{Clause: "indemnification", RiskLevel: "high", SuggestedFix: "cap it"}},
	})
	svc, repo := newTestCRM(t, inf)

	// Quote not found.
	if _, _, err := svc.AnalyzeContractRisk(context.Background(), orgID, quoteID, "text"); !errors.Is(err, services.ErrQuoteNotFound) {
		t.Errorf("missing quote: got %v, want ErrQuoteNotFound", err)
	}

	// Quote in another org.
	repo.SeedQuote(&services.CRMQuote{ID: quoteID, OrgID: uuid.New()})
	if _, _, err := svc.AnalyzeContractRisk(context.Background(), orgID, quoteID, "text"); !errors.Is(err, services.ErrQuoteNotInOrg) {
		t.Errorf("foreign quote: got %v, want ErrQuoteNotInOrg", err)
	}

	// Happy path persists the risk score and returns flagged clauses.
	repo.SeedQuote(&services.CRMQuote{ID: quoteID, OrgID: orgID})
	score, clauses, err := svc.AnalyzeContractRisk(context.Background(), orgID, quoteID, "indemnification clause")
	if err != nil {
		t.Fatalf("AnalyzeContractRisk: %v", err)
	}
	if score != 0.75 || len(clauses) != 1 || clauses[0].RiskLevel != "high" {
		t.Errorf("got score=%f clauses=%+v, want 0.75 and one high clause", score, clauses)
	}
	stored, _ := repo.GetQuoteByID(context.Background(), quoteID)
	if stored.AIRiskScore == nil || *stored.AIRiskScore != 0.75 {
		t.Errorf("risk score not persisted: %+v", stored.AIRiskScore)
	}
}

func TestLaunchCampaignErrorModesAndReach(t *testing.T) {
	orgID := uuid.New()
	campaignID := uuid.New()

	inf := newInf(&ai.CampaignReachResponse{EstimatedReach: 4200}, &ai.CampaignSegmentResponse{SegmentCriteria: "criteria"})
	svc, repo := newTestCRM(t, inf)

	if _, _, err := svc.LaunchCampaign(context.Background(), orgID, campaignID); !errors.Is(err, services.ErrCampaignNotFound) {
		t.Errorf("missing campaign: got %v, want ErrCampaignNotFound", err)
	}

	if _, _, err := svc.CreateCampaign(context.Background(), orgID, "spring", "email", nil); err != nil {
		t.Fatalf("CreateCampaign: %v", err)
	}

	// Create a campaign directly in the repo for a known ID.
	repo.SeedCampaign(&services.CRMCampaign{ID: campaignID, OrgID: orgID, Name: "q3", Channel: "email", Status: "draft"})

	foreign := uuid.New()
	if _, _, err := svc.LaunchCampaign(context.Background(), foreign, campaignID); !errors.Is(err, services.ErrCampaignNotInOrg) {
		t.Errorf("foreign campaign: got %v, want ErrCampaignNotInOrg", err)
	}

	campaign, reach, err := svc.LaunchCampaign(context.Background(), orgID, campaignID)
	if err != nil {
		t.Fatalf("LaunchCampaign: %v", err)
	}
	if campaign.Status != "active" {
		t.Errorf("status = %q, want active", campaign.Status)
	}
	if reach != 4200 {
		t.Errorf("reach = %d, want 4200", reach)
	}
}

func TestRepoFailureSurfacesThroughInterface(t *testing.T) {
	boom := errors.New("boom")
	svc, repo := newTestCRM(t, nil)
	repo.FailNext = boom

	if _, _, _, err := svc.CreateLead(context.Background(), uuid.New(), "A", nil, "a@b.c", nil, nil); !errors.Is(err, boom) {
		t.Errorf("repo failure not surfaced: %v", err)
	}
}

// ---------- Full coverage ----------

func TestCreateLeadDealTitleVariants(t *testing.T) {
	repo := crmmem.New()
	inf := &scriptedInferrer{}
	svc := services.NewCRMService(nil, inf, services.WithCRMRepo(repo))
	orgID := uuid.New()
	amt := 5000.0

	// With last name.
	contact, _, winProb, err := svc.CreateLead(context.Background(), orgID, "Grace", strptr("Hopper"), "g@navy.mil", nil, &amt)
	if err != nil {
		t.Fatal(err)
	}
	if winProb < 40 || winProb > 80 {
		t.Errorf("win probability out of expected range: %d", winProb)
	}
	// Deal should exist in repo with the right amount.
	deals := repo.AllDeals()
	if len(deals) != 1 || deals[0].Amount != 5000 || deals[0].Stage != "lead" {
		t.Errorf("unexpected deal: %+v", deals)
	}
	_ = contact // contact created with non-nil ID

	// Without last name, nil estimatedDealSize.
	contact2, _, _, err := svc.CreateLead(context.Background(), orgID, "Alan", nil, "alan@turing.uk", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if contact2.ID == uuid.Nil {
		t.Error("expected contact ID")
	}
}

func TestCreateTicketHappyPath(t *testing.T) {
	sentiment := &ai.SentimentResponse{Score: 0.9, SuggestedResponse: "We're on it.", Priority: "high"}
	svc, repo := newTestCRM(t, newInf(sentiment))
	orgID := uuid.New()
	contactID := uuid.New()
	repo.SeedContact(&services.CRMContact{ID: contactID, OrgID: orgID, FirstName: "Linus"})

	ticket, err := svc.CreateTicket(context.Background(), orgID, contactID, "kernel panic", "boot loops")
	if err != nil {
		t.Fatal(err)
	}
	if ticket.Priority != "high" {
		t.Errorf("priority = %q, want high", ticket.Priority)
	}
	if ticket.AISentimentScore == nil || *ticket.AISentimentScore != 0.9 {
		t.Errorf("sentiment score = %v", ticket.AISentimentScore)
	}
}

func TestCreateTicketContactNotFound(t *testing.T) {
	svc, _ := newTestCRM(t, &scriptedInferrer{sentiment: &ai.SentimentResponse{}})
	if _, err := svc.CreateTicket(context.Background(), uuid.New(), uuid.New(), "x", "y"); !errors.Is(err, services.ErrContactNotFound) {
		t.Errorf("got %v, want ErrContactNotFound", err)
	}
}

func TestCreateTicketContactNotInOrg(t *testing.T) {
	repo := crmmem.New()
	contactID := uuid.New()
	foreignOrg := uuid.New()
	repo.SeedContact(&services.CRMContact{ID: contactID, OrgID: foreignOrg})
	svc := services.NewCRMService(nil, &scriptedInferrer{sentiment: &ai.SentimentResponse{}}, services.WithCRMRepo(repo))
	// Use a different orgID than the seeded contact.
	callerOrg := uuid.New()
	if _, err := svc.CreateTicket(context.Background(), callerOrg, contactID, "x", "y"); !errors.Is(err, services.ErrContactNotInOrg) {
		t.Errorf("got %v, want ErrContactNotInOrg", err)
	}
}

func TestCreateTicketRepoFailure(t *testing.T) {
	repo := crmmem.New()
	contactID := uuid.New()
	repo.SeedContact(&services.CRMContact{ID: contactID, OrgID: uuid.New()})
	svc, _ := newTestCRM(t, &scriptedInferrer{sentiment: &ai.SentimentResponse{}}, services.WithCRMRepo(repo))
	repo.FailNext = errors.New("disk full")
	if _, err := svc.CreateTicket(context.Background(), uuid.New(), contactID, "x", "y"); err == nil {
		t.Error("expected error")
	}
}

func TestScheduleFieldVisitHappyPath(t *testing.T) {
	svc, repo := newTestCRM(t, nil)
	orgID := uuid.New()
	loc := 51.5
	visit, travel, err := svc.ScheduleFieldVisit(context.Background(), orgID, nil, nil, time.Now(), &loc, nil, strptr("on site"))
	if err != nil {
		t.Fatal(err)
	}
	if visit.ID == uuid.Nil {
		t.Error("expected visit ID")
	}
	if travel < 15 || travel > 60 {
		t.Errorf("travel time = %d, want 15–60", travel)
	}
	if len(repo.AllVisits()) != 1 {
		t.Errorf("expected 1 visit in repo, got %d", len(repo.AllVisits()))
	}
}

func TestScheduleFieldVisitRepoFailure(t *testing.T) {
	svc, repo := newTestCRM(t, nil)
	repo.FailNext = errors.New("boom")
	if _, _, err := svc.ScheduleFieldVisit(context.Background(), uuid.New(), nil, nil, time.Now(), nil, nil, nil); err == nil {
		t.Error("expected error")
	}
}

func TestCreateCampaignPopulatesSegmentCriteria(t *testing.T) {
	inf := newInf(&ai.CampaignSegmentResponse{SegmentCriteria: "premium_enterprise"})
	svc, repo := newTestCRM(t, inf)
	orgID := uuid.New()
	budget := 10000.0

	camp, criteria, err := svc.CreateCampaign(context.Background(), orgID, "Q3 push", "email", &budget)
	if err != nil {
		t.Fatal(err)
	}
	if camp.ID == uuid.Nil {
		t.Error("expected campaign ID")
	}
	if criteria != "premium_enterprise" {
		t.Errorf("segment criteria = %q, want premium_enterprise", criteria)
	}
	if camp.AITargetSegmentCriteria == nil || *camp.AITargetSegmentCriteria != "premium_enterprise" {
		t.Error("segment criteria not persisted on campaign")
	}
	if camp.Budget == nil || *camp.Budget != 10000 {
		t.Errorf("budget = %v", camp.Budget)
	}
	if len(repo.AllCampaigns()) != 1 {
		t.Errorf("expected 1 campaign, got %d", len(repo.AllCampaigns()))
	}
}

func TestCreateCampaignRepoFailure(t *testing.T) {
	repo := crmmem.New()
	svc := services.NewCRMService(nil, &scriptedInferrer{segment: &ai.CampaignSegmentResponse{SegmentCriteria: "x"}}, services.WithCRMRepo(repo))
	repo.FailNext = errors.New("write failed")
	if _, _, err := svc.CreateCampaign(context.Background(), uuid.New(), "x", "email", nil); err == nil {
		t.Error("expected error")
	}
}

func strptr(s string) *string { return &s }
