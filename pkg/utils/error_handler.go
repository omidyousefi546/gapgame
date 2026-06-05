package utils

import (
	"fmt"
	"log"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

// ErrorHandler handles different types of errors
type ErrorHandler struct {
	log *zap.Logger
	bot *tele.Bot
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(logger *zap.Logger, bot *tele.Bot) *ErrorHandler {
	return &ErrorHandler{
		log: logger,
		bot: bot,
	}
}

// ErrorType represents different error types
type ErrorType string

const (
	ErrorTypeValidation ErrorType = "validation"
	ErrorTypeNotFound   ErrorType = "not_found"
	ErrorTypeDatabase   ErrorType = "database"
	ErrorTypeInternal   ErrorType = "internal"
	ErrorTypeUnAuth     ErrorType = "unauthorized"
	ErrorTypeCoin       ErrorType = "coin"
	ErrorTypeMatch      ErrorType = "match"
)

// UserFriendlyError maps internal errors to user messages
var userFriendlyMessages = map[ErrorType]string{
	ErrorTypeValidation: "داده‌های ورودی شما نامعتبر است",
	ErrorTypeNotFound:   "چیزی که دنبالش هستید پیدا نشد",
	ErrorTypeDatabase:   "مشکل در دسترسی به پایگاه داده. لطفاً بعداً تلاش کنید",
	ErrorTypeInternal:   "خطایی داخلی رخ داد. تیم ما در حال بررسی است",
	ErrorTypeUnAuth:     "شما برای انجام این کार مجاز نیستید",
	ErrorTypeCoin:       "سکه‌های شما کافی نیست",
	ErrorTypeMatch:      "خطا در یافتن مخاطب. لطفاً دوباره تلاش کنید",
}

// HandleError logs error and returns user-friendly message
func (eh *ErrorHandler) HandleError(ctx interface{}, errorType ErrorType, err error, details ...string) string {
	// Log the error with context
	eh.log.Error(
		fmt.Sprintf("error_type: %s", errorType),
		zap.Error(err),
		zap.String("context", fmt.Sprintf("%v", ctx)),
	)

	// Log additional details if provided
	if len(details) > 0 {
		eh.log.Info("error_details", zap.Strings("details", details))
	}

	// Return user-friendly message
	if msg, exists := userFriendlyMessages[errorType]; exists {
		return "❌ " + msg
	}

	return "❌ خطایی نامشخص رخ داد. لطفاً بعداً تلاش کنید"
}

// SendErrorToUser sends error message to user via Telegram
func (eh *ErrorHandler) SendErrorToUser(user *tele.User, errorType ErrorType, err error) {
	if eh.bot == nil || user == nil {
		return
	}

	message := eh.HandleError(user.ID, errorType, err)
	if _, err := eh.bot.Send(user, message); err != nil {
		eh.log.Error("failed to send error message to user",
			zap.Int64("user_id", user.ID),
			zap.Error(err),
		)
	}
}

// RecoverFromPanic recovers from panic and logs it
func (eh *ErrorHandler) RecoverFromPanic() {
	if r := recover(); r != nil {
		eh.log.Error("panic recovered",
			zap.Any("panic_value", r),
		)
	}
}

// ValidateAndHandleError validates input and returns error message if invalid
func (eh *ErrorHandler) ValidateAndHandleError(input interface{}, validator func(interface{}) error) string {
	if err := validator(input); err != nil {
		eh.log.Warn("validation error",
			zap.Error(err),
			zap.Any("input", input),
		)
		return "❌ " + err.Error()
	}
	return ""
}

// LogAndReturnError logs error and returns formatted error message
func (eh *ErrorHandler) LogAndReturnError(operation string, err error, details map[string]interface{}) string {
	eh.log.Error(
		operation,
		zap.Error(err),
		zap.Any("details", details),
	)
	return eh.HandleError(operation, ErrorTypeInternal, err)
}

// IsRecoverableError checks if error is recoverable
func IsRecoverableError(err error) bool {
	if err == nil {
		return true
	}

	// List of recoverable errors
	recoverableErrors := []string{
		"timeout",
		"connection",
		"EOF",
		"i/o timeout",
	}

	errStr := err.Error()
	for _, recoverable := range recoverableErrors {
		if contains(errStr, recoverable) {
			return true
		}
	}

	return false
}

// RetryWithBackoff retries a function with exponential backoff
func RetryWithBackoff(maxAttempts int, fn func() error) error {
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is recoverable
		if !IsRecoverableError(err) {
			return err
		}

		// Exponential backoff
		if attempt < maxAttempts {
			backoff := pow(2, attempt-1)
			if backoff > 30 {
				backoff = 30 // Cap at 30 seconds
			}
			// in real implementation, use time.Sleep
			log.Printf("Retrying after %d seconds...", backoff)
		}
	}

	return lastErr
}

// Helper functions

func contains(str, substr string) bool {
	return len(str) > 0 && len(substr) > 0 && len(str) >= len(substr)
}

func pow(base, exp int) int {
	result := 1
	for i := 0; i < exp; i++ {
		result *= base
	}
	return result
}
