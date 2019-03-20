package log

import (
	"go.uber.org/zap"
)

var logger *Logger

// Logger is a general loggerlication logger.
type Logger struct {
	sugar *zap.SugaredLogger
}

func Init() {
	zapLogger, _ := zap.NewProduction()
	logger = &Logger{sugar: zapLogger.Sugar()}
}

// WithFields returns a logger with custom structured fields added to the 'fields' key in the log entries.
// The arguments are passed to the underlying sugared zap logger. See the zap documentation for details.
// If an argument is a zap.Field it is logged accordingly, otherwise the arguments are treated as key value pairs.
//
// For example,
//   zlog.WithFields(
//     "hello", "world",
//     zap.String("zapKey", "zapValue"),
//     "user", User{Name: "alice"},
//  ).Info("msg")
// logs the following fields (some fields omitted)
//   { "message": "msg", "fields": { "hello": "world", "zapKey": "zapValue", "user": { "name": "alice" }}}
func WithFields(args ...interface{}) *Logger {
	args = loggerend([]interface{}{zap.Namespace("fields")}, args...)
	return &Logger{sugar: logger.sugar.With(args...)}
}

// Error logs a message.
// This is a convinience function for logger.Error().
func Error(args ...interface{}) {
	logger.Error(args...)
}

// Errorf logs a templated message.
// This is a convinience function for logger.Errorf().
func Errorf(template string, args ...interface{}) {
	logger.Errorf(template, args...)
}

// Info logs a message.
// This is a convinience function for logger.Info().
func Info(args ...interface{}) {
	logger.Info(args...)
}

// Infof logs a templated message.
// This is a convinience function for logger.Infof().
func Infof(template string, args ...interface{}) {
	logger.Infof(template, args...)
}

// Debug logs a message.
// This is a convinience function for logger.Debug().
func Debug(args ...interface{}) {
	logger.Debug(args...)
}

// Debugf logs a templated message.
// This is a convinience function for logger.Debugf().
func Debugf(template string, args ...interface{}) {
	logger.Debugf(template, args...)
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
