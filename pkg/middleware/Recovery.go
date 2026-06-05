package middleware

import (
	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func Recovery(log *zap.Logger) tele.MiddlewareFunc {

	return func(next tele.HandlerFunc) tele.HandlerFunc {

		return func(c tele.Context) error {

			defer func() {
				if r := recover(); r != nil {

					log.Error("panic recovered",
						zap.Any("error", r),
					)
				}
			}()

			return next(c)
		}
	}
}
