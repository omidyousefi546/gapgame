package bot

import (
	"context"
	"encoding/json"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"

	"GapGame/internal/session"
)

// OptimizedMatchWorker handles efficient user matching with persistent state
type OptimizedMatchWorker struct {
	handler *Handler
	cache   *session.RedisCache
	log     *zap.Logger
	ticker  *time.Ticker
}

// StartMatchWorker starts the optimized matching worker
func (h *Handler) StartMatchWorker(log *zap.Logger, ctx context.Context) {
	// Get Redis client from session manager
	rdbClient := h.redis.GetRedisClient()
	cache := session.NewRedisCache(rdbClient)

	worker := &OptimizedMatchWorker{
		handler: h,
		cache:   cache,
		log:     log,
		ticker:  time.NewTicker(2 * time.Second), // Check every 2 seconds
	}

	log.Info("[MatchWorker] Started with optimized matching")
	worker.start(ctx)
}

// start begins the matching worker loop
func (w *OptimizedMatchWorker) start(ctx context.Context) {
	defer w.ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.log.Info("[MatchWorker] Stopped")
			return
		case <-w.ticker.C:
			w.processMatches(ctx)
		}
	}
}

// processMatches attempts to find and create matches
func (w *OptimizedMatchWorker) processMatches(ctx context.Context) {
	// Get all waiting users from queue
	queueEntries, err := w.getAllQueueEntries(ctx)
	if err != nil {
		w.log.Error("[MatchWorker] failed to get queue entries", zap.Error(err))
		return
	}

	if len(queueEntries) < 2 {
		return
	}

	// Try to match users
	matched := make(map[int64]bool)

	for i := 0; i < len(queueEntries); i++ {
		if matched[queueEntries[i].TelegramID] {
			continue
		}

		// Find compatible match
		for j := i + 1; j < len(queueEntries); j++ {
			if matched[queueEntries[j].TelegramID] {
				continue
			}

			if w.isCompatible(&queueEntries[i], &queueEntries[j]) {
				// createMatch is responsible for queue cleanup on success
				// and for refunding/notifying on failure.
				w.createMatch(ctx, &queueEntries[i], &queueEntries[j])
				matched[queueEntries[i].TelegramID] = true
				matched[queueEntries[j].TelegramID] = true
				break
			}
		}
	}
}

// getAllQueueEntries retrieves all current queue entries from Redis.
// We trust the queue contents — RemoveFromQueue is always called alongside
// LeaveQueue/cancel paths, so the per-entry waiting-key check has been
// dropped in favour of a single SCAN + per-key LRANGE.
func (w *OptimizedMatchWorker) getAllQueueEntries(ctx context.Context) ([]session.QueueEntry, error) {
	keys, err := w.cache.GetList(ctx, "chat:queue:*")
	if err != nil {
		return nil, err
	}

	var entries []session.QueueEntry
	seen := make(map[int64]bool)

	for _, key := range keys {
		queueItems, err := w.cache.ListGetAll(ctx, key)
		if err != nil {
			continue
		}

		for _, item := range queueItems {
			var entry session.QueueEntry
			if err := json.Unmarshal([]byte(item), &entry); err != nil {
				continue
			}
			// De-duplicate in case a user somehow ended up in two queues.
			if seen[entry.TelegramID] {
				continue
			}
			seen[entry.TelegramID] = true
			entries = append(entries, entry)
		}
	}

	return entries, nil
}

// isCompatible checks if two users can be matched based on their filters and genders
func (w *OptimizedMatchWorker) isCompatible(u1, u2 *session.QueueEntry) bool {
	// Both users exist and are different
	if u1.TelegramID == u2.TelegramID {
		return false
	}

	// Both are random - they don't care who they match with
	if u1.Filter == "random" && u2.Filter == "random" {
		return true
	}

	// One is random and the other has a gender filter
	if u1.Filter == "random" && u2.Filter != "random" {
		// Check if u1's actual gender matches u2's filter
		return u1.Gender == u2.Filter
	}

	if u2.Filter == "random" && u1.Filter != "random" {
		// Check if u2's actual gender matches u1's filter
		return u2.Gender == u1.Filter
	}

	// Both have gender-specific filters - check if complementary
	return isComplementaryFilter(u1.Filter, u2.Filter)
}

