package main

import (
	"GapGame/config"
	"GapGame/internal/bot"
	"GapGame/internal/game/game_manager"
	"GapGame/internal/service"
	"GapGame/internal/session"
	"GapGame/internal/storage"
	"GapGame/internal/user"
	"GapGame/pkg/logger"
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func main() {
	appLog := logger.New("app")
	match_worker := logger.New("match_worker")
	sync_worker := logger.New("last_seen_worker")

	cfg := config.Load()

	// Initialize PostgreSQL database
	db, err := storage.NewPostgres(cfg.DatabaseURL)
	if err != nil {
		appLog.Fatal("Failed to connect to database", zap.Error(err))
	}

	// Initialize Redis
	rdb, err := storage.NewRedis(cfg.RedisURL)
	if err != nil {
		appLog.Fatal("Failed to connect to Redis", zap.Error(err))
	}

	// Initialize Telegram bot
	b, err := tele.NewBot(tele.Settings{
		Token:  cfg.BotToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		appLog.Fatal("Failed to create bot", zap.Error(err))
	}

	// Initialize dependencies
	userRepo := user.NewRepository(db)
	userRepoOpt := user.NewRepositoryOptimizations(userRepo)

	sessionManager := session.NewManager(rdb)
	userService := service.NewUserService(userRepo, userRepoOpt, sessionManager)
	roomManager := game_manager.NewRoomManager()

	// Initialize handler
	h := bot.New(b, userService, sessionManager, userRepo, roomManager, appLog)
	h.RegisterHandlers()

	// Start match worker
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go h.StartMatchWorker(match_worker, ctx)

	go userService.StartSyncWorker(ctx, sync_worker)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go h.Start()

	<-quit
	cancel()
	h.Stop()
}
