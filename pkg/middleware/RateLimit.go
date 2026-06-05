package middleware

import (
	"sync"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

var (
	lastRequestMu sync.RWMutex
	lastRequest   = make(map[int64]time.Time)
)

func RateLimit(log *zap.Logger, limit *time.Duration) tele.MiddlewareFunc {

	return func(next tele.HandlerFunc) tele.HandlerFunc {

		return func(c tele.Context) error {

			user := c.Sender()
			if user == nil {
				return next(c)
			}

			lastRequestMu.RLock()
			last, ok := lastRequest[user.ID]
			lastRequestMu.RUnlock()

			if ok && time.Since(last) < *limit {
				log.Warn("rate limit",
					zap.Int64("user_id", user.ID),
				)
				return nil
			}

			lastRequestMu.Lock()
			lastRequest[user.ID] = time.Now()
			lastRequestMu.Unlock()

			return next(c)
		}
	}
}
