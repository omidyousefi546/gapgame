package storage

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ConnectionPool manages database connections
type ConnectionPool struct {
	db              *gorm.DB
	maxConnections  int
	idleConnections int
	activeCount     int
	waitingRequests int
	mu              sync.RWMutex
	healthCheckTick *time.Ticker
	log             *zap.Logger
}

// PoolStats holds pool statistics
type PoolStats struct {
	MaxConnections  int
	ActiveCount     int
	IdleCount       int
	WaitingRequests int
	Timestamp       time.Time
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(db *gorm.DB, maxConnections int, log *zap.Logger) *ConnectionPool {
	cp := &ConnectionPool{
		db:              db,
		maxConnections:  maxConnections,
		idleConnections: maxConnections,
		log:             log,
	}

	// Start health check routine
	cp.startHealthCheck()

	return cp
}

// AcquireConnection acquires a connection from the pool
func (cp *ConnectionPool) AcquireConnection(ctx context.Context) (*gorm.DB, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	// Wait for available connection
	for cp.activeCount >= cp.maxConnections {
		cp.waitingRequests++
		cp.mu.Unlock()

		// Wait a bit before trying again
		select {
		case <-ctx.Done():
			cp.mu.Lock()
			cp.waitingRequests--
			cp.mu.Unlock()
			return nil, ctx.Err()
		case <-time.After(100 * time.Millisecond):
			cp.mu.Lock()
		}
	}

	cp.activeCount++
	cp.idleConnections--

	return cp.db, nil
}

// ReleaseConnection releases a connection back to the pool
func (cp *ConnectionPool) ReleaseConnection() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if cp.activeCount > 0 {
		cp.activeCount--
		cp.idleConnections++
	}
}

// GetStats returns current pool statistics
func (cp *ConnectionPool) GetStats() PoolStats {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	return PoolStats{
		MaxConnections:  cp.maxConnections,
		ActiveCount:     cp.activeCount,
		IdleCount:       cp.idleConnections,
		WaitingRequests: cp.waitingRequests,
		Timestamp:       time.Now(),
	}
}

// startHealthCheck starts periodic health checks
func (cp *ConnectionPool) startHealthCheck() {
	cp.healthCheckTick = time.NewTicker(30 * time.Second)

	go func() {
		for range cp.healthCheckTick.C {
			if err := cp.healthCheck(); err != nil {
				cp.log.Error("connection pool health check failed", zap.Error(err))
			}
		}
	}()
}

// healthCheck performs a health check on the database
func (cp *ConnectionPool) healthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := cp.db.WithContext(ctx).Raw("SELECT 1").Error; err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	stats := cp.GetStats()
	cp.log.Debug("connection pool health check passed",
		zap.Int("active_connections", stats.ActiveCount),
		zap.Int("idle_connections", stats.IdleCount),
		zap.Int("waiting_requests", stats.WaitingRequests),
	)

	return nil
}

// Close closes the connection pool
func (cp *ConnectionPool) Close() error {
	if cp.healthCheckTick != nil {
		cp.healthCheckTick.Stop()
	}

	sqlDB, err := cp.db.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}

// WithConnection executes a function with a connection from the pool
func (cp *ConnectionPool) WithConnection(ctx context.Context, fn func(*gorm.DB) error) error {
	conn, err := cp.AcquireConnection(ctx)
	if err != nil {
		return err
	}
	defer cp.ReleaseConnection()

	return fn(conn)
}

// OptimizePoolSettings optimizes connection pool settings based on load
func (cp *ConnectionPool) OptimizePoolSettings() {
	statsFunc := func() {
		cp.mu.RLock()
		stats := PoolStats{
			MaxConnections:  cp.maxConnections,
			ActiveCount:     cp.activeCount,
			IdleCount:       cp.idleConnections,
			WaitingRequests: cp.waitingRequests,
		}
		cp.mu.RUnlock()

		// Log optimization suggestions
		if stats.WaitingRequests > 0 {
			cp.log.Warn("connection pool strain detected",
				zap.Int("waiting_requests", stats.WaitingRequests),
				zap.Int("active_connections", stats.ActiveCount),
				zap.String("recommendation", "consider increasing max_connections"),
			)
		}

		if stats.ActiveCount == 0 && stats.IdleCount > 20 {
			cp.log.Info("underutilized connection pool",
				zap.Int("idle_connections", stats.IdleCount),
				zap.String("recommendation", "consider decreasing max_connections"),
			)
		}
	}

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			statsFunc()
		}
	}()
}
