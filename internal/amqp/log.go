package amqp

import (
	"fmt"
	"strings"

	"github.com/makasim/amqpextra/logger"
)

// amqpextraLogger is a logger that inspects the prefix against the known levels
// of amqpextra and calls the corresponding leveled logger methods. If we cannot
// detect the level it is logged at level info.
type amqpextraLogger struct {
	leveledLogger Logger
	debugPrefix   string
	warnPrefix    string
	errorPrefix   string
	logPrefix     string
}

// newLogger returns an amqpextra/logger.Logger that logs with levels to l based
// on the format prefix of the logged lines.
func newLogger(l Logger, t string) logger.Logger {
	return &amqpextraLogger{
		leveledLogger: l,
		// prefixes of log lines from github.com/makasim/amqpextra
		debugPrefix: "[DEBUG]",
		warnPrefix:  "[WARN]",
		errorPrefix: "[ERROR]",
		// used to differentiate logs from the underlying amqpextra lib
		logPrefix: fmt.Sprintf("[amqp] system=amqpextra type=%s", t),
	}
}

func (l *amqpextraLogger) Printf(format string, args ...interface{}) {
	if strings.HasPrefix(format, l.debugPrefix) {
		l.leveledLogger.Debugf(l.logPrefix+strings.TrimPrefix(format, l.debugPrefix), args...)
		return
	}
	if strings.HasPrefix(format, l.warnPrefix) {
		l.leveledLogger.Infof(l.logPrefix+strings.TrimPrefix(format, l.warnPrefix), args...)
		return
	}
	if strings.HasPrefix(format, l.errorPrefix) {
		l.leveledLogger.Errorf(l.logPrefix+strings.TrimPrefix(format, l.errorPrefix), args...)
		return
	}
	l.leveledLogger.Infof(l.logPrefix+format, args...)
}
