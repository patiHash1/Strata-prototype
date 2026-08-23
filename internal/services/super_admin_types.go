package services

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// ── Domain types ──

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
	ActiveUsers  int            `json:"active_users"`
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

// HTTPLatencyRecord captures a single HTTP request's latency.
type HTTPLatencyRecord struct {
	Path       string        `json:"path"`
	Method     string        `json:"method"`
	StatusCode int           `json:"status_code"`
	Latency    time.Duration `json:"latency"`
	Module     string        `json:"module"`
	Timestamp  time.Time     `json:"timestamp"`
}

// TrafficBucket holds aggregated request counts for a one-minute window.
type TrafficBucket struct {
	Timestamp time.Time `json:"timestamp"`
	Count2xx  int64     `json:"count_2xx"`
	Count5xx  int64     `json:"count_5xx"`
}

// Redis channel constants.
const (
	RedisChannelMaintenanceSync = "strata:events:maintenance-sync"
	RedisChannelSecuritySOC     = "strata:events:security-soc"
)

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

// ── SSE Subscriber ──

// sseSubscriber is a single SSE client channel.
type sseSubscriber struct {
	ch   chan []byte
	done chan struct{}
}

func newSOCEventID() string { return uuid.NewString() }
