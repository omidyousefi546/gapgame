package session

import (
	"GapGame/internal/user"
	"GapGame/internal/utils"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	sessionTTL      = 24 * time.Hour
	redisCmdTimeout = time.Second
)

type Type string

const (
	TypeChat Type = "chat"
	TypeGame Type = "game"
)

type Session struct {
	ID        string    `json:"id"`
	User1ID   string    `json:"user1_id"`
	User2ID   string    `json:"user2_id"`
	Type      Type      `json:"type"`
	CreatedAt time.Time `json:"created_at"`
}

type SearchState struct {
	Type      string   `json:"type"`
	Gender    string   `json:"gender"`
	Provinces []string `json:"provinces"`
	Offset    int      `json:"offset"` // شماره صفحه (نه offset مطلق)

	NearbyLat *float64 `json:"nearby_lat,omitempty"`
	NearbyLng *float64 `json:"nearby_lng,omitempty"`
}

// Manager تمام عملیات session و state رو مدیریت می‌کنه
type Manager struct {
	rdb *redis.Client
}

func NewManager(rdb *redis.Client) *Manager {
	return &Manager{rdb: rdb}
}

// GetRedisClient returns the underlying Redis client
func (m *Manager) GetRedisClient() *redis.Client {
	return m.rdb
}

func (m *Manager) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), redisCmdTimeout)
}

// ─── Session ──────────────────────────────────────────────────

func (m *Manager) CreateSession(user1ID, user2ID string, sessionType Type) (*Session, error) {
	ctx, cancel := m.ctx()
	defer cancel()

	sessionID := makeSessionID(user1ID, user2ID)
	session := &Session{
		ID:        sessionID,
		User1ID:   user1ID,
		User2ID:   user2ID,
		Type:      sessionType,
		CreatedAt: time.Now(),
	}

	data, err := json.Marshal(session)
	if err != nil {
		return nil, fmt.Errorf("CreateSession marshal: %w", err)
	}

	pipe := m.rdb.TxPipeline()
	pipe.Set(ctx, sessionKey(sessionID), data, sessionTTL)
	pipe.Set(ctx, userSessionKey(user1ID), user2ID, sessionTTL)
	pipe.Set(ctx, userSessionKey(user2ID), user1ID, sessionTTL)

	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("CreateSession exec: %w", err)
	}

	return session, nil
}

func (m *Manager) GetSession(sessionID string) (*Session, error) {
	ctx, cancel := m.ctx()
	defer cancel()

	data, err := m.rdb.Get(ctx, sessionKey(sessionID)).Result()
	if err != nil {
		return nil, fmt.Errorf("GetSession: %w", err)
	}

	var session Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, fmt.Errorf("GetSession unmarshal: %w", err)
	}

	return &session, nil
}

func (m *Manager) DeleteSession(sessionID string) error {
	session, err := m.GetSession(sessionID)
	if err != nil {
		return fmt.Errorf("DeleteSession get: %w", err)
	}

	ctx, cancel := m.ctx()
	defer cancel()

	pipe := m.rdb.TxPipeline()
	pipe.Del(ctx, sessionKey(sessionID))
	pipe.Del(ctx, userSessionKey(session.User1ID))
	pipe.Del(ctx, userSessionKey(session.User2ID))

	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("DeleteSession exec: %w", err)
	}
	return nil
}

func (m *Manager) FindPartner(userID string) (string, error) {
	ctx, cancel := m.ctx()
	defer cancel()

	partnerID, err := m.rdb.Get(ctx, userSessionKey(userID)).Result()
	if err != nil {
		return "", fmt.Errorf("FindPartner: %w", err)
	}
	return partnerID, nil
}

// ─── User State ───────────────────────────────────────────────

func (m *Manager) SetUserState(userID int64, state string) error {
	ctx, cancel := m.ctx()
	defer cancel()
	return m.rdb.Set(ctx, userStateKey(userID), state, 0).Err()
}

func (m *Manager) GetUserState(userID int64) (string, error) {
	ctx, cancel := m.ctx()
	defer cancel()
	return m.rdb.Get(ctx, userStateKey(userID)).Result()
}

