package log

import (
	"context"

	"go.uber.org/zap"
)

// Logger is a general structured logger.
type Logger struct {
	sugar *zap.SugaredLogger
}

// Error logs a message.
func (l *Logger) Error(args ...interface{}) {
	l.sugar.Error(args...)
}

// Errorf logs a templated message.
func (l *Logger) Errorf(template string, args ...interface{}) {
	l.sugar.Errorf(template, args...)
}

// Info logs a message.
func (l *Logger) Info(args ...interface{}) {
	l.sugar.Info(args...)
}

// Infof logs a templated message.
func (l *Logger) Infof(template string, args ...interface{}) {
	l.sugar.Infof(template, args...)
}

// Debug logs a message.
func (l *Logger) Debug(args ...interface{}) {
	l.sugar.Debug(args...)
}

// Debugf logs a templated message.
func (l *Logger) Debugf(template string, args ...interface{}) {
	l.sugar.Debugf(template, args...)
}

// WithFields returns a logger with custom structured fields added to the 'fields' key in the log entries.
// The arguments are passed to the underlying sugared zap logger. See the zap documentation for details.
// If an argument is a zap.Field it is logged accordingly, otherwise the arguments are treated as key value pairs.
//
// For example,
//
//	log.WithFields("hello", "world").WithFields(zap.String("zapKey", "zapValue")).Info("msg")
//
// logs the following fields (some fields omitted)
//
//	{ "message": "msg", "fields": { "hello": "world", "zapKey": "zapValue" }}
func (l *Logger) WithFields(args ...interface{}) *Logger {
	args = append([]interface{}{zap.Namespace("fields")}, args...)
	return l.With(args...)
}

// With returns a logger with custom structured fields added to the root of the log entries.
// The arguments are passed to the underlying sugared zap logger. See the zap documentation for details.
// If an argument is a zap.Field it is logged accordingly, otherwise the arguments are treated as key value pairs.
//
// For example,
//
//	log.With("hello", "world").With(zap.String("zapKey", "zapValue")).Info("msg")
//
// logs the following fields (some fields omitted)
//
//	{ "message": "msg", "hello": "world", "zapKey": "zapValue"}
func (l *Logger) With(args ...interface{}) *Logger {
	return &Logger{sugar: l.sugar.With(args...)}
}

// WithContext returns a Logger instance. Every logged line with the returned
// logger will contain the extracted fields (if any) from the context.
func (l *Logger) WithContext(ctx context.Context) *Logger {
	fields, ok := ctx.Value(contextKey{}).([]interface{})
	if !ok {
		return l
	}
	return l.With(fields...)
}
