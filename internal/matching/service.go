package matching

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MatchingService handles efficient user matching
type MatchingService struct {
	db    *gorm.DB
	cache CacheProvider
	log   *zap.Logger
}

// CacheProvider interface for caching user data
type CacheProvider interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
	GetList(ctx context.Context, pattern string) ([]string, error)
}

// MatchRequest represents a matching request
type MatchRequest struct {
	UserID       string
	TelegramID   int64
	Gender       string
	Age          int
	Province     string
	City         string
	Latitude     *float64
	Longitude    *float64
	TargetGender string
	MaxDistance  *float64
	MinAge       *int
	MaxAge       *int
	RequestedAt  time.Time
}

// MatchResult contains two matched users
type MatchResult struct {
	User1ID   string
	User2ID   string
	User1TgID int64
	User2TgID int64
	MatchedAt time.Time
}

func NewMatchingService(db *gorm.DB, cache CacheProvider, log *zap.Logger) *MatchingService {
	return &MatchingService{
		db:    db,
		cache: cache,
		log:   log,
	}
}

// EnqueueForMatching adds a user to matching queue
func (ms *MatchingService) EnqueueForMatching(ctx context.Context, req *MatchRequest) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	// Store in cache with timeout
	key := fmt.Sprintf("match:queue:%s", req.UserID)
	return ms.cache.Set(ctx, key, string(data), 15*time.Minute)
}

// FindMatch attempts to find a match for a user
func (ms *MatchingService) FindMatch(ctx context.Context, req *MatchRequest) (*MatchResult, error) {
	// Get all waiting users
	pattern := "match:queue:*"
	keys, err := ms.cache.GetList(ctx, pattern)
	if err != nil {
		ms.log.Error("failed to get queue keys", zap.Error(err))
		return nil, err
	}

	for _, key := range keys {
		data, err := ms.cache.Get(ctx, key)
		if err != nil {
			continue
		}

		var candidate MatchRequest
		if err := json.Unmarshal([]byte(data), &candidate); err != nil {
			continue
		}

		// Skip self and already matched
		if candidate.UserID == req.UserID || candidate.TelegramID == req.TelegramID {
			continue
		}

		// Check if they match
		if ms.isCompatible(req, &candidate) {
			// Remove both from queue
			reqKey := fmt.Sprintf("match:queue:%s", req.UserID)
			candidateKey := fmt.Sprintf("match:queue:%s", candidate.UserID)
			ms.cache.Delete(ctx, reqKey, candidateKey)

			return &MatchResult{
				User1ID:   req.UserID,
				User2ID:   candidate.UserID,
				User1TgID: req.TelegramID,
				User2TgID: candidate.TelegramID,
				MatchedAt: time.Now(),
			}, nil
		}
	}

	return nil, nil
}

// isCompatible checks if two users are compatible for matching
func (ms *MatchingService) isCompatible(user1, user2 *MatchRequest) bool {
	// Check gender preferences
	if user1.TargetGender != "" && user1.TargetGender != user2.Gender {
		return false
	}
	if user2.TargetGender != "" && user2.TargetGender != user1.Gender {
		return false
	}

	// Check age constraints
	if user1.MinAge != nil && user2.Age < *user1.MinAge {
		return false
	}
	if user1.MaxAge != nil && user2.Age > *user1.MaxAge {
		return false
	}
	if user2.MinAge != nil && user1.Age < *user2.MinAge {
		return false
	}
	if user2.MaxAge != nil && user1.Age > *user2.MaxAge {
		return false
	}

	// Check province (exact match if specified)
	if user1.Province != "" && user1.Province != user2.Province {
		return false
	}
	if user2.Province != "" && user2.Province != user1.Province {
		return false
	}

	// Check distance
	if user1.Latitude != nil && user2.Latitude != nil && user1.MaxDistance != nil {
		distance := haversine(*user1.Latitude, *user1.Longitude, *user2.Latitude, *user2.Longitude)
		if distance > *user1.MaxDistance {
			return false
		}
	}
	if user2.Latitude != nil && user1.Latitude != nil && user2.MaxDistance != nil {
		distance := haversine(*user2.Latitude, *user2.Longitude, *user1.Latitude, *user1.Longitude)
		if distance > *user2.MaxDistance {
			return false
		}
	}

	return true
}

// haversine calculates the great-circle distance between two points
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0 // Earth radius in kilometers

	dLat := toRadians(lat2 - lat1)
	dLon := toRadians(lon2 - lon1)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRadians(lat1))*math.Cos(toRadians(lat2))*
			math.Sin(dLon/2)*math.Sin(dLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

func toRadians(degrees float64) float64 {
	return degrees * math.Pi / 180.0
}

// RemoveFromQueue removes a user from the matching queue
func (ms *MatchingService) RemoveFromQueue(ctx context.Context, userID string) error {
	key := fmt.Sprintf("match:queue:%s", userID)
	return ms.cache.Delete(ctx, key)
}

// CancelMatching cancels a user's matching request
func (ms *MatchingService) CancelMatching(ctx context.Context, userID string) error {
	return ms.RemoveFromQueue(ctx, userID)
}
