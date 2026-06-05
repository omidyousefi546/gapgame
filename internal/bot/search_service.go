package bot

import (
	"fmt"

	"GapGame/internal/user"
	"GapGame/internal/utils"
)

// SearchService encapsulates search functionality
type SearchService struct {
	repo *user.Repository
}

// NewSearchService creates a new search service
func NewSearchService(repo *user.Repository) *SearchService {
	return &SearchService{
		repo: repo,
	}
}

// ExecuteSearch performs a search based on search state
func (ss *SearchService) ExecuteSearch(
	myUser *user.User,
	searchType string,
	gender string,
	provinces []string,
	offset, limit int,
) ([]user.User, error) {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()

	filter := user.SearchFilter{
		Type:          searchType,
		Gender:        gender,
		Provinces:     provinces,
		ExcludeUserID: myUser.TelegramID,
		MinAge:        0,
		MaxAge:        0,
		Offset:        offset,
		Limit:         limit,
	}

	// Build filter based on search type
	switch searchType {
	case "age":
		// Search people within 3 years age difference
		age := myUser.SafeAge()
		filter.MinAge = age - 3
		filter.MaxAge = age + 3
		filter.Type = "age"

	case "province":
		// Search people from same province
		filter.Provinces = []string{myUser.Province}
		filter.Type = "province"

	case "new":
		// Search newly joined users (last 1 hour)
		filter.NewUsers = true
		filter.Type = "newest"

	case "nearby":
		// Search people nearby (requires GPS)
		if myUser.Latitude == nil || myUser.Longitude == nil {
			return nil, fmt.Errorf("GPS coordinates not available")
		}
		filter.NearbyLat = myUser.Latitude
		filter.NearbyLng = myUser.Longitude
		filter.RadiusKM = 30
		filter.Type = "nearby"

	case "popular":
		// Search by most liked users
		filter.Type = "popular"

	case "advanced":
		// Advanced search with custom filters
		// Gender and Provinces already set
		filter.Type = "advanced"

	default:
		filter.Type = "random"
	}

	return ss.repo.SearchUsers(ctx, filter)
}

// ExecuteNearbySearch performs a geographic search
func (ss *SearchService) ExecuteNearbySearch(
	myUser *user.User,
	gender string,
	offset, limit int,
) ([]user.GeoSearchResult, error) {
	if myUser.Latitude == nil || myUser.Longitude == nil {
		return nil, fmt.Errorf("GPS coordinates not available")
	}

	ctx, cancel := utils.NewRequestContext()
	defer cancel()

	filter := user.SearchFilter{
		Gender:        gender,
		ExcludeUserID: myUser.TelegramID,
		Offset:        offset,
		Limit:         limit,
		NearbyLat:     myUser.Latitude,
		NearbyLng:     myUser.Longitude,
		RadiusKM:      30,
	}

	return ss.repo.NearbySearchAdvanced(ctx, *myUser.Latitude, *myUser.Longitude, 30, filter)
}

// ValidateSearchRequest validates search parameters
func (ss *SearchService) ValidateSearchRequest(
	searchType, gender string,
	provinces []string,
	myUser *user.User,
) error {
	// Validate search type
	validTypes := map[string]bool{
		"age":      true,
		"province": true,
		"new":      true,
		"nearby":   true,
		"popular":  true,
		"advanced": true,
	}

	if !validTypes[searchType] {
		return fmt.Errorf("invalid search type: %s", searchType)
	}

	// Validate gender if provided
	if gender != "" && gender != "all" {
		if gender != string(user.Male) && gender != string(user.Female) {
			return fmt.Errorf("invalid gender: %s", gender)
		}
	}

	// Check GPS for nearby search
	if searchType == "nearby" {
		if myUser.Latitude == nil || myUser.Longitude == nil {
			return fmt.Errorf("GPS coordinates required for nearby search")
		}
	}

	return nil
}
