package middleware

import (
	tele "gopkg.in/telebot.v3"
)

// RequireAdmin only lets configured admin IDs through. Maps are reference
// types in Go, so passing the map itself (not a pointer) is sufficient.
func RequireAdmin(admins *map[int64]bool) tele.MiddlewareFunc {

	return func(next tele.HandlerFunc) tele.HandlerFunc {

		return func(c tele.Context) error {

			user := c.Sender()

			if user == nil || admins == nil || !(*admins)[user.ID] {
				// Silently ignore: don't advertise admin commands to users.
				return nil
			}

			return next(c)
		}
	}
}

// func RequireChannel(channel *string) tele.MiddlewareFunc {

// 	return func(next tele.HandlerFunc) tele.HandlerFunc {

// 		return func(c tele.Context) error {

// 			user := c.Sender()

// 			member, err := c.Bot().ChatMember(*channel, user)

// 			if err != nil || member.Role == tele.Left {
// 				return c.Send("📢 ابتدا عضو کانال شوید")
// 			}

// 			return next(c)
// 		}
// 	}
// }
