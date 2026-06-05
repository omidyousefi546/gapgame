package utils

import (
	"context"
	"time"
)

func NewRequestContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}
