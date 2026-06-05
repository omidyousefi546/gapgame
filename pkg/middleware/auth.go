package middleware

import (
	tele "gopkg.in/telebot.v3"
)

func RequireAdmin(admins *map[int64]bool) tele.MiddlewareFunc {

	return func(next tele.HandlerFunc) tele.HandlerFunc {

		return func(c tele.Context) error {

			user := c.Sender()

			if user == nil || !(*admins)[user.ID] {
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
