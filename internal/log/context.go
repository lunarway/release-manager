package log

import (
	"context"
)

type contextKey struct{}

// WithContext returns a Logger instance. Every logged line with the returned
// logger will contain the extracted fields (if any) from the context.
func WithContext(ctx context.Context) *Logger {
	return logger.WithContext(ctx)
}

// AddContext adds fields to the context that can be used in WithContext.
func AddContext(ctx context.Context, fields ...interface{}) context.Context {
	return context.WithValue(ctx, contextKey{}, fields)
}
