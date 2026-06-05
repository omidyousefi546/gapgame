package matching

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// QueueManager efficiently manages matching queue with indexing
type QueueManager struct {
	// In-memory cache of queue entries indexed by various criteria
	entries    map[string]*QueueEntry   // userID -> entry
	byGender   map[string][]*QueueEntry // gender -> [entries]
	byProvince map[string][]*QueueEntry // province -> [entries]
	byFilter   map[string][]*QueueEntry // filter type -> [entries]

	mu           sync.RWMutex
	timeout      time.Duration
	maxQueueSize int
	log          *zap.Logger

	cleanupTicker *time.Ticker
}

// QueueEntry represents a single queue entry
type QueueEntry struct {
	UserID     string
	TelegramID int64
	Gender     string
	Age        int
	Province   string
	City       string
	Latitude   *float64
	Longitude  *float64
	Filter     string
	Cost       int
	JoinedAt   time.Time
	ExpiresAt  time.Time
}

// NewQueueManager creates a new queue manager
func NewQueueManager(timeout time.Duration, maxSize int, log *zap.Logger) *QueueManager {
	qm := &QueueManager{
		entries:      make(map[string]*QueueEntry),
		byGender:     make(map[string][]*QueueEntry),
		byProvince:   make(map[string][]*QueueEntry),
		byFilter:     make(map[string][]*QueueEntry),
		timeout:      timeout,
		maxQueueSize: maxSize,
		log:          log,
	}

	// Start cleanup routine
	qm.startCleanup()

	return qm
}

// Enqueue adds a user to the queue
func (qm *QueueManager) Enqueue(entry *QueueEntry) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	// Check max queue size
	if len(qm.entries) >= qm.maxQueueSize {
		return fmt.Errorf("queue is full")
	}

	// Set expiration time
	entry.JoinedAt = time.Now()
	entry.ExpiresAt = time.Now().Add(qm.timeout)

	// Add to main entry map
	qm.entries[entry.UserID] = entry

	// Add to gender index
	if qm.byGender[entry.Gender] == nil {
		qm.byGender[entry.Gender] = make([]*QueueEntry, 0)
	}
	qm.byGender[entry.Gender] = append(qm.byGender[entry.Gender], entry)

	// Add to province index
	if entry.Province != "" {
		if qm.byProvince[entry.Province] == nil {
			qm.byProvince[entry.Province] = make([]*QueueEntry, 0)
		}
		qm.byProvince[entry.Province] = append(qm.byProvince[entry.Province], entry)
	}

	// Add to filter index
	if qm.byFilter[entry.Filter] == nil {
		qm.byFilter[entry.Filter] = make([]*QueueEntry, 0)
	}
	qm.byFilter[entry.Filter] = append(qm.byFilter[entry.Filter], entry)

	qm.log.Debug("user enqueued",
		zap.String("user_id", entry.UserID),
		zap.String("filter", entry.Filter),
		zap.Int("queue_size", len(qm.entries)),
	)

	return nil
}

// Dequeue removes a user from the queue
func (qm *QueueManager) Dequeue(userID string) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	entry, exists := qm.entries[userID]
	if !exists {
		return fmt.Errorf("user not in queue")
	}

	// Remove from main map
	delete(qm.entries, userID)

	// Remove from gender index
	if entries, ok := qm.byGender[entry.Gender]; ok {
		qm.byGender[entry.Gender] = removeEntry(entries, userID)
	}

	// Remove from province index
	if entry.Province != "" {
		if entries, ok := qm.byProvince[entry.Province]; ok {
			qm.byProvince[entry.Province] = removeEntry(entries, userID)
		}
	}

	// Remove from filter index
	if entries, ok := qm.byFilter[entry.Filter]; ok {
		qm.byFilter[entry.Filter] = removeEntry(entries, userID)
	}

	qm.log.Debug("user dequeued", zap.String("user_id", userID))

	return nil
}

// GetCandidatesByFilter returns potential matches for a given filter
func (qm *QueueManager) GetCandidatesByFilter(entry *QueueEntry) []*QueueEntry {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	candidates := make([]*QueueEntry, 0)

	// Get candidates based on filter type
	switch entry.Filter {
	case "random":
		// Random matches anyone
		for _, candidate := range qm.entries {
			if candidate.UserID != entry.UserID && !candidate.ExpiresAt.Before(time.Now()) {
				candidates = append(candidates, candidate)
			}
		}
	case "male", "female":
		// Gender-specific matches get opposite gender
		oppositeGender := getOppositeGender(entry.Gender)
		if entries, ok := qm.byGender[oppositeGender]; ok {
			for _, candidate := range entries {
				if candidate.UserID != entry.UserID && !candidate.ExpiresAt.Before(time.Now()) {
					candidates = append(candidates, candidate)
				}
			}
		}
	case "nearby":
		// Nearby matches others in same province
		if entry.Province != "" {
			if entries, ok := qm.byProvince[entry.Province]; ok {
				for _, candidate := range entries {
					if candidate.UserID != entry.UserID && !candidate.ExpiresAt.Before(time.Now()) {
						candidates = append(candidates, candidate)
					}
				}
			}
		}
	}

	return candidates
}

// GetQueueSize returns current queue size
func (qm *QueueManager) GetQueueSize() int {
	qm.mu.RLock()
	defer qm.mu.RUnlock()
	return len(qm.entries)
}

// GetQueueStats returns queue statistics
func (qm *QueueManager) GetQueueStats() map[string]interface{} {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_users"] = len(qm.entries)
	stats["by_filter"] = make(map[string]int)

	for filter, entries := range qm.byFilter {
		stats["by_filter"].(map[string]int)[filter] = len(entries)
	}

	return stats
}

// startCleanup starts periodic cleanup of expired entries
func (qm *QueueManager) startCleanup() {
	qm.cleanupTicker = time.NewTicker(10 * time.Second)

	go func() {
		for range qm.cleanupTicker.C {
			qm.cleanupExpired()
		}
	}()
}

// cleanupExpired removes expired entries from queue
func (qm *QueueManager) cleanupExpired() {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	now := time.Now()
	var expired []*QueueEntry

	// Find expired entries
	for _, entry := range qm.entries {
		if entry.ExpiresAt.Before(now) {
			expired = append(expired, entry)
		}
	}

	// Remove expired entries
	for _, entry := range expired {
		delete(qm.entries, entry.UserID)

		// Clean up indices
		if entries, ok := qm.byGender[entry.Gender]; ok {
			qm.byGender[entry.Gender] = removeEntry(entries, entry.UserID)
		}
		if entry.Province != "" {
			if entries, ok := qm.byProvince[entry.Province]; ok {
				qm.byProvince[entry.Province] = removeEntry(entries, entry.UserID)
			}
		}
		if entries, ok := qm.byFilter[entry.Filter]; ok {
			qm.byFilter[entry.Filter] = removeEntry(entries, entry.UserID)
		}

		qm.log.Debug("expired entry cleaned",
			zap.String("user_id", entry.UserID),
		)
	}

	if len(expired) > 0 {
		qm.log.Info("cleaned expired entries",
			zap.Int("expired_count", len(expired)),
		)
	}
}

// Close closes the queue manager
func (qm *QueueManager) Close() {
	if qm.cleanupTicker != nil {
		qm.cleanupTicker.Stop()
	}
}

// Helper functions

func removeEntry(entries []*QueueEntry, userID string) []*QueueEntry {
	result := make([]*QueueEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.UserID != userID {
			result = append(result, entry)
		}
	}
	return result
}

func getOppositeGender(gender string) string {
	if gender == "male" {
		return "female"
	}
	return "male"
}
