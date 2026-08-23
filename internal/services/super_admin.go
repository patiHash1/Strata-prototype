package services

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// SuperAdmin is the composition root for the observability modules. It owns
// the shared pool/redis lifecycle, constructs the four focused modules, and
// exposes PingRedis (a probe of its own rdb dependency). It is not a facade:
// it re-exports no domain methods — handlers and middleware reach the concrete
// modules directly.
type SuperAdmin struct {
	pool         *pgxpool.Pool
	rdb          *redis.Client
	Telemetry    *Telemetry
	SOCMonitor   *SOCMonitor
	Maintenance  *Maintenance
	ModuleHealth *ModuleHealthSvc
}

// NewSuperAdmin constructs the observability modules and wires them together.
func NewSuperAdmin(pool *pgxpool.Pool, rdb *redis.Client, userCounter ActiveUserCounter) *SuperAdmin {
	telemetry := NewTelemetry(pool, userCounter)
	maintenance := NewMaintenance(pool, rdb)
	soc := NewSOCMonitor(pool, rdb)
	moduleHealth := NewModuleHealth(pool, telemetry, maintenance)

	return &SuperAdmin{
		pool:         pool,
		rdb:          rdb,
		Telemetry:    telemetry,
		SOCMonitor:   soc,
		Maintenance:  maintenance,
		ModuleHealth: moduleHealth,
	}
}

// PingRedis reports whether the optional Redis client is configured and reachable.
func (s *SuperAdmin) PingRedis(ctx context.Context) (bool, error) {
	if s.rdb == nil {
		return false, nil
	}
	if err := s.rdb.Ping(ctx).Err(); err != nil {
		return false, err
	}
	return true, nil
}

// Shutdown stops all module goroutines.
func (s *SuperAdmin) Shutdown() {
	s.Maintenance.Shutdown()
	s.SOCMonitor.Shutdown()
}
