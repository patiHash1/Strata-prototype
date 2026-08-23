package services

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/patiHash1/Strata-prototype/internal/logger"
	"github.com/redis/go-redis/v9"
)

// socEventRetentionDays is how long SOC events are kept in the DB.
const socEventRetentionDays = 25

// SOCMonitor owns security-event ingestion, recent-event buffering, SSE
// subscriber fan-out, and Redis pub/sub. Handlers publish events; the SSE
// handler consumes the subscriber stream.
type SOCMonitor struct {
	repo *superAdminRepository
	pool *pgxpool.Pool
	rdb  *redis.Client

	socBuffer *RingBuffer[SOCEvent]

	sseMu   sync.Mutex
	sseSubs map[string]*sseSubscriber

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewSOCMonitor creates a SOCMonitor module.
func NewSOCMonitor(pool *pgxpool.Pool, rdb *redis.Client) *SOCMonitor {
	ctx, cancel := context.WithCancel(context.Background())

	s := &SOCMonitor{
		repo:      newSuperAdminRepository(pool),
		pool:      pool,
		rdb:       rdb,
		socBuffer: NewRingBuffer[SOCEvent](50),
		sseSubs:   make(map[string]*sseSubscriber),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Start Redis subscriber for SOC events → SSE fan-out.
	if s.rdb != nil {
		s.wg.Add(1)
		go s.subscribeSOCEvents()
	}

	// Start sliding-window cleanup for stale SOC events.
	if s.pool != nil {
		s.wg.Add(1)
		go s.pruneSOCEventsLoop()
	}

	return s
}

// Shutdown stops background goroutines.
func (s *SOCMonitor) Shutdown() {
	s.cancel()
	s.wg.Wait()
}

// PublishSOCEvent publishes a security event to Redis and fans out to local
// SSE subscribers (when rdb is nil, local fan-out still works).
func (s *SOCMonitor) PublishSOCEvent(ctx context.Context, event SOCEvent) {
	event.ID = newSOCEventID()
	event.Timestamp = time.Now()

	data, err := json.Marshal(event)
	if err != nil {
		logger.Error("failed to marshal SOC event", slog.String("error", err.Error()))
		return
	}

	// Buffer for recent-event replay.
	s.socBuffer.Push(event)

	// Persist to database (async, non-blocking).
	if s.pool != nil {
		evt := event
		go func() {
			dbCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := s.repo.InsertSOCEvent(dbCtx, &evt); err != nil {
				logger.Error("failed to persist SOC event", slog.String("error", err.Error()))
			}
		}()
	}

	// Publish to Redis for multi-node fan-out.
	if s.rdb != nil {
		if err := s.rdb.Publish(ctx, RedisChannelSecuritySOC, data).Err(); err != nil {
			logger.Error("failed to publish SOC event", slog.String("error", err.Error()))
		}
	}

	// Fan out to local SSE subscribers.
	s.fanoutSSE(data)
}

// RecentSOCEvents returns a snapshot of the most recent SOC events from the
// in-memory ring buffer.
func (s *SOCMonitor) RecentSOCEvents() []SOCEvent {
	return s.socBuffer.Snapshot()
}

// AddSSESubscriber registers a new SSE subscriber and returns its channel.
func (s *SOCMonitor) AddSSESubscriber() (chan []byte, func()) {
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

func (s *SOCMonitor) fanoutSSE(data []byte) {
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

// subscribeSOCEvents listens for SOC events from Redis and fans out to local SSE subscribers.
func (s *SOCMonitor) subscribeSOCEvents() {
	defer s.wg.Done()

	if s.rdb == nil {
		logger.Warn("Redis unavailable, SOC subscriber disabled")
		return
	}

	pubsub := s.rdb.Subscribe(s.ctx, RedisChannelSecuritySOC)
	defer pubsub.Close()

	logger.Info("subscribed to Redis channel", slog.String("channel", RedisChannelSecuritySOC))

	ch := pubsub.Channel()
	for {
		select {
		case <-s.ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			logger.Info("SOC event received from Redis", slog.Int("subscribers", len(s.sseSubs)))
			s.fanoutSSE([]byte(msg.Payload))
		}
	}
}

// pruneSOCEventsLoop runs a sliding-window delete every 6 hours.
func (s *SOCMonitor) pruneSOCEventsLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	time.Sleep(30 * time.Second)
	s.pruneSOCEvents()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.pruneSOCEvents()
		}
	}
}

func (s *SOCMonitor) pruneSOCEvents() {
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	rows, err := s.repo.PruneSOCEvents(ctx, socEventRetentionDays*24*time.Hour)
	if err != nil {
		logger.Error("SOC event pruning failed", slog.String("error", err.Error()))
		return
	}
	if rows > 0 {
		logger.Info("pruned stale SOC events", slog.Int64("rows", rows), slog.Int("retention_days", socEventRetentionDays))
	}
}
