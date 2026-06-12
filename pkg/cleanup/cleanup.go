package cleanup

import (
	"GapGame/internal/game/game_manager"

	tele "gopkg.in/telebot.v3"
)

// StartCleanup historically swept an in-memory room map. RoomManager is now
// backed by Redis and every room/player key is written with a TTL
// (game_manager.roomTTL), so expired rooms are evicted automatically by Redis.
// This function is kept for backward compatibility and is a no-op.
func StartCleanup(rm *game_manager.RoomManager, b *tele.Bot) {
	// Intentionally empty: Redis TTL handles room expiration.
}
