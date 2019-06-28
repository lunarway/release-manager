package log

import (
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Init initializes the global logger with proided level.
//
// Default level is zapcore.InfoLevel and non-development.
func Init(c *Configuration) {
	if c == nil {
		c = &Configuration{}
	}
	fmt.Printf("Config: %+v\n", *c)
	encoder := zapcore.EncoderConfig{
		TimeKey:        "@timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	config := zap.Config{
		Level:            zap.NewAtomicLevelAt(c.Level.Level),
		Development:      false,
		Encoding:         "json",
		EncoderConfig:    encoder,
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}
	if c.Development {
		config.Development = true
		config.Encoding = "console"
		config.EncoderConfig.TimeKey = ""
	}
	zapLogger, _ := config.Build(zap.AddCallerSkip(2))
	logger = &Logger{sugar: zapLogger.Sugar()}
}

// Configuration represents a log configuration.
type Configuration struct {
	Level       Level
	Development bool
}

// ParseFromEnvironmnet parses configuration from the environment and writes
// found values to configuration struct c.
func (c *Configuration) ParseFromEnvironmnet() {
	if c == nil {
		c = &Configuration{}
	}
	parseEnvLevel(c)
	parseEnvDevelopment(c)
}

func parseEnvLevel(c *Configuration) {
	l, ok := os.LookupEnv("LOG_LEVEL")
	if !ok {
		return
	}
	var level zapcore.Level
	err := level.Set(l)
	if err != nil {
		fmt.Printf("internal/log: failed to parse LOG_LEVEL: %v\n", err)
		return
	}
	c.Level = Level{
		Level: level,
	}
}

func parseEnvDevelopment(c *Configuration) {
	d, ok := os.LookupEnv("LOG_DEVELOPMENT")
	if !ok {
		return
	}
	development, err := strconv.ParseBool(d)
	if err != nil {
		fmt.Printf("internal/log: failed to parse LOG_DEVELOPMENT '%s' as bool\n", d)
		return
	}
	c.Development = development
}

// RegisterFlags registers logging configuration flags on command cmd and
// returns pointers to the values.
func RegisterFlags(cmd *cobra.Command) *Configuration {
	var c Configuration
	cmd.PersistentFlags().Var(&c.Level, "log.level", "configure log level. Available values are \"debug\", \"info\", \"error\" (fallback to LOG_LEVEL)")
	cmd.PersistentFlags().BoolVar(&c.Development, "log.development", false, "configure log for development with human readable output (fallback to LOG_DEVELOPMENT)")
	return &c
}

var _ pflag.Value = &Level{}

// Level is a wrapped zapcore.Level that implements the pflag.Value interface.
type Level struct {
	zapcore.Level
}

func (*Level) Type() string {
	return "string"
}
