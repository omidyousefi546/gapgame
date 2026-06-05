package user

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SearchQueryBuilder builds complex user search queries
type SearchQueryBuilder struct {
	db   *gorm.DB
	ctx  context.Context
	log  *zap.Logger
	q    *gorm.DB
	errs []error
}

// NewSearchQueryBuilder creates a new search query builder
func NewSearchQueryBuilder(db *gorm.DB, ctx context.Context, log *zap.Logger) *SearchQueryBuilder {
	return &SearchQueryBuilder{
		db:   db,
		ctx:  ctx,
		log:  log,
		q:    db.WithContext(ctx),
		errs: make([]error, 0),
	}
}

// WithBasicFilters applies basic user filters (common to all searches)
func (sqb *SearchQueryBuilder) WithBasicFilters(excludeUserID int64) *SearchQueryBuilder {
	// Exclude self
	sqb.q = sqb.q.Where("telegram_id != ?", excludeUserID)

	// Only complete profiles
	sqb.q = sqb.q.Where("profile_state = ?", StateComplete)

	// Only users active in the last 30 days (more reasonable for searches)
	sqb.q = sqb.q.Where("last_seen_at > ?", time.Now().Add(-30*24*time.Hour))

	return sqb
}

// WithGender filters by gender
func (sqb *SearchQueryBuilder) WithGender(gender string) *SearchQueryBuilder {
	if gender == "" || gender == "all" {
		return sqb
	}

	if gender != string(Male) && gender != string(Female) {
		sqb.errs = append(sqb.errs, fmt.Errorf("invalid gender: %s", gender))
		return sqb
	}

	sqb.q = sqb.q.Where("gender = ?", gender)
	return sqb
}

// WithAgeRange filters by age range
func (sqb *SearchQueryBuilder) WithAgeRange(minAge, maxAge int) *SearchQueryBuilder {
	if minAge > 0 && maxAge > 0 {
		if minAge > maxAge {
			sqb.errs = append(sqb.errs, errors.New("minAge cannot be greater than maxAge"))
			return sqb
		}
		sqb.q = sqb.q.Where("age BETWEEN ? AND ?", minAge, maxAge)
	} else if minAge > 0 {
		sqb.q = sqb.q.Where("age >= ?", minAge)
	} else if maxAge > 0 {
		sqb.q = sqb.q.Where("age <= ?", maxAge)
	}

	return sqb
}

// WithProvince filters by single or multiple provinces
func (sqb *SearchQueryBuilder) WithProvince(provinces ...string) *SearchQueryBuilder {
	if len(provinces) == 0 {
		return sqb
	}

	validProvinces := make([]string, 0, len(provinces))
	for _, p := range provinces {
		if p != "" {
			validProvinces = append(validProvinces, p)
		}
	}

	if len(validProvinces) == 0 {
		return sqb
	}

	if len(validProvinces) == 1 {
		sqb.q = sqb.q.Where("province = ?", validProvinces[0])
	} else {
		sqb.q = sqb.q.Where("province IN ?", validProvinces)
	}

	return sqb
}

// WithCity filters by single or multiple cities
func (sqb *SearchQueryBuilder) WithCity(cities ...string) *SearchQueryBuilder {
	if len(cities) == 0 {
		return sqb
	}

	validCities := make([]string, 0, len(cities))
	for _, c := range cities {
		if c != "" {
			validCities = append(validCities, c)
		}
	}

	if len(validCities) == 0 {
		return sqb
	}

	if len(validCities) == 1 {
		sqb.q = sqb.q.Where("city = ?", validCities[0])
	} else {
		sqb.q = sqb.q.Where("city IN ?", validCities)
	}

	return sqb
}

// WithNewUsers filters only users created recently
func (sqb *SearchQueryBuilder) WithNewUsers(hoursBack int) *SearchQueryBuilder {
	if hoursBack > 0 {
		sqb.q = sqb.q.Where("created_at > ?", time.Now().Add(-time.Duration(hoursBack)*time.Hour))
	}
	return sqb
}

// WithoutBlocked excludes users blocked by the searcher
func (sqb *SearchQueryBuilder) WithoutBlocked(blockerID int64) *SearchQueryBuilder {
	sqb.q = sqb.q.Where("telegram_id NOT IN (SELECT blocked_id FROM blocks WHERE blocker_id = ?)", blockerID)
	return sqb
}

// WithHasGPS filters only users with GPS coordinates
func (sqb *SearchQueryBuilder) WithHasGPS() *SearchQueryBuilder {
	sqb.q = sqb.q.Where("latitude IS NOT NULL AND longitude IS NOT NULL")
	return sqb
}

