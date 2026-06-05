package matching

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

type QueueItem struct {
	UserID    string    `json:"user_id"`
	Gender    string    `json:"gender"`
	Filters   Filters   `json:"filters"`
	Timestamp time.Time `json:"timestamp"`
}

type Filters struct {
	TargetGender string   `json:"target_gender"`
	MaxDistance  *float64 `json:"max_distance,omitempty"`
	MinAge       *int     `json:"min_age,omitempty"`
	MaxAge       *int     `json:"max_age,omitempty"`
	Province     string   `json:"province,omitempty"`
	City         string   `json:"city,omitempty"`
}

type Queue struct {
	rdb *redis.Client
	key string
}

func NewQueue(rdb *redis.Client) *Queue {
	return &Queue{
		rdb: rdb,
		key: "matching:queue",
	}
}

func (q *Queue) Enqueue(ctx context.Context, item *QueueItem) error {
	data, err := json.Marshal(item)
	if err != nil {
		return err
	}

	score := float64(item.Timestamp.Unix())
	return q.rdb.ZAdd(ctx, q.key, redis.Z{
		Score:  score,
		Member: data,
	}).Err()
}

func (q *Queue) GetAll(ctx context.Context) ([]*QueueItem, error) {
	results, err := q.rdb.ZRange(ctx, q.key, 0, -1).Result()
	if err != nil {
		return nil, err
	}

	items := make([]*QueueItem, 0, len(results))
	for _, r := range results {
		var item QueueItem
		if err := json.Unmarshal([]byte(r), &item); err != nil {
			continue
		}
		items = append(items, &item)
	}

	return items, nil
}

func (q *Queue) Remove(ctx context.Context, userID string) error {
	items, _ := q.GetAll(ctx)
	for _, item := range items {
		if item.UserID == userID {
			data, _ := json.Marshal(item)
			return q.rdb.ZRem(ctx, q.key, data).Err()
		}
	}
	return nil
}

func (q *Queue) RemoveExpired(ctx context.Context, timeout time.Duration) ([]string, error) {
	cutoff := time.Now().Add(-timeout).Unix()

	items, _ := q.GetAll(ctx)
	expired := []string{}

	for _, item := range items {
		if item.Timestamp.Unix() < cutoff {
			expired = append(expired, item.UserID)
			data, _ := json.Marshal(item)
			q.rdb.ZRem(ctx, q.key, data)
		}
	}

	return expired, nil
}
