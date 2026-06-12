package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken    string
	DatabaseURL string
	RedisURL    string
	BotUsername string
	// AdminIDs are the Telegram user IDs allowed to run admin commands.
	// Configured via ADMIN_IDS as a comma-separated list, e.g. "123,456".
	AdminIDs map[int64]bool
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, reading from environment")
	}

	return &Config{
		BotToken:    getEnv("BOT_TOKEN", ""),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		RedisURL:    getEnv("REDIS_URL", "localhost:6379"),
		BotUsername: getEnv("BOT_USERNAME", ""),
		AdminIDs:    parseAdminIDs(getEnv("ADMIN_IDS", "")),
	}
}

func parseAdminIDs(raw string) map[int64]bool {
	admins := make(map[int64]bool)
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			log.Printf("config: invalid admin id %q ignored", part)
			continue
		}
		admins[id] = true
	}
	return admins
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
