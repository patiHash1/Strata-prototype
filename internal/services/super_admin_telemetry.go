package services

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/patiHash1/Strata-prototype/internal/logger"
)

// ActiveUserCounter counts active users for telemetry snapshots.
type ActiveUserCounter interface {
	CountActiveUsers(ctx context.Context, within time.Duration) (int, error)
}

// HTTPMetricsSource provides access to per-module HTTP metrics. ModuleHealth
// depends on this interface so a second implementation can be introduced later.
type HTTPMetricsSource interface {
	PerModuleMetrics() map[string]*ModuleHTTPMetrics
}

// Telemetry owns runtime, DB, HTTP, latency, and traffic metrics, plus the
// panic ring buffer. It is the deep module behind the observability seam:
// middleware pushes records in, consumers read snapshots/series/prom text out.
type Telemetry struct {
	pool            *pgxpool.Pool
	userCounter     ActiveUserCounter
	panicBuffer     *RingBuffer[SystemError]
	latencyBuffer   *RingBuffer[HTTPLatencyRecord]
	telemetryBuffer *RingBuffer[TelemetrySnapshot]
	trafficSeries   *RingBuffer[TrafficBucket]

	httpMu      sync.Mutex
	httpMetrics HTTPMetrics

	trafficMu     sync.Mutex
	currentBucket TrafficBucket
}

// NewTelemetry creates a Telemetry module.
func NewTelemetry(pool *pgxpool.Pool, userCounter ActiveUserCounter) *Telemetry {
	return &Telemetry{
		pool:            pool,
		userCounter:     userCounter,
		panicBuffer:     NewRingBuffer[SystemError](100),
		latencyBuffer:   NewRingBuffer[HTTPLatencyRecord](100),
		telemetryBuffer: NewRingBuffer[TelemetrySnapshot](100),
		trafficSeries:   NewRingBuffer[TrafficBucket](60),
		httpMetrics: HTTPMetrics{
			PerModule: make(map[string]*ModuleHTTPMetrics),
		},
	}
}

// RecordHTTPLatency records an HTTP request's latency for metrics.
func (t *Telemetry) RecordHTTPLatency(record HTTPLatencyRecord) {
	t.latencyBuffer.Push(record)

	t.httpMu.Lock()
	defer t.httpMu.Unlock()

	t.httpMetrics.TotalRequests++
	switch {
	case record.StatusCode >= 200 && record.StatusCode < 300:
		t.httpMetrics.Status2xx++
	case record.StatusCode >= 400 && record.StatusCode < 500:
		t.httpMetrics.Status4xx++
	case record.StatusCode >= 500:
		t.httpMetrics.Status5xx++
	}

	// Per-module tracking.
	mod := record.Module
	if mod == "" {
		mod = "unknown"
	}
	if t.httpMetrics.PerModule[mod] == nil {
		t.httpMetrics.PerModule[mod] = &ModuleHTTPMetrics{}
	}
	pm := t.httpMetrics.PerModule[mod]
	pm.Requests++
	pm.TotalLatencyMs += float64(record.Latency.Milliseconds())
	if record.StatusCode >= 500 {
		pm.Errors5xx++
	}

	// Bucket into per-minute time-series.
	bucketTS := record.Timestamp.Truncate(time.Minute)

	t.trafficMu.Lock()
	if t.currentBucket.Timestamp.Equal(bucketTS) {
		if record.StatusCode >= 200 && record.StatusCode < 300 {
			t.currentBucket.Count2xx++
		} else if record.StatusCode >= 500 {
			t.currentBucket.Count5xx++
		}
	} else {
		// Flush the in-progress bucket if non-zero.
		if t.currentBucket.Count2xx > 0 || t.currentBucket.Count5xx > 0 {
			t.trafficSeries.Push(t.currentBucket)
		}
		t.currentBucket = TrafficBucket{Timestamp: bucketTS}
		if record.StatusCode >= 200 && record.StatusCode < 300 {
			t.currentBucket.Count2xx++
		} else if record.StatusCode >= 500 {
			t.currentBucket.Count5xx++
		}
	}
	// Flush if we have accumulated a lot.
	if t.currentBucket.Count2xx+t.currentBucket.Count5xx >= 100 {
		t.trafficSeries.Push(t.currentBucket)
		t.currentBucket = TrafficBucket{Timestamp: bucketTS}
	}
	t.trafficMu.Unlock()
}

