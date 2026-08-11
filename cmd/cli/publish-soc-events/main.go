// publish-soc-events is a temporary CLI tool that publishes mock SOC security
// events to Redis for testing the real-time security feed on the super admin
// dashboard.
//
// Usage:
//
//	go run ./cmd/cli/publish-soc-events [-count N] [-redis addr]
//
// It publishes one event per second and prints each event to stdout.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const redisChannel = "strata:events:security-soc"

var mockEvents = []struct {
	Type     string
	Severity string
	Message  string
}{
	{"failed_auth", "high", "Failed login attempt for admin@strata.local from 10.0.0.99 (3rd attempt)"},
	{"rls_violation", "critical", "Row-level security violation: user attempted to access cross-tenant data"},
	{"rate_limit", "medium", "Rate limit exceeded for POST /api/v1/crm/leads from 192.168.1.50"},
	{"failed_auth", "high", "Invalid API key used on /api/v1/fleet/telematics/ingest"},
	{"anomaly", "critical", "Unusual data export detected: 12,400 rows extracted by user-42"},
	{"failed_auth", "low", "Password reset requested for unknown email: test@example.com"},
	{"org_suspended", "high", "Organization acme-corp has been suspended for policy violation"},
	{"anomaly", "medium", "Spike in 5xx errors detected on /api/v1/hr/employees endpoint"},
	{"rate_limit", "low", "API key rate limit approaching threshold for key-abc-123"},
	{"failed_auth", "critical", "Brute-force attack detected: 50 failed logins from 203.0.113.42 in 60s"},
}

func main() {
	count := flag.Int("count", 10, "number of events to publish")
	redisAddr := flag.String("redis", "localhost:6379", "Redis address")
	redisPass := flag.String("redis-pass", "", "Redis password")
	flag.Parse()

	rdb := redis.NewClient(&redis.Options{
		Addr:     *redisAddr,
		Password: *redisPass,
		DB:       0,
	})
	defer rdb.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Check Redis connectivity.
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis at %s: %v", *redisAddr, err)
	}
	fmt.Printf("✓ Connected to Redis at %s\n\n", *redisAddr)

	now := time.Now()
	for i := 0; i < *count; i++ {
		mock := mockEvents[i%len(mockEvents)]

		event := map[string]any{
			"id":         uuid.NewString(),
			"type":       mock.Type,
			"severity":   mock.Severity,
			"message":    mock.Message,
			"ip_address": fmt.Sprintf("10.0.0.%d", 50+i),
			"user_id":    uuid.NewString(),
			"timestamp":  now.Add(time.Duration(i) * time.Second),
		}

		data, err := json.Marshal(event)
		if err != nil {
			log.Printf("Failed to marshal event: %v", err)
			continue
		}

		if err := rdb.Publish(ctx, redisChannel, data).Err(); err != nil {
			log.Printf("Failed to publish event: %v", err)
			continue
		}

		severity := strings.ToUpper(mock.Severity)
		fmt.Printf("[%d/%d] %s %-10s %s\n", i+1, *count, severity, mock.Type, mock.Message)
		time.Sleep(1 * time.Second)
	}

	fmt.Printf("\n✓ Published %d events to %s\n", *count, redisChannel)
}
