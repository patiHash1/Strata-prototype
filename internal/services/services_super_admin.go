package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// ── Types ──

// MaintenanceRule represents a partitioned maintenance lock.
type MaintenanceRule struct {
	ID           int64     `json:"id"`
	Scope        string    `json:"scope"`
	TargetID     string    `json:"target_id"`
	IsActive     bool      `json:"is_active"`
	Reason       string    `json:"reason"`
	AllowedRoles []string  `json:"allowed_roles"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// SystemError represents a captured panic or system error.
type SystemError struct {
	ID           int64     `json:"id"`
	Module       string    `json:"module"`
	ErrorMessage string    `json:"error_message"`
	StackTrace   string    `json:"stack_trace"`
	StatusCode   int       `json:"status_code"`
	CreatedAt    time.Time `json:"created_at"`
}

// CIHealthReport represents a CI health ingestion payload.
type CIHealthReport struct {
	ID                   int64     `json:"id"`
	Module               string    `json:"module"`
	CoveragePercent      float64   `json:"coverage_percent"`
	LinterIssues         int       `json:"linter_issues"`
	VulnerabilitiesCount int       `json:"vulnerabilities_count"`
	CommitSHA            string    `json:"commit_sha"`
	CreatedAt            time.Time `json:"created_at"`
}

// TelemetrySnapshot holds aggregated runtime and DB metrics.
type TelemetrySnapshot struct {
	Timestamp    time.Time      `json:"timestamp"`
	Runtime      RuntimeMetrics `json:"runtime"`
	DB           DBMetrics      `json:"db"`
	HTTP         HTTPMetrics    `json:"http"`
	RecentPanics []SystemError  `json:"recent_panics"`
}

// RuntimeMetrics holds Go runtime statistics.
type RuntimeMetrics struct {
	AllocatedMB float64 `json:"allocated_mb"`
	GCRuns      uint32  `json:"gc_runs"`
	Goroutines  int     `json:"goroutines"`
	HeapObjects uint64  `json:"heap_objects"`
}

// DBMetrics holds pgxpool connection pool statistics.
type DBMetrics struct {
	AcquiredConns int32 `json:"acquired_conns"`
	IdleConns     int32 `json:"idle_conns"`
	TotalConns    int32 `json:"total_conns"`
	MaxConns      int32 `json:"max_conns"`
}

// HTTPMetrics holds aggregated HTTP request statistics.
type HTTPMetrics struct {
	TotalRequests int64                         `json:"total_requests"`
	Status2xx     int64                         `json:"status_2xx"`
	Status4xx     int64                         `json:"status_4xx"`
	Status5xx     int64                         `json:"status_5xx"`
	LatencyP50    float64                       `json:"latency_p50_ms"`
	LatencyP95    float64                       `json:"latency_p95_ms"`
	LatencyP99    float64                       `json:"latency_p99_ms"`
	PerModule     map[string]*ModuleHTTPMetrics `json:"per_module"`
}

// ModuleHTTPMetrics holds per-module HTTP stats.
type ModuleHTTPMetrics struct {
	Requests       int64   `json:"requests"`
	Errors5xx      int64   `json:"errors_5xx"`
	TotalLatencyMs float64 `json:"total_latency_ms"`
}

// ModuleHealth holds a composite health score for a module.
type ModuleHealth struct {
	Module          string  `json:"module"`
	HealthScore     float64 `json:"health_score"`
	CoveragePercent float64 `json:"coverage_percent"`
	LinterIssues    int     `json:"linter_issues"`
	Vulnerabilities int     `json:"vulnerabilities"`
	ErrorRate5xx    float64 `json:"error_rate_5xx_percent"`
}

// SOCEvent represents a real-time security event.
type SOCEvent struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Severity  string         `json:"severity"`
	Message   string         `json:"message"`
	IPAddress string         `json:"ip_address,omitempty"`
	UserID    string         `json:"user_id,omitempty"`
	OrgID     string         `json:"org_id,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// MaintenanceToggleRequest is the payload for toggling maintenance.
type MaintenanceToggleRequest struct {
	Scope        string   `json:"scope"`
	TargetID     string   `json:"target_id"`
	IsActive     bool     `json:"is_active"`
	Reason       string   `json:"reason"`
	AllowedRoles []string `json:"allowed_roles,omitempty"`
}

// CIHealthIngestRequest is the payload for CI health ingestion.
type CIHealthIngestRequest struct {
	Module               string  `json:"module"`
	CoveragePercent      float64 `json:"coverage_percent"`
	LinterIssues         int     `json:"linter_issues"`
	VulnerabilitiesCount int     `json:"vulnerabilities_count"`
	CommitSHA            string  `json:"commit_sha"`
}

// ── Ring Buffer ──

// RingBuffer is a fixed-size thread-safe ring buffer.
type RingBuffer[T any] struct {
	mu    sync.Mutex
	buf   []T
	size  int
	head  int
	count int
}

// NewRingBuffer creates a ring buffer with the given capacity.
func NewRingBuffer[T any](size int) *RingBuffer[T] {
	return &RingBuffer[T]{
		buf:  make([]T, size),
		size: size,
	}
}

// Push adds an item to the ring buffer.
func (rb *RingBuffer[T]) Push(item T) {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	rb.buf[rb.head] = item
	rb.head = (rb.head + 1) % rb.size
	if rb.count < rb.size {
		rb.count++
	}
}

// Snapshot returns a copy of all items in insertion order.
func (rb *RingBuffer[T]) Snapshot() []T {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	out := make([]T, rb.count)
	for i := 0; i < rb.count; i++ {
		idx := (rb.head - rb.count + i + rb.size) % rb.size
		out[i] = rb.buf[idx]
	}
	return out
}

// ── Repository ──

type superAdminRepository struct {
	pool *pgxpool.Pool
}

func newSuperAdminRepository(pool *pgxpool.Pool) *superAdminRepository {
	return &superAdminRepository{pool: pool}
}

func (r *superAdminRepository) UpsertMaintenanceRule(ctx context.Context, rule *MaintenanceRule) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO super_admin_maintenance_rules (scope, target_id, is_active, reason, allowed_roles, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (scope, target_id)
		DO UPDATE SET is_active = $3, reason = $4, allowed_roles = $5, updated_at = NOW()
	`, rule.Scope, rule.TargetID, rule.IsActive, rule.Reason, rule.AllowedRoles)
	return err
}

func (r *superAdminRepository) ListActiveRules(ctx context.Context) ([]MaintenanceRule, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, scope, target_id, is_active, reason, allowed_roles, created_at, updated_at
		FROM super_admin_maintenance_rules
		WHERE is_active = TRUE
		ORDER BY scope, target_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []MaintenanceRule
	for rows.Next() {
		var rule MaintenanceRule
		if err := rows.Scan(&rule.ID, &rule.Scope, &rule.TargetID, &rule.IsActive,
			&rule.Reason, &rule.AllowedRoles, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (r *superAdminRepository) InsertSystemError(ctx context.Context, errRec *SystemError) error {
	_, dbErr := r.pool.Exec(ctx, `
		INSERT INTO super_admin_system_errors (module, error_message, stack_trace, status_code)
		VALUES ($1, $2, $3, $4)
	`, errRec.Module, errRec.ErrorMessage, errRec.StackTrace, errRec.StatusCode)
	return dbErr
}

func (r *superAdminRepository) InsertCIHealthReport(ctx context.Context, report *CIHealthReport) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO super_admin_ci_health_reports (module, coverage_percent, linter_issues, vulnerabilities_count, commit_sha)
		VALUES ($1, $2, $3, $4, $5)
	`, report.Module, report.CoveragePercent, report.LinterIssues, report.VulnerabilitiesCount, report.CommitSHA)
	return err
}