func (m *Manager) ClearUserState(userID int64) error {
	ctx, cancel := m.ctx()
	defer cancel()
	return m.rdb.Del(ctx, userStateKey(userID)).Err()
}

// ─── DM State ───────────────────────────────────────────────

const dmTargetTTL = 30 * time.Minute

func (m *Manager) SetDMTarget(userID, targetID int64) error {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return m.rdb.Set(ctx, dmTargetKey(userID), targetID, dmTargetTTL).Err()

}

func (m *Manager) GetDMTarget(userID int64) (int64, error) {

	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	val, err := m.rdb.Get(ctx, dmTargetKey(userID)).Result()

	if err != nil {

		return 0, err
	}
	targetID, _ := strconv.ParseInt(val, 10, 64)

	return targetID, nil

}

func (m *Manager) ClearDMTarget(userID int64) error {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return m.rdb.Del(ctx, dmTargetKey(userID)).Err()

}

// contact

// key: contact_pending:{telegramID} -> contactID
func (m *Manager) SetPendingContact(telegramID, contactID int64) error {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return m.rdb.Set(ctx, fmt.Sprintf("contact_pending:%d", telegramID), contactID, 2*time.Minute).Err()
}

func (m *Manager) GetPendingContact(telegramID int64) (int64, error) {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	val, err := m.rdb.Get(ctx, fmt.Sprintf("contact_pending:%d", telegramID)).Int64()
	return val, err
}

func (m *Manager) DelPendingContact(telegramID int64) error {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return m.rdb.Del(ctx, fmt.Sprintf("contact_pending:%d", telegramID)).Err()
}

// ─── Last Seen ────────────────────────────────────────────────

const lastSeenTTL = 7 * 24 * time.Hour
const onlineTTL = 2 * time.Minute

func (m *Manager) UpdateLastSeen(ctx context.Context, userID int64) error {
	pipe := m.rdb.TxPipeline()
	// ذخیره timestamp برای نمایش آخرین بازدید
	pipe.Set(ctx, lastSeenKey(userID), time.Now().Unix(), lastSeenTTL)

	_, err := pipe.Exec(ctx)
	return err
}

func (m *Manager) GetLastSeen(ctx context.Context, userID int64) (time.Time, error) {
	val, err := m.rdb.Get(ctx, lastSeenKey(userID)).Int64()
	if err == redis.Nil {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("GetLastSeen: %w", err)
	}
	return time.Unix(val, 0), nil
}

