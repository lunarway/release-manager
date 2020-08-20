package amqpextra

import (
	"strings"

	"github.com/lunarway/release-manager/internal/log"
	"github.com/makasim/amqpextra"
)

// newLogger returns an amqpextra.LoggerFunc that logs with levels to l based on
// the format prefix of the logged lines.
func newLogger(l *log.Logger) amqpextra.LoggerFunc {
	// prefixes of log lines from github.com/makasim/amqpextra
	var (
		debugPrefix = "[DEBUG]"
		warnPrefix  = "[WARN]"
		errorPrefix = "[ERROR]"
	)
	return func(format string, args ...interface{}) {
		if strings.HasPrefix(format, debugPrefix) {
			l.Debugf(strings.TrimPrefix(format, debugPrefix), args...)
			return
		}
		if strings.HasPrefix(format, warnPrefix) {
			l.Infof(strings.TrimPrefix(format, warnPrefix), args...)
			return
		}
		if strings.HasPrefix(format, errorPrefix) {
			l.Errorf(strings.TrimPrefix(format, errorPrefix), args...)
			return
		}
	}
}