func (r *superAdminRepository) GetLatestCIHealthByModule(ctx context.Context, module string) (*CIHealthReport, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, module, coverage_percent, linter_issues, vulnerabilities_count, commit_sha, created_at
		FROM super_admin_ci_health_reports
		WHERE module = $1
		ORDER BY created_at DESC
		LIMIT 1
	`, module)

	var report CIHealthReport
	err := row.Scan(&report.ID, &report.Module, &report.CoveragePercent,
		&report.LinterIssues, &report.VulnerabilitiesCount, &report.CommitSHA, &report.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &report, nil
}

func (r *superAdminRepository) GetAllLatestCIHealth(ctx context.Context) ([]CIHealthReport, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT ON (module) id, module, coverage_percent, linter_issues, vulnerabilities_count, commit_sha, created_at
		FROM super_admin_ci_health_reports
		ORDER BY module, created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []CIHealthReport
	for rows.Next() {
		var report CIHealthReport
		if err := rows.Scan(&report.ID, &report.Module, &report.CoveragePercent,
			&report.LinterIssues, &report.VulnerabilitiesCount, &report.CommitSHA, &report.CreatedAt); err != nil {
			return nil, err
		}
		reports = append(reports, report)
	}
	return reports, rows.Err()
}

// ── SSE Subscriber ──

type sseSubscriber struct {
	ch   chan []byte
	done chan struct{}
}

// ── SuperAdminService ──

// SuperAdminService handles system observability, SOC monitoring,
// partitioned maintenance, and CI health ingestion.
type SuperAdminService struct {
	repo *superAdminRepository
	pool *pgxpool.Pool
	rdb  *redis.Client

	// Maintenance cache
	cacheMu    sync.RWMutex
	cacheRules map[string]*MaintenanceRule // key: "scope:target_id"

	// Ring buffers
	panicBuffer     *RingBuffer[SystemError]
	latencyBuffer   *RingBuffer[HTTPLatencyRecord]
	telemetryBuffer *RingBuffer[TelemetrySnapshot]

	// HTTP metrics aggregation
	httpMu      sync.Mutex
	httpMetrics HTTPMetrics

	// SSE subscribers
	sseMu   sync.Mutex
	sseSubs map[string]*sseSubscriber

	// Shutdown
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// HTTPLatencyRecord captures a single HTTP request's latency.
type HTTPLatencyRecord struct {
	Path       string        `json:"path"`
	Method     string        `json:"method"`
	StatusCode int           `json:"status_code"`
	Latency    time.Duration `json:"latency"`
	Module     string        `json:"module"`
	Timestamp  time.Time     `json:"timestamp"`
}

// Redis channel constants.
const (
	RedisChannelMaintenanceSync = "strata:events:maintenance-sync"
	RedisChannelSecuritySOC     = "strata:events:security-soc"
)

// NewSuperAdminService creates a new SuperAdminService.
func NewSuperAdminService(pool *pgxpool.Pool, rdb *redis.Client) *SuperAdminService {
	ctx, cancel := context.WithCancel(context.Background())

	svc := &SuperAdminService{
		repo:            newSuperAdminRepository(pool),
		pool:            pool,
		rdb:             rdb,
		cacheRules:      make(map[string]*MaintenanceRule),
		panicBuffer:     NewRingBuffer[SystemError](100),
		latencyBuffer:   NewRingBuffer[HTTPLatencyRecord](100),
		telemetryBuffer: NewRingBuffer[TelemetrySnapshot](100),
		httpMetrics: HTTPMetrics{
			PerModule: make(map[string]*ModuleHTTPMetrics),
		},
		sseSubs: make(map[string]*sseSubscriber),
		ctx:     ctx,
		cancel:  cancel,
	}

	// Load initial maintenance rules into cache.
	if err := svc.reloadCache(context.Background()); err != nil {
		log.Printf("[super-admin] initial cache load failed: %v", err)
	}

	// Start Redis subscriber for cache invalidation.
	svc.wg.Add(1)
	go svc.subscribeMaintenanceSync()

	// Start Redis subscriber for SOC events → SSE fan-out.
	svc.wg.Add(1)
	go svc.subscribeSOCEvents()

	return svc
}

// Shutdown gracefully stops background goroutines.
func (s *SuperAdminService) Shutdown() {
	s.cancel()
	s.wg.Wait()
}

// ── Maintenance Cache ──

func (s *SuperAdminService) cacheKey(scope, targetID string) string {
	return scope + ":" + targetID
}

func (s *SuperAdminService) reloadCache(ctx context.Context) error {
	rules, err := s.repo.ListActiveRules(ctx)
	if err != nil {
		return err
	}

	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	s.cacheRules = make(map[string]*MaintenanceRule, len(rules))
	for i := range rules {
		rule := rules[i]
		s.cacheRules[s.cacheKey(rule.Scope, rule.TargetID)] = &rule
	}
	return nil
}

// IsUnderMaintenance checks if the given scope+targetID is under maintenance.
// Returns the rule and true if maintenance is active.
func (s *SuperAdminService) IsUnderMaintenance(scope, targetID string) (*MaintenanceRule, bool) {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	rule, ok := s.cacheRules[s.cacheKey(scope, targetID)]
	if !ok {
		return nil, false
	}
	return rule, rule.IsActive
}

// ToggleMaintenance activates or deactivates a maintenance rule and publishes
// a cache-invalidation event to Redis.
func (s *SuperAdminService) ToggleMaintenance(ctx context.Context, req MaintenanceToggleRequest) (*MaintenanceRule, error) {
	rule := &MaintenanceRule{
		Scope:        req.Scope,
		TargetID:     req.TargetID,
		IsActive:     req.IsActive,
		Reason:       req.Reason,
		AllowedRoles: req.AllowedRoles,
	}

	if err := s.repo.UpsertMaintenanceRule(ctx, rule); err != nil {
		return nil, fmt.Errorf("upsert maintenance rule: %w", err)
	}

	// Publish cache invalidation event.
	msg, _ := json.Marshal(map[string]string{
		"action":    "toggle",
		"scope":     req.Scope,
		"target_id": req.TargetID,
	})
	if s.rdb != nil {
		if err := s.rdb.Publish(ctx, RedisChannelMaintenanceSync, msg).Err(); err != nil {
			log.Printf("[super-admin] failed to publish maintenance sync: %v", err)
		}
	}

	// Reload local cache immediately.
	if err := s.reloadCache(ctx); err != nil {
		log.Printf("[super-admin] cache reload after toggle failed: %v", err)
	}

	return rule, nil
}

// ListMaintenanceRules returns all active maintenance rules.
func (s *SuperAdminService) ListMaintenanceRules(ctx context.Context) ([]MaintenanceRule, error) {
	return s.repo.ListActiveRules(ctx)
}

// subscribeMaintenanceSync listens for Redis cache-invalidation messages.
func (s *SuperAdminService) subscribeMaintenanceSync() {
	defer s.wg.Done()

	if s.rdb == nil {
		return
	}

	pubsub := s.rdb.Subscribe(s.ctx, RedisChannelMaintenanceSync)
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case <-s.ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			log.Printf("[super-admin] maintenance sync received: %s", msg.Payload)
			if err := s.reloadCache(context.Background()); err != nil {
				log.Printf("[super-admin] cache reload from sync failed: %v", err)
			}
		}
	}
}

// ── Telemetry ──

// CollectSnapshot gathers runtime and DB metrics into a snapshot.
func (s *SuperAdminService) CollectSnapshot() TelemetrySnapshot {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	snapshot := TelemetrySnapshot{
		Timestamp: time.Now(),
		Runtime: RuntimeMetrics{
			AllocatedMB: float64(mem.Alloc) / 1024 / 1024,
			GCRuns:      mem.NumGC,
			Goroutines:  runtime.NumGoroutine(),
			HeapObjects: mem.HeapObjects,
		},
		RecentPanics: s.panicBuffer.Snapshot(),
	}

	// DB pool stats.
	if s.pool != nil {
		stats := s.pool.Stat()
		snapshot.DB = DBMetrics{
			AcquiredConns: stats.AcquiredConns(),
			IdleConns:     stats.IdleConns(),
			TotalConns:    stats.TotalConns(),
			MaxConns:      stats.MaxConns(),
		}
	}

	// HTTP metrics snapshot.
	s.httpMu.Lock()
	httpCopy := s.httpMetrics
	// Compute latency percentiles from ring buffer.
	latencies := s.latencyBuffer.Snapshot()
	s.httpMu.Unlock()

	if len(latencies) > 0 {
		// Simple percentile computation on sorted latencies.
		sorted := make([]float64, len(latencies))
		for i, l := range latencies {
			sorted[i] = float64(l.Latency.Milliseconds())
		}
		// Insertion sort for small slices (ring buffer max 100).
		for i := 1; i < len(sorted); i++ {
			key := sorted[i]
			j := i - 1
			for j >= 0 && sorted[j] > key {
				sorted[j+1] = sorted[j]
				j--
			}
			sorted[j+1] = key
		}
		n := len(sorted)
		httpCopy.LatencyP50 = sorted[n*50/100]
		httpCopy.LatencyP95 = sorted[n*95/100]
		httpCopy.LatencyP99 = sorted[n*99/100]
	}

	snapshot.HTTP = httpCopy

	s.telemetryBuffer.Push(snapshot)
	return snapshot
}

// RecordPanic stores a panic trace in the ring buffer and persists to DB.
func (s *SuperAdminService) RecordPanic(module string, errMsg string, stackTrace string, statusCode int) {
	errRec := SystemError{
		Module:       module,
		ErrorMessage: errMsg,
		StackTrace:   stackTrace,
		StatusCode:   statusCode,
		CreatedAt:    time.Now(),
	}
	s.panicBuffer.Push(errRec)

	// Async persist to DB.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if dbErr := s.repo.InsertSystemError(ctx, &errRec); dbErr != nil {
			log.Printf("[super-admin] failed to persist panic: %v", dbErr)
		}
	}()
}

// RecordHTTPLatency records an HTTP request's latency for metrics.
func (s *SuperAdminService) RecordHTTPLatency(record HTTPLatencyRecord) {
	s.latencyBuffer.Push(record)

	s.httpMu.Lock()
	defer s.httpMu.Unlock()

	s.httpMetrics.TotalRequests++
	switch {
	case record.StatusCode >= 200 && record.StatusCode < 300:
		s.httpMetrics.Status2xx++
	case record.StatusCode >= 400 && record.StatusCode < 500:
		s.httpMetrics.Status4xx++
	case record.StatusCode >= 500:
		s.httpMetrics.Status5xx++
	}

	// Per-module tracking.
	mod := record.Module
	if mod == "" {
		mod = "unknown"
	}
	if s.httpMetrics.PerModule[mod] == nil {
		s.httpMetrics.PerModule[mod] = &ModuleHTTPMetrics{}
	}
	pm := s.httpMetrics.PerModule[mod]
	pm.Requests++
	pm.TotalLatencyMs += float64(record.Latency.Milliseconds())
	if record.StatusCode >= 500 {
		pm.Errors5xx++
	}
}

// ── SOC Security Events ──

// PublishSOCEvent publishes a security event to Redis and fans out to local SSE subscribers.
func (s *SuperAdminService) PublishSOCEvent(ctx context.Context, event SOCEvent) {
	event.ID = uuid.NewString()
	event.Timestamp = time.Now()

	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("[super-admin] failed to marshal SOC event: %v", err)
		return
	}

	// Publish to Redis for multi-node fan-out.
	if s.rdb != nil {
		if err := s.rdb.Publish(ctx, RedisChannelSecuritySOC, data).Err(); err != nil {
			log.Printf("[super-admin] failed to publish SOC event: %v", err)
		}
	}

	// Fan out to local SSE subscribers.
	s.fanoutSSE(data)
}

// subscribeSOCEvents listens for SOC events from Redis and fans out to local SSE subscribers.
func (s *SuperAdminService) subscribeSOCEvents() {
	defer s.wg.Done()

	if s.rdb == nil {
		return
	}

	pubsub := s.rdb.Subscribe(s.ctx, RedisChannelSecuritySOC)
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case <-s.ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			s.fanoutSSE([]byte(msg.Payload))
		}
	}
}

// ── SSE Subscriber Management ──

// AddSSESubscriber registers a new SSE subscriber and returns its channel.
func (s *SuperAdminService) AddSSESubscriber() (chan []byte, func()) {
	sub := &sseSubscriber{
		ch:   make(chan []byte, 64),
		done: make(chan struct{}),
	}

	id := uuid.NewString()

	s.sseMu.Lock()
	s.sseSubs[id] = sub
	s.sseMu.Unlock()

	cleanup := func() {
		s.sseMu.Lock()
		delete(s.sseSubs, id)
		s.sseMu.Unlock()
		close(sub.done)
	}

	return sub.ch, cleanup
}

func (s *SuperAdminService) fanoutSSE(data []byte) {
	s.sseMu.Lock()
	defer s.sseMu.Unlock()

	for _, sub := range s.sseSubs {
		select {
		case sub.ch <- data:
		default:
			// Subscriber too slow, drop message.
		}
	}
}

// ── CI Health ──

// IngestCIHealth stores a CI health report.
func (s *SuperAdminService) IngestCIHealth(ctx context.Context, req CIHealthIngestRequest) (*CIHealthReport, error) {
	report := &CIHealthReport{
		Module:               req.Module,
		CoveragePercent:      req.CoveragePercent,
		LinterIssues:         req.LinterIssues,
		VulnerabilitiesCount: req.VulnerabilitiesCount,
		CommitSHA:            req.CommitSHA,
	}

	if err := s.repo.InsertCIHealthReport(ctx, report); err != nil {
		return nil, fmt.Errorf("insert CI health report: %w", err)
	}
	return report, nil
}

// GetModuleHealth computes composite health scores for all modules.
func (s *SuperAdminService) GetModuleHealth(ctx context.Context) ([]ModuleHealth, error) {
	reports, err := s.repo.GetAllLatestCIHealth(ctx)
	if err != nil {
		return nil, fmt.Errorf("get CI health reports: %w", err)
	}

	s.httpMu.Lock()
	perModule := make(map[string]*ModuleHTTPMetrics, len(s.httpMetrics.PerModule))
	for k, v := range s.httpMetrics.PerModule {
		perModule[k] = v
	}
	s.httpMu.Unlock()

	// Build a set of known modules from CI reports.
	seen := make(map[string]bool)
	var healths []ModuleHealth

	for _, r := range reports {
		h := ModuleHealth{
			Module:          r.Module,
			CoveragePercent: r.CoveragePercent,
			LinterIssues:    r.LinterIssues,
			Vulnerabilities: r.VulnerabilitiesCount,
		}

		if pm, ok := perModule[r.Module]; ok && pm.Requests > 0 {
			h.ErrorRate5xx = float64(pm.Errors5xx) / float64(pm.Requests) * 100
		}

		// Composite health score:
		// 40% coverage, 30% linter penalty, 20% vulnerability penalty, 10% error rate penalty.
		coverageScore := r.CoveragePercent * 0.4

		linterScore := 30.0
		if r.LinterIssues > 0 {
			linterPenalty := float64(r.LinterIssues) * 0.5
			if linterPenalty > 30 {
				linterPenalty = 30
			}
			linterScore = 30 - linterPenalty
		}

		vulnScore := 20.0
		if r.VulnerabilitiesCount > 0 {
			vulnPenalty := float64(r.VulnerabilitiesCount) * 5
			if vulnPenalty > 20 {
				vulnPenalty = 20
			}
			vulnScore = 20 - vulnPenalty
		}

		errorScore := 10.0
		if h.ErrorRate5xx > 0 {
			errorPenalty := h.ErrorRate5xx * 2
			if errorPenalty > 10 {
				errorPenalty = 10
			}
			errorScore = 10 - errorPenalty
		}

		h.HealthScore = coverageScore + linterScore + vulnScore + errorScore
		healths = append(healths, h)
		seen[r.Module] = true
	}

	// Include modules that have HTTP metrics but no CI reports.
	for mod := range perModule {
		if seen[mod] {
			continue
		}
		h := ModuleHealth{
			Module: mod,
		}
		if perModule[mod].Requests > 0 {
			h.ErrorRate5xx = float64(perModule[mod].Errors5xx) / float64(perModule[mod].Requests) * 100
		}
		healths = append(healths, h)
	}

	return healths, nil
}

// ── Prometheus Format ──

// PrometheusMetrics returns metrics in Prometheus text exposition format.
func (s *SuperAdminService) PrometheusMetrics() string {
	snapshot := s.CollectSnapshot()

	var b strings.Builder

	// Runtime metrics.
	b.WriteString("# HELP strata_runtime_allocated_mb Heap memory allocated in MB\n")
	b.WriteString("# TYPE strata_runtime_allocated_mb gauge\n")
	fmt.Fprintf(&b, "strata_runtime_allocated_mb %.2f\n", snapshot.Runtime.AllocatedMB)

	b.WriteString("# HELP strata_runtime_gc_runs Total GC runs\n")
	b.WriteString("# TYPE strata_runtime_gc_runs counter\n")
	fmt.Fprintf(&b, "strata_runtime_gc_runs %d\n", snapshot.Runtime.GCRuns)

	b.WriteString("# HELP strata_runtime_goroutines Number of goroutines\n")
	b.WriteString("# TYPE strata_runtime_goroutines gauge\n")
	fmt.Fprintf(&b, "strata_runtime_goroutines %d\n", snapshot.Runtime.Goroutines)

	b.WriteString("# HELP strata_runtime_heap_objects Number of heap objects\n")
	b.WriteString("# TYPE strata_runtime_heap_objects gauge\n")
	fmt.Fprintf(&b, "strata_runtime_heap_objects %d\n", snapshot.Runtime.HeapObjects)

	// DB metrics.
	b.WriteString("# HELP strata_db_acquired_conns Acquired connections\n")
	b.WriteString("# TYPE strata_db_acquired_conns gauge\n")
	fmt.Fprintf(&b, "strata_db_acquired_conns %d\n", snapshot.DB.AcquiredConns)

	b.WriteString("# HELP strata_db_idle_conns Idle connections\n")
	b.WriteString("# TYPE strata_db_idle_conns gauge\n")
	fmt.Fprintf(&b, "strata_db_idle_conns %d\n", snapshot.DB.IdleConns)

	b.WriteString("# HELP strata_db_total_conns Total connections\n")
	b.WriteString("# TYPE strata_db_total_conns gauge\n")
	fmt.Fprintf(&b, "strata_db_total_conns %d\n", snapshot.DB.TotalConns)

	// HTTP metrics.
	b.WriteString("# HELP strata_http_requests_total Total HTTP requests\n")
	b.WriteString("# TYPE strata_http_requests_total counter\n")
	fmt.Fprintf(&b, "strata_http_requests_total %d\n", snapshot.HTTP.TotalRequests)

	b.WriteString("# HELP strata_http_requests_2xx HTTP 2xx responses\n")
	b.WriteString("# TYPE strata_http_requests_2xx counter\n")
	fmt.Fprintf(&b, "strata_http_requests_2xx %d\n", snapshot.HTTP.Status2xx)

	b.WriteString("# HELP strata_http_requests_4xx HTTP 4xx responses\n")
	b.WriteString("# TYPE strata_http_requests_4xx counter\n")
	fmt.Fprintf(&b, "strata_http_requests_4xx %d\n", snapshot.HTTP.Status4xx)

	b.WriteString("# HELP strata_http_requests_5xx HTTP 5xx responses\n")
	b.WriteString("# TYPE strata_http_requests_5xx counter\n")
	fmt.Fprintf(&b, "strata_http_requests_5xx %d\n", snapshot.HTTP.Status5xx)

	b.WriteString("# HELP strata_http_latency_p50_ms HTTP latency p50 in ms\n")
	b.WriteString("# TYPE strata_http_latency_p50_ms gauge\n")
	fmt.Fprintf(&b, "strata_http_latency_p50_ms %.2f\n", snapshot.HTTP.LatencyP50)

	b.WriteString("# HELP strata_http_latency_p95_ms HTTP latency p95 in ms\n")
	b.WriteString("# TYPE strata_http_latency_p95_ms gauge\n")
	fmt.Fprintf(&b, "strata_http_latency_p95_ms %.2f\n", snapshot.HTTP.LatencyP95)

	b.WriteString("# HELP strata_http_latency_p99_ms HTTP latency p99 in ms\n")
	b.WriteString("# TYPE strata_http_latency_p99_ms gauge\n")
	fmt.Fprintf(&b, "strata_http_latency_p99_ms %.2f\n", snapshot.HTTP.LatencyP99)

	return b.String()
}
