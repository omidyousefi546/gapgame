package service

import (
	"context"
	"time"

	"go.uber.org/zap"
)

func (s *UserService) StartSyncWorker(ctx context.Context, logger *zap.Logger) {

	logger.Info("Sync worker started")

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {

		case <-ticker.C:

			err := s.session.SyncLastSeenStream(ctx, s.repoOpt, logger)
			if err != nil {
				logger.Error("sync failed", zap.Error(err))
			}

		case <-ctx.Done():
			logger.Info("Sync worker stopped")
			return
		}
	}
}
