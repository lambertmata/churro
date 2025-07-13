package churro

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"
)

// Request ID context key
const requestIDKey contextKey = "churro:request_id"

// generateRequestID creates a random request ID
func generateRequestID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if random fails
		return hex.EncodeToString([]byte(time.Now().Format("20060102150405.000")))
	}
	return hex.EncodeToString(bytes)
}

// WithRequestID adds a request ID to the context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// GetRequestID retrieves the request ID from the context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// timeoutDuration converts seconds to time.Duration
func timeoutDuration(seconds int) time.Duration {
	return time.Duration(seconds) * time.Second
}