func (m *Manager) GetAllLastSeen(ctx context.Context) (map[int64]*time.Time, error) {
	var cursor uint64
	result := make(map[int64]*time.Time)

	for {
		keys, nextCursor, err := m.rdb.Scan(ctx, cursor, "last_seen:*", 1000).Result()
		if err != nil {
			return nil, err
		}

		if len(keys) > 0 {
			pipe := m.rdb.Pipeline()
			cmds := make([]*redis.StringCmd, len(keys))
			for i, key := range keys {
				cmds[i] = pipe.Get(ctx, key)
			}
			pipe.Exec(ctx)

			for i, cmd := range cmds {
				val, err := cmd.Result()
				if err != nil {
					continue
				}

				idStr := strings.TrimPrefix(keys[i], "last_seen:")
				id, err := strconv.ParseInt(idStr, 10, 64)
				if err != nil {
					continue
				}

				// Unix timestamp رو parse کن
				timestamp, err := strconv.ParseInt(val, 10, 64)
				if err != nil {
					continue
				}
				t := time.Unix(timestamp, 0)
				result[id] = &t
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return result, nil
}

func (m *Manager) SyncLastSeenStream(
	ctx context.Context,
	repo *user.RepositoryOptimizations,
	logger *zap.Logger,
) error {

	var cursor uint64
	const scanCount = 1000

	for {
		keys, nextCursor, err := m.rdb.Scan(ctx, cursor, "last_seen:*", scanCount).Result()
		if err != nil {
			return err
		}

		if len(keys) > 0 {

			pipe := m.rdb.Pipeline()
			cmds := make([]*redis.StringCmd, len(keys))

			for i, key := range keys {
				cmds[i] = pipe.Get(ctx, key)
			}

			_, err := pipe.Exec(ctx)
			if err != nil {
				logger.Error("redis pipeline exec failed", zap.Error(err))
				return err
			}

			// timestamp -> ids
			grouped := make(map[int64][]int64)

			for i, cmd := range cmds {

				val, err := cmd.Result()
				if err != nil {
					continue
				}

				idStr := strings.TrimPrefix(keys[i], "last_seen:")
				id, err := strconv.ParseInt(idStr, 10, 64)
				if err != nil {
					continue
				}

				timestamp, err := strconv.ParseInt(val, 10, 64)
				if err != nil {
					continue
				}

				grouped[timestamp] = append(grouped[timestamp], id)
			}

			// DB update
			for ts, ids := range grouped {

				t := time.Unix(ts, 0)

				err := repo.BulkUpdateLastSeen(ctx, ids, t)
				if err != nil {
					logger.Error("bulk update failed",
						zap.Int("count", len(ids)),
						zap.Error(err),
					)
					continue
				}
			}

			// delete synced keys
			delPipe := m.rdb.Pipeline()

			for _, key := range keys {
				delPipe.Del(ctx, key)
			}

			_, err = delPipe.Exec(ctx)
			if err != nil {
				logger.Error("redis delete pipeline failed", zap.Error(err))
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return nil
}

// func (m *Manager) IsOnline(ctx context.Context, telegramID int64) bool {
// 	exists, _ := m.rdb.Exists(ctx, onlineKey(telegramID)).Result()
// 	return exists > 0
// }

// ─── Search State ─────────────────────────────────────────────

const searchStateTTL = 10 * time.Minute

func (m *Manager) SetSearchState(userID int64, state *SearchState) error {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	b, err := json.Marshal(*state)
	if err != nil {
		return fmt.Errorf("SetSearchState marshal: %w", err)
	}
	return m.rdb.Set(ctx, searchStateKey(userID), b, searchStateTTL).Err()
}

func (m *Manager) GetSearchState(userID int64) (*SearchState, error) {
	var state SearchState
	ctx, cancel := utils.NewRequestContext()
	defer cancel()

	val, err := m.rdb.Get(ctx, searchStateKey(userID)).Result()
	if err == redis.Nil {
		return &state, nil
	}
	if err != nil {
		return &state, fmt.Errorf("GetSearchState: %w", err)
	}
	if err := json.Unmarshal([]byte(val), &state); err != nil {
		return &state, fmt.Errorf("GetSearchState unmarshal: %w", err)
	}
	return &state, nil
}

func (m *Manager) UpdateSearchOffset(userID int64, offset int) error {

	state, err := m.GetSearchState(userID)
	if err != nil {
		return err
	}
	state.Offset = offset
	return m.SetSearchState(userID, state)
}

// func (m *Manager) UpdateSwipeOffset(userID int64, offset int) error {

// 	state, err := m.GetSearchState(userID)
// 	if err != nil {
// 		return err
// 	}
// 	state.SwipeOffset = offset
// 	return m.SetSearchState(userID, state)
// }

func (m *Manager) ClearSearchState(userID int64) error {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()
	return m.rdb.Del(ctx, searchStateKey(userID)).Err()
}

// ─── Queue Management ─────────────────────────────────────────

const queueTTL = 5 * time.Minute

// IsInQueue checks if user is already in queue
func (m *Manager) IsInQueue(userID int64) (bool, error) {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()

	exists, err := m.rdb.Exists(ctx, inQueueKey(userID)).Result()
	return exists > 0, err
}

// JoinQueue marks user as in queue
func (m *Manager) JoinQueue(userID int64) error {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()

	return m.rdb.Set(ctx, inQueueKey(userID), "1", queueTTL).Err()
}

// LeaveQueue removes user from queue
func (m *Manager) LeaveQueue(userID int64) error {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()

	return m.rdb.Del(ctx, inQueueKey(userID)).Err()
}

// ─── Active Chat Management ──────────────────────────────────

const activeChatSessionTTL = 24 * time.Hour

// StartChat marks user as in active chat
func (m *Manager) StartChat(userID int64, partnerID int64) error {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()

	// Set active chat for both users
	pipe := m.rdb.TxPipeline()
	pipe.Set(ctx, activeChatKey(userID), partnerID, activeChatSessionTTL)
	pipe.Set(ctx, activeChatKey(partnerID), userID, activeChatSessionTTL)

	_, err := pipe.Exec(ctx)
	return err
}

// GetActiveChat returns partner ID if user is in active chat
func (m *Manager) GetActiveChat2(userID int64) (int64, error) {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()

	cs, err := m.GetActiveChat(ctx, userID)
	if err != nil {
		if err == redis.Nil {
			return 0, nil // No active chat
		}
		return 0, err
	}
	if cs == nil {
		return 0, nil
	}
	if cs.User1ID == userID {
		return cs.User2ID, nil
	}
	return cs.User1ID, nil
}

// HasActiveChat checks if user already has active chat
func (m *Manager) HasActiveChat(userID int64) (bool, error) {
	partnerID, err := m.GetActiveChat2(userID)
	if err != nil {
		return false, err
	}
	return partnerID != 0, nil
}

// HasActiveChatSilent checks if user already has active chat ignoring errors
func (m *Manager) HasActiveChatSilent(userID int64) bool {
	has, _ := m.HasActiveChat(userID)
	return has
}

// EndChat removes active chat for user
func (m *Manager) EndChat(userID int64) error {
	ctx, cancel := utils.NewRequestContext()
	defer cancel()

	// Get partner first
	partnerID, err := m.GetActiveChat2(userID)
	if err != nil || partnerID == 0 {
		// Just delete self if no partner found
		return m.rdb.Del(ctx, activeChatKey(userID)).Err()
	}

	// Delete for both users
	pipe := m.rdb.TxPipeline()
	pipe.Del(ctx, activeChatKey(userID))
	pipe.Del(ctx, activeChatKey(partnerID))

	_, err = pipe.Exec(ctx)
	return err
}

// ─── Notify Online ────────────────────────────────────────────

func (m *Manager) AddOnlineNotify(ctx context.Context, targetShortID, notifyTelegramID int64) error {
	key := notifyOnlineKey(targetShortID)
	if err := m.rdb.SAdd(ctx, key, notifyTelegramID).Err(); err != nil {
		return err
	}
	return m.rdb.Expire(ctx, key, 24*time.Hour).Err()
}

func (m *Manager) PopOnlineNotifyList(ctx context.Context, targetShortID int64) ([]string, error) {
	key := notifyOnlineKey(targetShortID)
	members, err := m.rdb.SMembers(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if len(members) > 0 {
		m.rdb.Del(ctx, key)
	}
	return members, nil
}

// ─── Generic Helpers ──────────────────────────────────────────

func (m *Manager) Refresh(ctx context.Context, key string, ttl time.Duration) error {
	return m.rdb.Expire(ctx, key, ttl).Err()
}

// ─── Key Builders ─────────────────────────────────────────────

func sessionKey(id string) string     { return "session:" + id }
func userSessionKey(id string) string { return "session:user:" + id }
func userStateKey(id int64) string    { return fmt.Sprintf("user:state:%d", id) }
func lastSeenKey(id int64) string     { return fmt.Sprintf("last_seen:%d", id) }
func dmTargetKey(id int64) string     { return fmt.Sprintf("dm:target:%d", id) }
func searchStateKey(id int64) string  { return fmt.Sprintf("search_state:%d", id) }
func notifyOnlineKey(id int64) string { return fmt.Sprintf("notify_online:%d", id) }
func inQueueKey(id int64) string      { return fmt.Sprintf("in_queue:%d", id) }
func activeChatKey(id int64) string   { return fmt.Sprintf("active_chat:%d", id) }

func makeSessionID(a, b string) string {
	if a < b {
		return a + ":" + b
	}
	return b + ":" + a
}
