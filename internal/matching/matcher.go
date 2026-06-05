package matching

// import (
// 	"context"
// 	"math"
// 	"time"

// 	"GapGame/internal/user"
// )

// type MatchCallback func(u1, u2 *user.User)
// type TimeoutCallback func(userID string)

// type Matcher struct {
// 	queue         *Queue
// 	userService   *user.Service
// 	onMatch       MatchCallback
// 	onTimeout     TimeoutCallback
// 	stopChan      chan struct{}
// 	timeout       time.Duration
// 	checkInterval time.Duration
// }

// func NewMatcher(queue *Queue, userService *user.Service, onMatch MatchCallback, onTimeout TimeoutCallback) *Matcher {
// 	return &Matcher{
// 		queue:         queue,
// 		userService:   userService,
// 		onMatch:       onMatch,
// 		onTimeout:     onTimeout,
// 		stopChan:      make(chan struct{}),
// 		timeout:       5 * time.Minute,
// 		checkInterval: 3 * time.Second,
// 	}
// }

// func (m *Matcher) Start() {
// 	ticker := time.NewTicker(m.checkInterval)
// 	defer ticker.Stop()

// 	for {
// 		select {
// 		case <-ticker.C:
// 			m.processQueue()
// 		case <-m.stopChan:
// 			return
// 		}
// 	}
// }

// func (m *Matcher) Stop() {
// 	close(m.stopChan)
// }

// func (m *Matcher) processQueue() {
// 	ctx := context.Background()

// 	// حذف افراد منقضی شده
// 	expired, _ := m.queue.RemoveExpired(ctx, m.timeout)
// 	for _, userID := range expired {
// 		if m.onTimeout != nil {
// 			m.onTimeout(userID)
// 		}
// 	}

// 	// دریافت لیست صف
// 	items, err := m.queue.GetAll(ctx)
// 	if err != nil || len(items) < 2 {
// 		return
// 	}

// 	// تلاش برای تطبیق
// 	for i := 0; i < len(items); i++ {
// 		for j := i + 1; j < len(items); j++ {
// 			if m.isMatch(items[i], items[j]) {
// 				u1, _ := m.userService.GetByID(items[i].UserID)
// 				u2, _ := m.userService.GetByID(items[j].UserID)

// 				if u1 != nil && u2 != nil {
// 					m.queue.Remove(ctx, items[i].UserID)
// 					m.queue.Remove(ctx, items[j].UserID)

// 					if m.onMatch != nil {
// 						m.onMatch(u1, u2)
// 					}
// 					return
// 				}
// 			}
// 		}
// 	}
// }

// func (m *Matcher) isMatch(item1, item2 *QueueItem) bool {
// 	// بررسی جنسیت
// 	if item1.Filters.TargetGender != "" && item1.Filters.TargetGender != item2.Gender {
// 		return false
// 	}
// 	if item2.Filters.TargetGender != "" && item2.Filters.TargetGender != item1.Gender {
// 		return false
// 	}

// 	u1, _ := m.userService.GetByID(item1.UserID)
// 	u2, _ := m.userService.GetByID(item2.UserID)

// 	if u1 == nil || u2 == nil {
// 		return false
// 	}

// 	// بررسی سن
// 	if item1.Filters.MinAge != nil && u2.Age != nil && *u2.Age < *item1.Filters.MinAge {
// 		return false
// 	}
// 	if item1.Filters.MaxAge != nil && u2.Age != nil && *u2.Age > *item1.Filters.MaxAge {
// 		return false
// 	}
// 	if item2.Filters.MinAge != nil && u1.Age != nil && *u1.Age < *item2.Filters.MinAge {
// 		return false
// 	}
// 	if item2.Filters.MaxAge != nil && u1.Age != nil && *u1.Age > *item2.Filters.MaxAge {
// 		return false
// 	}

// 	// بررسی استان/شهر
// 	if item1.Filters.Province != "" && u2.Province != item1.Filters.Province {
// 		return false
// 	}
// 	if item2.Filters.Province != "" && u1.Province != item2.Filters.Province {
// 		return false
// 	}
// 	if item1.Filters.City != "" && u2.City != item1.Filters.City {
// 		return false
// 	}
// 	if item2.Filters.City != "" && u1.City != item2.Filters.City {
// 		return false
// 	}

// 	// بررسی فاصله جغرافیایی
// 	if item1.Filters.MaxDistance != nil && u1.Latitude != nil && u2.Latitude != nil {
// 		dist := haversine(*u1.Latitude, *u1.Longitude, *u2.Latitude, *u2.Longitude)
// 		if dist > *item1.Filters.MaxDistance {
// 			return false
// 		}
// 	}
// 	if item2.Filters.MaxDistance != nil && u1.Latitude != nil && u2.Latitude != nil {
// 		dist := haversine(*u1.Latitude, *u1.Longitude, *u2.Latitude, *u2.Longitude)
// 		if dist > *item2.Filters.MaxDistance {
// 			return false
// 		}
// 	}

// 	return true
// }

// func haversine(lat1, lon1, lat2, lon2 float64) float64 {
// 	const R = 6371 // شعاع زمین به کیلومتر

// 	dLat := (lat2 - lat1) * math.Pi / 180
// 	dLon := (lon2 - lon1) * math.Pi / 180

// 	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
// 		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
// 			math.Sin(dLon/2)*math.Sin(dLon/2)

// 	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
// 	return R * c
// }
