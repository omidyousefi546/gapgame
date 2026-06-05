package user

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// CachedRepository wraps the regular repository with caching capabilities
type CachedRepository struct {
	repo  *Repository
	cache *redis.Client
	log   *zap.Logger
	ttl   time.Duration
}

// NewCachedRepository creates a new cached repository
func NewCachedRepository(repo *Repository, cache *redis.Client, log *zap.Logger) *CachedRepository {
	return &CachedRepository{
		repo:  repo,
		cache: cache,
		log:   log,
		ttl:   5 * time.Minute,
	}
}

// GetByTelegramID retrieves a user with caching
func (cr *CachedRepository) GetByTelegramID(ctx context.Context, telegramID int64) (*User, error) {
	cacheKey := fmt.Sprintf("user:tg:%d", telegramID)

	// Try to get from cache
	cached, err := cr.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		var u User
		if err := json.Unmarshal([]byte(cached), &u); err == nil {
			return &u, nil
		}
	}

	// Get from database
	u, err := cr.repo.GetByTelegramID(ctx, telegramID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if data, err := json.Marshal(u); err == nil {
		cr.cache.Set(ctx, cacheKey, data, cr.ttl)
	}

	return u, nil
}

// GetByID retrieves a user by ID with caching
func (cr *CachedRepository) GetByID(ctx context.Context, id string) (*User, error) {
	cacheKey := fmt.Sprintf("user:id:%s", id)

	// Try to get from cache
	cached, err := cr.cache.Get(ctx, cacheKey).Result()
	if err == nil {
		var u User
		if err := json.Unmarshal([]byte(cached), &u); err == nil {
			return &u, nil
		}
	}

	// Get from database
	u, err := cr.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if data, err := json.Marshal(u); err == nil {
		cr.cache.Set(ctx, cacheKey, data, cr.ttl)
	}

	return u, nil
}

// Update invalidates cache after updating
func (cr *CachedRepository) Update(ctx context.Context, u *User) error {
	err := cr.repo.Update(ctx, u)
	if err == nil {
		// Invalidate cache
		cacheKeys := []string{
			fmt.Sprintf("user:tg:%d", u.TelegramID),
			fmt.Sprintf("user:id:%s", u.ID),
		}
		cr.cache.Del(ctx, cacheKeys...)
	}
	return err
}

// Create caches the new user
func (cr *CachedRepository) Create(ctx context.Context, u *User) error {
	err := cr.repo.Create(ctx, u)
	if err == nil && u.ID != "" {
		if data, err := json.Marshal(u); err == nil {
			cacheKey := fmt.Sprintf("user:id:%s", u.ID)
			cr.cache.Set(ctx, cacheKey, data, cr.ttl)
		}
	}
	return err
}

// InvalidateUserCache removes a user from cache
func (cr *CachedRepository) InvalidateUserCache(ctx context.Context, id string, telegramID int64) error {
	cacheKeys := []string{
		fmt.Sprintf("user:id:%s", id),
		fmt.Sprintf("user:tg:%d", telegramID),
	}
	return cr.cache.Del(ctx, cacheKeys...).Err()
}

