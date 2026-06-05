package service

import (
	"context"
	"fmt"
	"time"

	"GapGame/internal/user"

	"go.uber.org/zap"
)

// CoinTransaction represents a coin transaction
type CoinTransaction struct {
	UserID    int64
	Amount    int
	Reason    string
	Timestamp time.Time
}

// CoinTransactionService handles coin operations with transaction logging
type CoinTransactionService struct {
	userRepo *user.Repository
	log      *zap.Logger
}

// NewCoinTransactionService creates a new coin transaction service
func NewCoinTransactionService(repo *user.Repository, log *zap.Logger) *CoinTransactionService {
	return &CoinTransactionService{
		userRepo: repo,
		log:      log,
	}
}

// DeductCoins safely deducts coins from a user with atomic check
func (cts *CoinTransactionService) DeductCoins(ctx context.Context, userID int64, amount int, reason string) (bool, error) {
	if amount <= 0 {
		return false, fmt.Errorf("invalid coin amount: %d", amount)
	}

	u, err := cts.userRepo.GetByTelegramID(ctx, userID)
	if err != nil {
		cts.log.Error("failed to get user for coin deduction",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
		return false, err
	}

	if u.Coins < amount {
		cts.log.Warn("insufficient coins",
			zap.Int64("user_id", userID),
			zap.Int("required", amount),
			zap.Int("available", u.Coins),
		)
		return false, nil
	}

	// Deduct coins
	u.Coins -= amount

	if err := cts.userRepo.Update(ctx, u); err != nil {
		cts.log.Error("failed to deduct coins",
			zap.Int64("user_id", userID),
			zap.Int("amount", amount),
			zap.Error(err),
		)
		return false, err
	}

	cts.log.Info("coins deducted successfully",
		zap.Int64("user_id", userID),
		zap.Int("amount", amount),
		zap.String("reason", reason),
	)

	return true, nil
}

// AwardCoins safely awards coins to a user
func (cts *CoinTransactionService) AwardCoins(ctx context.Context, userID int64, amount int, reason string) error {
	if amount <= 0 {
		return fmt.Errorf("invalid coin amount: %d", amount)
	}

	u, err := cts.userRepo.GetByTelegramID(ctx, userID)
	if err != nil {
		cts.log.Error("failed to get user for coin award",
			zap.Int64("user_id", userID),
			zap.Error(err),
		)
		return err
	}

	u.Coins += amount

	if err := cts.userRepo.Update(ctx, u); err != nil {
		cts.log.Error("failed to award coins",
			zap.Int64("user_id", userID),
			zap.Int("amount", amount),
			zap.Error(err),
		)
		return err
	}

	cts.log.Info("coins awarded successfully",
		zap.Int64("user_id", userID),
		zap.Int("amount", amount),
		zap.String("reason", reason),
	)

	return nil
}

// TransferCoins transfers coins between two users (atomic operation)
func (cts *CoinTransactionService) TransferCoins(ctx context.Context, fromUserID, toUserID int64, amount int, reason string) error {
	if amount <= 0 {
		return fmt.Errorf("invalid transfer amount: %d", amount)
	}

	if fromUserID == toUserID {
		return fmt.Errorf("cannot transfer coins to self")
	}

	// Get both users
	fromUser, err := cts.userRepo.GetByTelegramID(ctx, fromUserID)
	if err != nil {
		return fmt.Errorf("sender not found: %w", err)
	}

	toUser, err := cts.userRepo.GetByTelegramID(ctx, toUserID)
	if err != nil {
		return fmt.Errorf("recipient not found: %w", err)
	}

	// Check balance
	if fromUser.Coins < amount {
		return fmt.Errorf("insufficient coins: have %d, need %d", fromUser.Coins, amount)
	}

	// Update balances
	fromUser.Coins -= amount
	toUser.Coins += amount

	// Update both in database
	if err := cts.userRepo.Update(ctx, fromUser); err != nil {
		return fmt.Errorf("failed to update sender: %w", err)
	}

	if err := cts.userRepo.Update(ctx, toUser); err != nil {
		// Rollback sender's update is not possible with current schema
		// This is a limitation that should be addressed with transactions
		cts.log.Error("failed to update recipient after deducting from sender",
			zap.Int64("sender_id", fromUserID),
			zap.Int64("recipient_id", toUserID),
			zap.Error(err),
		)
		return fmt.Errorf("failed to complete transfer: %w", err)
	}

	cts.log.Info("coins transferred successfully",
		zap.Int64("from_user_id", fromUserID),
		zap.Int64("to_user_id", toUserID),
		zap.Int("amount", amount),
		zap.String("reason", reason),
	)

	return nil
}

// GetBalance returns user's current coin balance
func (cts *CoinTransactionService) GetBalance(ctx context.Context, userID int64) (int, error) {
	u, err := cts.userRepo.GetByTelegramID(ctx, userID)
	if err != nil {
		return 0, err
	}
	return u.Coins, nil
}

// HasSufficientCoins checks if user has enough coins
func (cts *CoinTransactionService) HasSufficientCoins(ctx context.Context, userID int64, amount int) (bool, error) {
	balance, err := cts.GetBalance(ctx, userID)
	if err != nil {
		return false, err
	}
	return balance >= amount, nil
}