// RecordPanic stores a panic trace in the ring buffer and persists to DB.
func (t *Telemetry) RecordPanic(module string, errMsg string, stackTrace string, statusCode int) {
	errRec := SystemError{
		Module:       module,
		ErrorMessage: errMsg,
		StackTrace:   stackTrace,
		StatusCode:   statusCode,
		CreatedAt:    time.Now(),
	}
	t.panicBuffer.Push(errRec)

	if t.pool == nil {
		return
	}

	// Async persist to DB.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		repo := newSuperAdminRepository(t.pool)
		if dbErr := repo.InsertSystemError(ctx, &errRec); dbErr != nil {
			logger.Error("failed to persist panic", slog.String("error", dbErr.Error()))
		}
	}()
}

// CollectSnapshot gathers runtime and DB metrics into a snapshot.
func (t *Telemetry) CollectSnapshot() TelemetrySnapshot {
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
		RecentPanics: t.panicBuffer.Snapshot(),
	}

	// DB pool stats.
	if t.pool != nil {
		stats := t.pool.Stat()
		snapshot.DB = DBMetrics{
			AcquiredConns: stats.AcquiredConns(),
			IdleConns:     stats.IdleConns(),
			TotalConns:    stats.TotalConns(),
			MaxConns:      stats.MaxConns(),
		}
	}

	// HTTP metrics snapshot.
	t.httpMu.Lock()
	httpCopy := t.httpMetrics
	latencies := t.latencyBuffer.Snapshot()
	t.httpMu.Unlock()

	if len(latencies) > 0 {
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

	// Active users: users who logged in within the last 7 days.
	if t.userCounter != nil {
		if count, err := t.userCounter.CountActiveUsers(context.Background(), 7*24*time.Hour); err == nil {
			snapshot.ActiveUsers = count
		}
	}

	t.telemetryBuffer.Push(snapshot)
	return snapshot
}

// TrafficSeries returns the last 30 one-minute traffic buckets for charting.
func (t *Telemetry) TrafficSeries() []TrafficBucket {
	snap := t.trafficSeries.Snapshot()

	// Include the in-progress current bucket if it has a timestamp set.
	t.trafficMu.Lock()
	current := t.currentBucket
	t.trafficMu.Unlock()

	var result []TrafficBucket
	result = append(result, snap...)
	if !current.Timestamp.IsZero() {
		result = append(result, current)
	}

	// Return last 30 buckets.
	if len(result) > 30 {
		result = result[len(result)-30:]
	}
	return result
}

// PerModuleMetrics returns a snapshot of per-module HTTP metrics. It implements
// HTTPMetricsSource so ModuleHealth can consume them.
func (t *Telemetry) PerModuleMetrics() map[string]*ModuleHTTPMetrics {
	t.httpMu.Lock()
	defer t.httpMu.Unlock()

	out := make(map[string]*ModuleHTTPMetrics, len(t.httpMetrics.PerModule))
	for k, v := range t.httpMetrics.PerModule {
		out[k] = v
	}
	return out
}

// PrometheusMetrics returns metrics in Prometheus text exposition format.
func (t *Telemetry) PrometheusMetrics() string {
	snapshot := t.CollectSnapshot()

	var b strings.Builder

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

	b.WriteString("# HELP strata_db_acquired_conns Acquired connections\n")
	b.WriteString("# TYPE strata_db_acquired_conns gauge\n")
	fmt.Fprintf(&b, "strata_db_acquired_conns %d\n", snapshot.DB.AcquiredConns)

	b.WriteString("# HELP strata_db_idle_conns Idle connections\n")
	b.WriteString("# TYPE strata_db_idle_conns gauge\n")
	fmt.Fprintf(&b, "strata_db_idle_conns %d\n", snapshot.DB.IdleConns)

	b.WriteString("# HELP strata_db_total_conns Total connections\n")
	b.WriteString("# TYPE strata_db_total_conns gauge\n")
	fmt.Fprintf(&b, "strata_db_total_conns %d\n", snapshot.DB.TotalConns)

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
