package session

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache provides a Redis-based cache implementation
type RedisCache struct {
	rdb *redis.Client
}

// NewRedisCache creates a new Redis cache provider
func NewRedisCache(rdb *redis.Client) *RedisCache {
	return &RedisCache{rdb: rdb}
}

// Get retrieves a value from cache
func (rc *RedisCache) Get(ctx context.Context, key string) (string, error) {
	return rc.rdb.Get(ctx, key).Result()
}

// Set stores a value in cache with TTL
func (rc *RedisCache) Set(ctx context.Context, key string, value string, ttl time.Duration) error {
	return rc.rdb.Set(ctx, key, value, ttl).Err()
}

// Delete removes one or more keys from cache
func (rc *RedisCache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return rc.rdb.Del(ctx, keys...).Err()
}

// GetList retrieves all keys matching a pattern
func (rc *RedisCache) GetList(ctx context.Context, pattern string) ([]string, error) {
	var keys []string
	var cursor uint64

	for {
		keysSlice, nextCursor, err := rc.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, err
		}

		keys = append(keys, keysSlice...)

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return keys, nil
}

// GetPattern retrieves all key-value pairs matching a pattern
func (rc *RedisCache) GetPattern(ctx context.Context, pattern string) (map[string]string, error) {
	result := make(map[string]string)

	keys, err := rc.GetList(ctx, pattern)
	if err != nil {
		return nil, err
	}

	if len(keys) == 0 {
		return result, nil
	}

	values, err := rc.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	for i, key := range keys {
		if i < len(values) && values[i] != nil {
			result[key] = fmt.Sprintf("%v", values[i])
		}
	}

	return result, nil
}

// Exists checks if a key exists
func (rc *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	count, err := rc.rdb.Exists(ctx, key).Result()
	return count > 0, err
}

// IncrementInt increments an integer value
func (rc *RedisCache) IncrementInt(ctx context.Context, key string) (int64, error) {
	return rc.rdb.Incr(ctx, key).Result()
}

// DecrementInt decrements an integer value
func (rc *RedisCache) DecrementInt(ctx context.Context, key string) (int64, error) {
	return rc.rdb.Decr(ctx, key).Result()
}

// Expire sets expiration on a key
func (rc *RedisCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return rc.rdb.Expire(ctx, key, ttl).Err()
}

// GetWithPrefix gets multiple keys with a specific prefix and returns a map
func (rc *RedisCache) GetWithPrefix(ctx context.Context, prefix string) (map[string]string, error) {
	return rc.GetPattern(ctx, prefix+"*")
}

// DeleteWithPrefix deletes all keys with a specific prefix
func (rc *RedisCache) DeleteWithPrefix(ctx context.Context, prefix string) error {
	keys, err := rc.GetList(ctx, prefix+"*")
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		return nil
	}

	return rc.Delete(ctx, keys...)
}

// SetNX sets a value only if the key doesn't exist
func (rc *RedisCache) SetNX(ctx context.Context, key string, value string, ttl time.Duration) (bool, error) {
	return rc.rdb.SetNX(ctx, key, value, ttl).Result()
}

// GetPrefix retrieves keys with a specific prefix
func (rc *RedisCache) GetPrefix(ctx context.Context, prefix string) ([]string, error) {
	var keys []string

	iter := rc.rdb.Scan(ctx, 0, prefix+"*", 0).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	return keys, iter.Err()
}

// ListAdd adds values to a Redis list
func (rc *RedisCache) ListAdd(ctx context.Context, key string, values ...string) error {
	if len(values) == 0 {
		return nil
	}
	args := make([]interface{}, len(values))
	for i, v := range values {
		args[i] = v
	}
	return rc.rdb.RPush(ctx, key, args...).Err()
}

// ListRemove removes a value from a Redis list
func (rc *RedisCache) ListRemove(ctx context.Context, key string, value string) error {
	return rc.rdb.LRem(ctx, key, 1, value).Err()
}

// ListGetAll retrieves all values from a Redis list
func (rc *RedisCache) ListGetAll(ctx context.Context, key string) ([]string, error) {
	return rc.rdb.LRange(ctx, key, 0, -1).Result()
}

// ListLen returns the length of a Redis list
func (rc *RedisCache) ListLen(ctx context.Context, key string) (int64, error) {
	return rc.rdb.LLen(ctx, key).Result()
}

// SetAdd adds a member to a Redis set
func (rc *RedisCache) SetAdd(ctx context.Context, key string, members ...string) error {
	if len(members) == 0 {
		return nil
	}
	args := make([]interface{}, len(members))
	for i, m := range members {
		args[i] = m
	}
	return rc.rdb.SAdd(ctx, key, args...).Err()
}

// SetMembers returns all members of a Redis set
func (rc *RedisCache) SetMembers(ctx context.Context, key string) ([]string, error) {
	return rc.rdb.SMembers(ctx, key).Result()
}

// IsMember checks if a member exists in a Redis set
func (rc *RedisCache) IsMember(ctx context.Context, key string, member string) (bool, error) {
	return rc.rdb.SIsMember(ctx, key, member).Result()
}

// ClearPattern clears all keys matching a pattern with a limit to prevent memory issues
func (rc *RedisCache) ClearPattern(ctx context.Context, pattern string, maxKeys int64) error {
	iter := rc.rdb.Scan(ctx, 0, pattern, maxKeys).Iterator()
	var keys []string

	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return err
	}

	if len(keys) == 0 {
		return nil
	}

	return rc.Delete(ctx, keys...)
}

// CleanExpiredKeys removes keys that match a pattern and contain a specific string (helper for cleanup)
func (rc *RedisCache) CleanExpiredKeys(ctx context.Context, pattern string) (int, error) {
	iter := rc.rdb.Scan(ctx, 0, pattern, 0).Iterator()
	var deleted int

	for iter.Next(ctx) {
		key := iter.Val()
		ttl, err := rc.rdb.TTL(ctx, key).Result()
		if err != nil {
			continue
		}

		// If TTL is -1 (no expiration), remove it for matching patterns that should expire
		if ttl == -1 && strings.Contains(key, "temp") {
			rc.Delete(ctx, key)
			deleted++
		}
	}

	return deleted, iter.Err()
}
