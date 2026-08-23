package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/patiHash1/Strata-prototype/internal/logger"
	"github.com/redis/go-redis/v9"
)

// MaintenanceChecker answers whether a scope+target is under maintenance.
// ModuleHealth depends on this interface so a second implementation can be
// introduced later.
type MaintenanceChecker interface {
	IsUnderMaintenance(scope, targetID string) (*MaintenanceRule, bool)
}

// Maintenance owns the partitioned-maintenance rule cache, its Redis
// invalidation sync, and the rule CRUD. Middleware reads IsUnderMaintenance;
// handlers write rules via Toggle/Delete/List.
type Maintenance struct {
	repo *superAdminRepository
	rdb  *redis.Client

	cacheMu    sync.RWMutex
	cacheRules map[string]*MaintenanceRule

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewMaintenance creates a Maintenance module and loads the rule cache.
func NewMaintenance(pool *pgxpool.Pool, rdb *redis.Client) *Maintenance {
	ctx, cancel := context.WithCancel(context.Background())
	var repo *superAdminRepository
	if pool != nil {
		repo = newSuperAdminRepository(pool)
	}

	m := &Maintenance{
		repo:       repo,
		rdb:        rdb,
		cacheRules: make(map[string]*MaintenanceRule),
		ctx:        ctx,
		cancel:     cancel,
	}

	if m.repo != nil {
		if err := m.reloadCache(context.Background()); err != nil {
			logger.Error("initial maintenance cache load failed", slog.String("error", err.Error()))
		}
	}

	// Start Redis subscriber for cache invalidation.
	if m.rdb != nil {
		m.wg.Add(1)
		go m.subscribeMaintenanceSync()
	}

	return m
}

// Shutdown stops the Redis subscriber goroutine.
func (m *Maintenance) Shutdown() {
	m.cancel()
	m.wg.Wait()
}

func (m *Maintenance) cacheKey(scope, targetID string) string {
	return scope + ":" + targetID
}

func (m *Maintenance) reloadCache(ctx context.Context) error {
	if m.repo == nil {
		return nil
	}
	rules, err := m.repo.ListActiveMaintenanceRules(ctx)
	if err != nil {
		return err
	}

	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()
	m.cacheRules = make(map[string]*MaintenanceRule, len(rules))
	for i := range rules {
		rule := rules[i]
		m.cacheRules[m.cacheKey(rule.Scope, rule.TargetID)] = &rule
	}
	return nil
}

// IsUnderMaintenance checks if the given scope+targetID is under maintenance.
func (m *Maintenance) IsUnderMaintenance(scope, targetID string) (*MaintenanceRule, bool) {
	m.cacheMu.RLock()
	defer m.cacheMu.RUnlock()

	rule, ok := m.cacheRules[m.cacheKey(scope, targetID)]
	if !ok {
		return nil, false
	}
	return rule, rule.IsActive
}

// ToggleMaintenance activates or deactivates a maintenance rule and publishes
// a cache-invalidation event to Redis.
func (m *Maintenance) ToggleMaintenance(ctx context.Context, req MaintenanceToggleRequest) (*MaintenanceRule, error) {
	if m.repo == nil {
		return nil, fmt.Errorf("database not available")
	}

	rule := &MaintenanceRule{
		Scope:        req.Scope,
		TargetID:     req.TargetID,
		IsActive:     req.IsActive,
		Reason:       req.Reason,
		AllowedRoles: req.AllowedRoles,
	}

	if err := m.repo.UpdateMaintenanceRule(ctx, rule); err != nil {
		return nil, fmt.Errorf("upsert maintenance rule: %w", err)
	}

	// Publish cache invalidation event.
	msg, _ := json.Marshal(map[string]string{
		"action":    "toggle",
		"scope":     req.Scope,
		"target_id": req.TargetID,
	})
	if m.rdb != nil {
		if err := m.rdb.Publish(ctx, RedisChannelMaintenanceSync, msg).Err(); err != nil {
			logger.Error("failed to publish maintenance sync", slog.String("error", err.Error()))
		}
	}

	// Reload local cache immediately.
	if err := m.reloadCache(ctx); err != nil {
		logger.Error("cache reload after toggle failed", slog.String("error", err.Error()))
	}

	return rule, nil
}

// DeleteMaintenanceRule soft-deactivates a maintenance rule by ID and publishes
// a cache-invalidation event via Redis.
func (m *Maintenance) DeleteMaintenanceRule(ctx context.Context, id int64) error {
	if m.repo == nil {
		return fmt.Errorf("database not available")
	}
	if err := m.repo.DeleteMaintenanceRule(ctx, id); err != nil {
		return fmt.Errorf("delete maintenance rule: %w", err)
	}

	// Publish cache invalidation event.
	msg, _ := json.Marshal(map[string]string{
		"action": "delete",
		"id":     fmt.Sprintf("%d", id),
	})
	if m.rdb != nil {
		if err := m.rdb.Publish(ctx, RedisChannelMaintenanceSync, msg).Err(); err != nil {
			logger.Error("failed to publish maintenance sync", slog.String("error", err.Error()))
		}
	}

	// Reload local cache immediately.
	if err := m.reloadCache(ctx); err != nil {
		logger.Error("cache reload after delete failed", slog.String("error", err.Error()))
	}

	return nil
}

// ListActiveRules returns all active maintenance rules.
func (m *Maintenance) ListActiveRules(ctx context.Context) ([]MaintenanceRule, error) {
	if m.repo == nil {
		return []MaintenanceRule{}, nil
	}
	return m.repo.ListActiveMaintenanceRules(ctx)
}

// ListAllMaintenanceRules returns all maintenance rules (active and inactive).
func (m *Maintenance) ListAllMaintenanceRules(ctx context.Context) ([]MaintenanceRule, error) {
	if m.repo == nil {
		return []MaintenanceRule{}, nil
	}
	return m.repo.ListAllMaintenanceRules(ctx)
}

// CacheRules returns a snapshot of the active rule cache (module/feature-level).
// Used by ModuleHealth to surface feature-level maintenance state.
func (m *Maintenance) CacheRules() map[string]*MaintenanceRule {
	m.cacheMu.RLock()
	defer m.cacheMu.RUnlock()

	out := make(map[string]*MaintenanceRule, len(m.cacheRules))
	for k, v := range m.cacheRules {
		out[k] = v
	}
	return out
}

// subscribeMaintenanceSync listens for Redis cache-invalidation messages.
func (m *Maintenance) subscribeMaintenanceSync() {
	defer m.wg.Done()

	if m.rdb == nil {
		return
	}

	pubsub := m.rdb.Subscribe(m.ctx, RedisChannelMaintenanceSync)
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case <-m.ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			logger.Info("maintenance sync received", slog.String("payload", msg.Payload))
			if err := m.reloadCache(context.Background()); err != nil {
				logger.Error("cache reload from sync failed", slog.String("error", err.Error()))
			}
		}
	}
}