// WithoutChats filters users who haven't chatted before (or long time ago)
func (sqb *SearchQueryBuilder) WithoutChats(daysBack int) *SearchQueryBuilder {
	if daysBack > 0 {
		sqb.q = sqb.q.Where("(last_seen_at IS NULL OR last_seen_at < ?)",
			time.Now().Add(-time.Duration(daysBack)*24*time.Hour))
	}
	return sqb
}

// OrderByLastSeen orders results by last activity
func (sqb *SearchQueryBuilder) OrderByLastSeen() *SearchQueryBuilder {
	sqb.q = sqb.q.Order("last_seen_at DESC")
	return sqb
}

// OrderByLikes orders results by number of likes
func (sqb *SearchQueryBuilder) OrderByLikes() *SearchQueryBuilder {
	sqb.q = sqb.q.Order("likes DESC, last_seen_at DESC")
	return sqb
}

// OrderByNewest orders results by creation date (newest first)
func (sqb *SearchQueryBuilder) OrderByNewest() *SearchQueryBuilder {
	sqb.q = sqb.q.Order("created_at DESC")
	return sqb
}

// Paginate applies limit and offset
func (sqb *SearchQueryBuilder) Paginate(offset, limit int) *SearchQueryBuilder {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100 // Max 100 per request
	}
	if offset < 0 {
		offset = 0
	}

	sqb.q = sqb.q.Offset(offset).Limit(limit)
	return sqb
}

// SelectFields selects specific columns to return
func (sqb *SearchQueryBuilder) SelectFields() *SearchQueryBuilder {
	sqb.q = sqb.q.Select(
		"id, telegram_id, name, gender, age, city, province, longitude, latitude, likes, last_seen_at, created_at",
	)
	return sqb
}

// Build executes the query and returns results
func (sqb *SearchQueryBuilder) Build() ([]User, error) {
	if len(sqb.errs) > 0 {
		return nil, sqb.errs[0]
	}

	var users []User
	if err := sqb.q.Find(&users).Error; err != nil {
		sqb.log.Error("search query failed",
			zap.Error(err),
		)
		return nil, fmt.Errorf("search failed: %w", err)
	}

	return users, nil
}

// BuildWithTotal executes the query and returns results with total count
func (sqb *SearchQueryBuilder) BuildWithTotal() ([]User, int64, error) {
	if len(sqb.errs) > 0 {
		return nil, 0, sqb.errs[0]
	}

	var total int64
	var users []User

	// Get total count before pagination
	// IMPORTANT: Must use Model(&User{}) to specify table and preserve WHERE clauses
	if err := sqb.q.Model(&User{}).Count(&total).Error; err != nil {
		if sqb.log != nil {
			sqb.log.Error("count query failed", zap.Error(err))
		}
		return nil, 0, fmt.Errorf("count failed: %w", err)
	}

	if err := sqb.q.Find(&users).Error; err != nil {
		if sqb.log != nil {
			sqb.log.Error("search query failed", zap.Error(err))
		}
		return nil, 0, fmt.Errorf("search failed: %w", err)
	}

	return users, total, nil
}

// GeoSearchResult represents a user with distance information
type GeoSearchResult struct {
	User     User
	Distance float64 // in kilometers
}

// NearbySearch performs a geographic-based search
func (r *Repository) NearbySearch(
	ctx context.Context,
	userLat, userLng float64,
	radiusKM float64,
	excludeUserID int64,
	maxResults int,
) ([]GeoSearchResult, error) {
	// Calculate bounding box for initial query (more efficient)
	latDelta := radiusKM / 111.0
	lngDelta := radiusKM / (111.0 * math.Cos(userLat*math.Pi/180))

	var candidates []User

	// Get candidates within bounding box
	q := r.db.WithContext(ctx).
		Where("telegram_id != ?", excludeUserID).
		Where("profile_state = ?", StateComplete).
		Where("last_seen_at > ?", time.Now().Add(-6*24*time.Hour)).
		Where("latitude IS NOT NULL AND longitude IS NOT NULL").
		Where("latitude BETWEEN ? AND ?", userLat-latDelta, userLat+latDelta).
		Where("longitude BETWEEN ? AND ?", userLng-lngDelta, userLng+lngDelta).
		Where("telegram_id NOT IN (SELECT blocked_id FROM blocks WHERE blocker_id = ?)", excludeUserID).
		Limit(maxResults * 2). // Get more than needed for filtering
		Find(&candidates)

	if q.Error != nil {
		return nil, fmt.Errorf("nearby search failed: %w", q.Error)
	}

	// Calculate exact distances and filter
	results := make([]GeoSearchResult, 0, len(candidates))
	for _, u := range candidates {
		if u.Latitude == nil || u.Longitude == nil {
			continue
		}

		distance := calcDistance(userLat, userLng, *u.Latitude, *u.Longitude)
		if distance <= radiusKM {
			results = append(results, GeoSearchResult{
				User:     u,
				Distance: distance,
			})
		}
	}

	// Sort by distance
	sort.Slice(results, func(i, j int) bool {
		return results[i].Distance < results[j].Distance
	})

	// Return only maxResults
	if len(results) > maxResults {
		results = results[:maxResults]
	}

	return results, nil
}

