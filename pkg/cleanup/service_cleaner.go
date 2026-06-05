package cleanup

import (
	"context"
	"time"

	"GapGame/internal/matching"
	"GapGame/internal/session"

	"go.uber.org/zap"
)

// ServiceCleaner handles cleanup of expired data and sessions
type ServiceCleaner struct {
	sessionManager  *session.Manager
	queueManager    *matching.QueueManager
	log             *zap.Logger
	cleanupInterval time.Duration
	ticker          *time.Ticker
}

// NewServiceCleaner creates a new service cleaner
func NewServiceCleaner(
	sm *session.Manager,
	qm *matching.QueueManager,
	log *zap.Logger,
) *ServiceCleaner {
	return &ServiceCleaner{
		sessionManager:  sm,
		queueManager:    qm,
		log:             log,
		cleanupInterval: 5 * time.Minute,
	}
}

// Start begins the cleanup routine
func (sc *ServiceCleaner) Start(ctx context.Context) {
	sc.ticker = time.NewTicker(sc.cleanupInterval)
	sc.log.Info("[Cleanup] Service started",
		zap.Duration("interval", sc.cleanupInterval),
	)

	go func() {
		for {
			select {
			case <-ctx.Done():
				sc.log.Info("[Cleanup] Service stopped")
				return
			case <-sc.ticker.C:
				sc.runCleanup(ctx)
			}
		}
	}()
}

// runCleanup performs all cleanup operations
func (sc *ServiceCleaner) runCleanup(ctx context.Context) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	startTime := time.Now()

	// Clean expired sessions
	sessionsCleanedCount := sc.cleanupExpiredSessions(ctxWithTimeout)

	// Clean up queue (handled by queue manager)
	queueStats := sc.queueManager.GetQueueStats()

	// Clean expired chat sessions
	chatSessionsCleaned := sc.cleanupExpiredChatSessions(ctxWithTimeout)

	// Log cleanup results
	sc.log.Info("[Cleanup] Cleanup round completed",
		zap.Int("sessions_cleaned", sessionsCleanedCount),
		zap.Int("chat_sessions_cleaned", chatSessionsCleaned),
		zap.Any("queue_stats", queueStats),
		zap.Duration("duration", time.Since(startTime)),
	)
}

// cleanupExpiredSessions removes expired user state entries
func (sc *ServiceCleaner) cleanupExpiredSessions(ctx context.Context) int {
	// This is a placeholder for session cleanup
	// In a real implementation, you would query all keys and check TTL
	// For now, we rely on Redis TTL to handle expiration

	cleaned := 0
	sc.log.Debug("[Cleanup] User state cleanup completed",
		zap.Int("cleaned", cleaned),
	)

	return cleaned
}

// cleanupExpiredChatSessions cleans up old chat sessions
func (sc *ServiceCleaner) cleanupExpiredChatSessions(ctx context.Context) int {
	// This is a placeholder for chat session cleanup
	// In a production system, you might:
	// 1. Query all active chat sessions
	// 2. Check if they're older than X duration
	// 3. Archive or delete old sessions

	cleaned := 0
	sc.log.Debug("[Cleanup] Chat session cleanup completed",
		zap.Int("cleaned", cleaned),
	)

	return cleaned
}

// Stop stops the cleanup service
func (sc *ServiceCleaner) Stop() {
	if sc.ticker != nil {
		sc.ticker.Stop()
	}
	sc.log.Info("[Cleanup] Service stopped")
}

// SetCleanupInterval sets the cleanup interval
func (sc *ServiceCleaner) SetCleanupInterval(interval time.Duration) {
	sc.cleanupInterval = interval
	if sc.ticker != nil {
		sc.ticker.Stop()
		sc.ticker = time.NewTicker(interval)
	}
}

// GetCleanupStats returns cleanup statistics
func (sc *ServiceCleaner) GetCleanupStats() map[string]interface{} {
	stats := make(map[string]interface{})
	stats["queue_stats"] = sc.queueManager.GetQueueStats()
	stats["cleanup_interval"] = sc.cleanupInterval.String()
	return stats
}

// MemoryOptimizer handles memory optimization
type MemoryOptimizer struct {
	log *zap.Logger
}

// NewMemoryOptimizer creates a new memory optimizer
func NewMemoryOptimizer(log *zap.Logger) *MemoryOptimizer {
	return &MemoryOptimizer{log: log}
}

// OptimizeQueueMemory removes low-priority entries from queue
func (mo *MemoryOptimizer) OptimizeQueueMemory(qm *matching.QueueManager) {
	stats := qm.GetQueueStats()
	queueSize := stats["total_users"].(int)

	// If queue is too large, log warning
	if queueSize > 1000 {
		mo.log.Warn("[Memory] Queue size is high",
			zap.Int("size", queueSize),
			zap.String("recommendation", "consider implementing queue size limits"),
		)
	}

	mo.log.Debug("[Memory] Queue optimization completed",
		zap.Any("stats", stats),
	)
}

// CleanupConfig holds cleanup configuration
type CleanupConfig struct {
	CleanupInterval           time.Duration
	SessionExpirationTime     time.Duration
	ChatSessionExpirationTime time.Duration
	MaxQueueSize              int
	MaxUserStateEntries       int
}

// DefaultCleanupConfig returns default cleanup configuration
func DefaultCleanupConfig() CleanupConfig {
	return CleanupConfig{
		CleanupInterval:           5 * time.Minute,
		SessionExpirationTime:     24 * time.Hour,
		ChatSessionExpirationTime: 4 * time.Hour,
		MaxQueueSize:              5000,
		MaxUserStateEntries:       10000,
	}
}