// QueryBuilder provides optimized query building for complex queries
type QueryBuilder struct {
	db *gorm.DB
	q  *gorm.DB
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder(db *gorm.DB) *QueryBuilder {
	return &QueryBuilder{
		db: db,
		q:  db,
	}
}

// WithContext sets the context for the query
func (qb *QueryBuilder) WithContext(ctx context.Context) *QueryBuilder {
	qb.q = qb.db.WithContext(ctx)
	return qb
}

// ExcludeUser excludes a specific user
func (qb *QueryBuilder) ExcludeUser(telegramID int64) *QueryBuilder {
	qb.q = qb.q.Where("telegram_id != ?", telegramID)
	return qb
}

// ProfileComplete only selects users with complete profiles
func (qb *QueryBuilder) ProfileComplete() *QueryBuilder {
	qb.q = qb.q.Where("profile_state = ?", StateComplete)
	return qb
}

// RecentlyActive selects users active in the last N hours
func (qb *QueryBuilder) RecentlyActive(hours int) *QueryBuilder {
	qb.q = qb.q.Where("last_seen_at > ?", time.Now().Add(-time.Duration(hours)*time.Hour))
	return qb
}

// GenderFilter applies gender filter
func (qb *QueryBuilder) GenderFilter(gender string) *QueryBuilder {
	if gender != "" && gender != "all" {
		qb.q = qb.q.Where("gender = ?", gender)
	}
	return qb
}

// ProvinceFilter applies province filter
func (qb *QueryBuilder) ProvinceFilter(province string) *QueryBuilder {
	if province != "" {
		qb.q = qb.q.Where("province = ?", province)
	}
	return qb
}

// AgeRange filters by age range
func (qb *QueryBuilder) AgeRange(minAge, maxAge int) *QueryBuilder {
	qb.q = qb.q.Where("age BETWEEN ? AND ?", minAge, maxAge)
	return qb
}

// OrderByLastSeen orders results by last seen time
func (qb *QueryBuilder) OrderByLastSeen() *QueryBuilder {
	qb.q = qb.q.Order("last_seen_at DESC")
	return qb
}

// Paginate applies limit and offset
func (qb *QueryBuilder) Paginate(offset, limit int) *QueryBuilder {
	qb.q = qb.q.Offset(offset).Limit(limit)
	return qb
}

// SelectColumns selects specific columns
func (qb *QueryBuilder) SelectColumns(columns ...string) *QueryBuilder {
	qb.q = qb.q.Select(columns)
	return qb
}

// Build executes the query and returns results
func (qb *QueryBuilder) Build() ([]User, error) {
	var users []User
	err := qb.q.Find(&users).Error
	return users, err
}

// Count returns the count of matching records
func (qb *QueryBuilder) Count() (int64, error) {
	var count int64
	err := qb.q.Model(&User{}).Count(&count).Error
	return count, err
}

// First returns the first matching record
func (qb *QueryBuilder) First() (*User, error) {
	var u User
	err := qb.q.First(&u).Error
	return &u, err
}

// RepositoryOptimizations provides performance-optimized repository methods
type RepositoryOptimizations struct {
	repo *Repository
}

// NewRepositoryOptimizations creates a new optimizations helper
func NewRepositoryOptimizations(repo *Repository) *RepositoryOptimizations {
	return &RepositoryOptimizations{repo: repo}
}

// BatchGetByTelegramIDs retrieves multiple users efficiently
func (ro *RepositoryOptimizations) BatchGetByTelegramIDs(ctx context.Context, ids []int64) ([]User, error) {
	var users []User
	err := ro.repo.db.WithContext(ctx).
		Where("telegram_id IN ?", ids).
		Find(&users).Error
	return users, err
}

// BulkUpdateCoins updates coins for multiple users efficiently
func (ro *RepositoryOptimizations) BulkUpdateCoins(ctx context.Context, updates map[int64]int) error {
	return ro.repo.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for telegramID, coinsChange := range updates {
			if err := tx.Model(&User{}).
				Where("telegram_id = ?", telegramID).
				Update("coins", gorm.Expr("coins + ?", coinsChange)).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// BulkUpdateLastSeen updates last_seen for multiple users efficiently
func (ro *RepositoryOptimizations) BulkUpdateLastSeen(ctx context.Context, ids []int64, timestamp time.Time) error {
	return ro.repo.db.WithContext(ctx).
		Model(&User{}).
		Where("telegram_id IN ?", ids).
		Update("last_seen_at", timestamp).Error
}

// DeleteInactiveUsers removes users inactive for N days
func (ro *RepositoryOptimizations) DeleteInactiveUsers(ctx context.Context, days int) (int64, error) {
	cutoff := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	result := ro.repo.db.WithContext(ctx).
		Where("last_seen_at < ?", cutoff).
		Delete(&User{})
	return result.RowsAffected, result.Error
}

// GetUserStats returns aggregate statistics
func (ro *RepositoryOptimizations) GetUserStats(ctx context.Context) (map[string]interface{}, error) {
	var stats struct {
		TotalUsers  int64
		ActiveUsers int64
		MaleCount   int64
		FemaleCount int64
		AverageAge  float64
		TotalCoins  int64
	}

	err := ro.repo.db.WithContext(ctx).Model(&User{}).
		Select(
			"COUNT(*) as total_users",
			"COUNT(CASE WHEN last_seen_at > ? THEN 1 END) as active_users",
			"COUNT(CASE WHEN gender = 'male' THEN 1 END) as male_count",
			"COUNT(CASE WHEN gender = 'female' THEN 1 END) as female_count",
			"AVG(age) as average_age",
			"SUM(coins) as total_coins",
		).
		Where("last_seen_at > ?", time.Now().Add(-30*24*time.Hour)).
		Scan(&stats).Error

	result := map[string]interface{}{
		"total_users":  stats.TotalUsers,
		"active_users": stats.ActiveUsers,
		"male_count":   stats.MaleCount,
		"female_count": stats.FemaleCount,
		"average_age":  stats.AverageAge,
		"total_coins":  stats.TotalCoins,
	}

	return result, err
}
