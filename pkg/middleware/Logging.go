package middleware

import (
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

const handlerKey = "handler_name"

func WithHandlerName(name string) tele.MiddlewareFunc {
	return func(next tele.HandlerFunc) tele.HandlerFunc {
		return func(c tele.Context) error {

			c.Set(handlerKey, name)

			return next(c)
		}
	}
}

func Logging(log *zap.Logger) tele.MiddlewareFunc {

	return func(next tele.HandlerFunc) tele.HandlerFunc {

		return func(c tele.Context) error {

			start := time.Now()

			err := next(c)

			handler_name, _ := c.Get("handler_name").(string)

			user := c.Sender()

			if user != nil {
				log.Info(
					"update handled",
					zap.String("handler", handler_name),
					zap.Int64("user_id", user.ID),
					zap.String("first_name", user.FirstName),
					zap.Duration("latency", time.Since(start)),
				)
			}

			return err
		}
	}
}
