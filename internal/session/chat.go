// internal/session/manager.go — بخش chat queue

package session

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

const (
	chatQueueTTL  = 1 * time.Minute
	activeChatTTL = 4 * time.Hour
)

type QueueEntry struct {
	TelegramID int64     `json:"telegram_id"`
	Gender     string    `json:"gender"`
	Filter     string    `json:"filter"`
	JoinedAt   time.Time `json:"joined_at"`
	Cost       int       `json:"cost"`
	Lat        *float64  `json:"lat,omitempty"`
	Lon        *float64  `json:"lon,omitempty"`
}

type ChatSession struct {
	User1ID    int64     `json:"user1_id"`
	User2ID    int64     `json:"user2_id"`
	User1Coins int       `json:"user1_coins"`
	User2Coins int       `json:"user2_coins"`
	StartedAt  time.Time `json:"started_at"`
}

// EnqueueChat کاربر رو به صف اضافه میکنه
func (m *Manager) EnqueueChat(ctx context.Context, entry *QueueEntry) error {
	b, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	// ذخیره filter برای cancel بعداً
	pipe := m.rdb.TxPipeline()
	pipe.RPush(ctx, chatQueueKey(entry.Filter), b)
	pipe.Expire(ctx, chatQueueKey(entry.Filter), chatQueueTTL)
	pipe.Set(ctx, chatWaitingKey(entry.TelegramID), entry.Filter, chatQueueTTL)
	_, err = pipe.Exec(ctx)
	return err
}

func (m *Manager) RemoveFromQueue(ctx context.Context, userID int64, filter string) error {
	key := chatQueueKey(filter)
	items, err := m.rdb.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return err
	}
	for _, item := range items {
		var e QueueEntry
		if json.Unmarshal([]byte(item), &e) == nil && e.TelegramID == userID {
			m.rdb.LRem(ctx, key, 1, item)
			break
		}
	}
	return m.rdb.Del(ctx, chatWaitingKey(userID)).Err()
}

func (m *Manager) GetWaitingFilter(ctx context.Context, userID int64) (string, error) {
	return m.rdb.Get(ctx, chatWaitingKey(userID)).Result()
}

func (m *Manager) SetActiveChat(ctx context.Context, cs *ChatSession) error {
	b, _ := json.Marshal(cs)
	pipe := m.rdb.TxPipeline()
	pipe.Set(ctx, activeChatKey(cs.User1ID), b, activeChatTTL)
	pipe.Set(ctx, activeChatKey(cs.User2ID), b, activeChatTTL)
	_, err := pipe.Exec(ctx)
	return err
}

func (m *Manager) GetActiveChat(ctx context.Context, userID int64) (*ChatSession, error) {
	val, err := m.rdb.Get(ctx, activeChatKey(userID)).Result()
	if err != nil {
		return nil, err
	}

	var cs ChatSession
	if err := json.Unmarshal([]byte(val), &cs); err == nil {
		return &cs, nil
	}

	partnerID, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("active chat parse: %w", err)
	}

	return &ChatSession{User1ID: userID, User2ID: partnerID, StartedAt: time.Now()}, nil
}

func (m *Manager) DeleteActiveChat(ctx context.Context, userID int64) error {
	cs, err := m.GetActiveChat(ctx, userID)
	if err != nil {
		return m.rdb.Del(ctx, activeChatKey(userID)).Err()
	}
	pipe := m.rdb.TxPipeline()
	pipe.Del(ctx, activeChatKey(cs.User1ID))
	pipe.Del(ctx, activeChatKey(cs.User2ID))
	_, err = pipe.Exec(ctx)
	return err
}

func chatQueueKey(filter string) string { return "chat:queue:" + filter }
func chatWaitingKey(id int64) string    { return fmt.Sprintf("chat:waiting:%d", id) }

func (m *Manager) GetQueueEntry(ctx context.Context, userID int64, filter string) (*QueueEntry, error) {
	items, err := m.rdb.LRange(ctx, chatQueueKey(filter), 0, -1).Result()
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		var e QueueEntry
		if json.Unmarshal([]byte(item), &e) == nil && e.TelegramID == userID {
			return &e, nil
		}
	}
	return nil, nil
}

// BLPop برای worker
func (m *Manager) BLPop(ctx context.Context, timeout time.Duration, keys ...string) ([]string, error) {
	return m.rdb.BLPop(ctx, timeout, keys...).Result()
}

// RemoveWaitingState فقط waiting key رو پاک میکنه (بدون دست زدن به صف)
func (m *Manager) RemoveWaitingState(ctx context.Context, userID int64) error {
	return m.rdb.Del(ctx, chatWaitingKey(userID)).Err()
}
