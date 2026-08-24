// Package crmmem provides an in-memory adapter for the CRMRepository seam,
// for tests. It reproduces the contractual error modes (not-found returns
// nil,nil; not-in-org is the service's job via OrgID checks) but not SQL
// semantics.
package crmmem

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/patiHash1/Strata-prototype/internal/services"
)

// Repo is an in-memory services.CRMRepository.
type Repo struct {
	mu        sync.Mutex
	contacts  map[uuid.UUID]*services.CRMContact
	deals     map[uuid.UUID]*services.CRMDeal
	quotes    map[uuid.UUID]*services.CRMQuote
	tickets   map[uuid.UUID]*services.CRMHelpdeskTicket
	visits    map[uuid.UUID]*services.FieldSalesVisit
	campaigns map[uuid.UUID]*services.CRMCampaign

	// FailNext makes the next write return this error (error-mode testing).
	FailNext error
}

// New returns an empty in-memory CRM repository.
func New() *Repo {
	return &Repo{
		contacts:  map[uuid.UUID]*services.CRMContact{},
		deals:     map[uuid.UUID]*services.CRMDeal{},
		quotes:    map[uuid.UUID]*services.CRMQuote{},
		tickets:   map[uuid.UUID]*services.CRMHelpdeskTicket{},
		visits:    map[uuid.UUID]*services.FieldSalesVisit{},
		campaigns: map[uuid.UUID]*services.CRMCampaign{},
	}
}

// SeedQuote inserts a quote directly, bypassing the service.
func (r *Repo) SeedQuote(q *services.CRMQuote) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.quotes[q.ID] = q
}

// SeedCampaign inserts a campaign directly, bypassing the service.
func (r *Repo) SeedCampaign(c *services.CRMCampaign) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.campaigns[c.ID] = c
}

// SeedContact inserts a contact directly, bypassing the service.
func (r *Repo) SeedContact(c *services.CRMContact) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.contacts[c.ID] = c
}

// AllDeals returns all deals currently in the repo (for test assertions).
func (r *Repo) AllDeals() []services.CRMDeal {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]services.CRMDeal, 0, len(r.deals))
	for _, d := range r.deals {
		out = append(out, *d)
	}
	return out
}

// AllVisits returns all field visits currently in the repo.
func (r *Repo) AllVisits() []services.FieldSalesVisit {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]services.FieldSalesVisit, 0, len(r.visits))
	for _, v := range r.visits {
		out = append(out, *v)
	}
	return out
}

// AllCampaigns returns all campaigns currently in the repo.
func (r *Repo) AllCampaigns() []services.CRMCampaign {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]services.CRMCampaign, 0, len(r.campaigns))
	for _, c := range r.campaigns {
		out = append(out, *c)
	}
	return out
}

func (r *Repo) checkFail() error {
	if r.FailNext != nil {
		err := r.FailNext
		r.FailNext = nil
		return err
	}
	return nil
}

func (r *Repo) CreateContact(_ context.Context, c *services.CRMContact) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.checkFail(); err != nil {
		return err
	}
	// Mutate pointer to match pgx repo's behaviour (assigns ID + CreatedAt
	// before the INSERT), so the caller sees the populated values.
	c.ID = uuid.New()
	c.CreatedAt = time.Now()
	cp := *c
	r.contacts[c.ID] = &cp
	return nil
}

func (r *Repo) CreateDeal(_ context.Context, d *services.CRMDeal) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.checkFail(); err != nil {
		return err
	}
	d.ID = uuid.New()
	d.CreatedAt = time.Now()
	cp := *d
	r.deals[d.ID] = &cp
	return nil
}

func (r *Repo) GetQuoteByID(_ context.Context, id uuid.UUID) (*services.CRMQuote, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	q, ok := r.quotes[id]
	if !ok {
		return nil, nil // matches pgx.ErrNoRows → nil, nil contract
	}
	cp := *q
	return &cp, nil
}

func (r *Repo) UpdateQuoteRisk(_ context.Context, id uuid.UUID, riskScore float64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.checkFail(); err != nil {
		return err
	}
	q, ok := r.quotes[id]
	if !ok {
		return nil // UPDATE on missing row is a no-op in Postgres too
	}
	q.AIRiskScore = &riskScore
	return nil
}

func (r *Repo) GetContactByID(_ context.Context, id uuid.UUID) (*services.CRMContact, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.contacts[id]
	if !ok {
		return nil, nil
	}
	cp := *c
	return &cp, nil
}

func (r *Repo) CreateTicket(_ context.Context, t *services.CRMHelpdeskTicket) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.checkFail(); err != nil {
		return err
	}
	t.ID = uuid.New()
	t.CreatedAt = time.Now()
	if t.Priority == "" {
		t.Priority = "medium"
	}
	if t.Status == "" {
		t.Status = "open"
	}
	cp := *t
	r.tickets[t.ID] = &cp
	return nil
}

func (r *Repo) CreateFieldSalesVisit(_ context.Context, v *services.FieldSalesVisit) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.checkFail(); err != nil {
		return err
	}
	v.ID = uuid.New()
	v.CreatedAt = time.Now()
	if v.Status == "" {
		v.Status = "scheduled"
	}
	cp := *v
	r.visits[v.ID] = &cp
	return nil
}

func (r *Repo) CreateCampaign(_ context.Context, c *services.CRMCampaign) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.checkFail(); err != nil {
		return err
	}
	c.ID = uuid.New()
	c.CreatedAt = time.Now()
	if c.Status == "" {
		c.Status = "draft"
	}
	cp := *c
	r.campaigns[c.ID] = &cp
	return nil
}

func (r *Repo) GetCampaignByID(_ context.Context, id uuid.UUID) (*services.CRMCampaign, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.campaigns[id]
	if !ok {
		return nil, nil
	}
	cp := *c
	return &cp, nil
}

func (r *Repo) UpdateCampaignStatus(_ context.Context, id uuid.UUID, status string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.checkFail(); err != nil {
		return err
	}
	c, ok := r.campaigns[id]
	if !ok {
		return nil
	}
	c.Status = status
	return nil
}
