package storage

import (
	"GapGame/internal/user"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

var pgLog *zap.Logger

func init() {
	var err error
	pgLog, err = zap.NewProduction()
	if err != nil {
		panic(err)
	}
}

// NewPostgres creates and initializes a PostgreSQL database connection with connection pooling
func NewPostgres(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger.Default.LogMode(gormLogger.Info),
	})
	if err != nil {
		pgLog.Error("Failed to connect to PostgreSQL", zap.Error(err))
		return nil, fmt.Errorf("postgres connection failed: %w", err)
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		pgLog.Error("Failed to get SQL DB instance", zap.Error(err))
		return nil, fmt.Errorf("get sql db failed: %w", err)
	}

	// Connection pooling configuration
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(2 * time.Minute)

	// Run migrations
	err = db.AutoMigrate(
		&user.User{},
		&user.Block{},
		&user.Like{},
		&user.Contact{},
		&user.Report{},
	)
	if err != nil {
		pgLog.Error("Migration failed", zap.Error(err))
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		pgLog.Error("Failed to ping PostgreSQL", zap.Error(err))
		return nil, fmt.Errorf("ping failed: %w", err)
	}

	pgLog.Info("PostgreSQL connected successfully with pooling configured")
	return db, nil
}