// isComplementaryFilter checks if two filters are compatible
func isComplementaryFilter(filter1, filter2 string) bool {
	// Only call this when both filters are gender-specific and different
	complementary := map[string][]string{
		"male":          {"female"},
		"female":        {"male"},
		"nearby":        {"nearby_male", "nearby_female"},
		"nearby_male":   {"nearby_female"},
		"nearby_female": {"nearby_male"},
	}

	allowed, exists := complementary[filter1]
	if !exists {
		return false // Reject unknown filters
	}

	for _, f := range allowed {
		if f == filter2 {
			return true
		}
	}
	return false
}

// createMatch creates a chat session for two matched users.
func (w *OptimizedMatchWorker) createMatch(ctx context.Context, user1, user2 *session.QueueEntry) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	chatSession := &session.ChatSession{
		User1ID:    user1.TelegramID,
		User2ID:    user2.TelegramID,
		User1Coins: user1.Cost,
		User2Coins: user2.Cost,
		StartedAt:  time.Now(),
	}

	if err := w.handler.redis.SetActiveChat(ctxWithTimeout, chatSession); err != nil {
		w.log.Error("[MatchWorker] failed to start chat",
			zap.Error(err),
			zap.Int64("user1", user1.TelegramID),
			zap.Int64("user2", user2.TelegramID),
		)

		// Refund coins and notify both users.
		if user1.Cost > 0 {
			if rerr := w.handler.users.AwardCoinsByTelegramID(user1.TelegramID, user1.Cost, "match_error"); rerr != nil {
				w.log.Error("[MatchWorker] refund failed for user1", zap.Error(rerr))
			}
		}
		if user2.Cost > 0 {
			if rerr := w.handler.users.AwardCoinsByTelegramID(user2.TelegramID, user2.Cost, "match_error"); rerr != nil {
				w.log.Error("[MatchWorker] refund failed for user2", zap.Error(rerr))
			}
		}

		w.handler.bot.Send(&tele.User{ID: user1.TelegramID}, "❌ خطا در برقراری اتصال. سکه‌های شما برگشت داده شد.")
		w.handler.bot.Send(&tele.User{ID: user2.TelegramID}, "❌ خطا در برقراری اتصال. سکه‌های شما برگشت داده شد.")
		return
	}

	// Remove both users from any waiting list (processMatches also calls
	// removeFromQueue, but doing it here too is safe and avoids races).
	w.removeFromQueue(ctxWithTimeout, user1.TelegramID)
	w.removeFromQueue(ctxWithTimeout, user2.TelegramID)

	const msg = "👀 پیدا شد! متصل شدی. به مخاطبت سلام بده 🗣️"
	kb := ActiveChatKeyboard()

	w.handler.bot.Send(&tele.User{ID: user1.TelegramID}, msg, kb)
	w.handler.bot.Send(&tele.User{ID: user2.TelegramID}, msg, kb)

	w.log.Info("[MatchWorker] Users matched successfully",
		zap.Int64("user1", user1.TelegramID),
		zap.Int64("user2", user2.TelegramID),
	)
}

// removeFromQueue removes a user from all queues
func (w *OptimizedMatchWorker) removeFromQueue(ctx context.Context, userID int64) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	// Get the filter the user was waiting in
	filter, err := w.handler.redis.GetWaitingFilter(ctxWithTimeout, userID)
	if err == nil && filter != "" {
		w.handler.redis.RemoveFromQueue(ctxWithTimeout, userID, filter)
	}

	// Also remove waiting state
	w.handler.redis.RemoveWaitingState(ctxWithTimeout, userID)
}
