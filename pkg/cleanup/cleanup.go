package cleanup

import (
	"GapGame/internal/game/game_manager"
	"time"

	tele "gopkg.in/telebot.v3"
)

func StartCleanup(rm *game_manager.RoomManager, b *tele.Bot) {

	ticker := time.NewTicker(60 * time.Minute)

	for range ticker.C {

		var expired []*game_manager.Room

		rm.Mu.Lock()

		for id, room := range rm.Rooms {

			if time.Since(room.LastMove) > 60*time.Minute {

				expired = append(expired, room)

				delete(rm.Rooms, id)
				delete(rm.PlayerRoom, room.Player1.ID)

				if room.Player2 != nil {
					delete(rm.PlayerRoom, room.Player2.ID)
				}
			}
		}

		rm.Mu.Unlock()

		for _, room := range expired {

			b.Send(room.Player1, "/start")

			if room.Player2 != nil {
				b.Send(room.Player2, "/start")
			}
		}
	}
}