// NearbySearchAdvanced performs advanced geographic search with additional filters
func (r *Repository) NearbySearchAdvanced(
	ctx context.Context,
	userLat, userLng float64,
	radiusKM float64,
	filter SearchFilter,
) ([]GeoSearchResult, error) {
	// Get candidates within bounding box
	latDelta := radiusKM / 111.0
	lngDelta := radiusKM / (111.0 * math.Cos(userLat*math.Pi/180))

	sqb := NewSearchQueryBuilder(r.db, ctx, nil).
		SelectFields().
		WithBasicFilters(filter.ExcludeUserID).
		WithoutBlocked(filter.ExcludeUserID).
		WithHasGPS()

	// Apply geographic bounds
	sqb.q = sqb.q.
		Where("latitude BETWEEN ? AND ?", userLat-latDelta, userLat+latDelta).
		Where("longitude BETWEEN ? AND ?", userLng-lngDelta, userLng+lngDelta)

	// Apply additional filters
	if filter.Gender != "" && filter.Gender != "all" {
		sqb = sqb.WithGender(filter.Gender)
	}

	if filter.MinAge > 0 || filter.MaxAge > 0 {
		sqb = sqb.WithAgeRange(filter.MinAge, filter.MaxAge)
	}

	if len(filter.Provinces) > 0 {
		sqb = sqb.WithProvince(filter.Provinces...)
	}

	filter.Limit = filter.Limit * 2 // Get more candidates for distance filtering

	// Get candidates
	candidates, err := sqb.Build()
	if err != nil {
		return nil, err
	}

	// Calculate exact distances and filter
	results := make([]GeoSearchResult, 0, len(candidates))
	for _, u := range candidates {
		if u.Latitude == nil || u.Longitude == nil {
			continue
		}

		distance := calcDistance(userLat, userLng, *u.Latitude, *u.Longitude)
		if distance <= radiusKM {
			results = append(results, GeoSearchResult{
				User:     u,
				Distance: distance,
			})
		}
	}

	// Sort by distance
	sort.Slice(results, func(i, j int) bool {
		return results[i].Distance < results[j].Distance
	})

	// Apply pagination
	start := filter.Offset
	if start >= len(results) {
		return []GeoSearchResult{}, nil
	}

	end := start + filter.Limit
	if end > len(results) {
		end = len(results)
	}

	return results[start:end], nil
}

// ExecuteSearch executes a complex search with all filters applied correctly
func (r *Repository) ExecuteSearch(ctx context.Context, f SearchFilter) ([]User, int64, error) {
	sqb := NewSearchQueryBuilder(r.db, ctx, nil).
		SelectFields().
		WithBasicFilters(f.ExcludeUserID).
		WithoutBlocked(f.ExcludeUserID)

	// Apply optional filters
	if f.Gender != "" && f.Gender != "all" {
		sqb = sqb.WithGender(f.Gender)
	}

	if f.MinAge > 0 || f.MaxAge > 0 {
		sqb = sqb.WithAgeRange(f.MinAge, f.MaxAge)
	}

	if len(f.Provinces) > 0 {
		sqb = sqb.WithProvince(f.Provinces...)
	}

	if f.NewUsers {
		sqb = sqb.WithNewUsers(1) // Users created in last 1 hour
	}

	// Apply sorting
	switch f.Type {
	case "newest":
		sqb = sqb.OrderByNewest()
	case "popular":
		sqb = sqb.OrderByLikes()
	default:
		sqb = sqb.OrderByLastSeen()
	}

	// Apply pagination
	sqb = sqb.Paginate(f.Offset, f.Limit)
	return sqb.BuildWithTotal()
}
