package log

import (
	"log/slog"
	"os"

	gethlog "github.com/ethereum/go-ethereum/log"
)

type LogConfig struct {
	Level string `mapstructure:"level"`
}

func InitLogger(cfg *LogConfig) {
	var slogLevel slog.Level

	switch cfg.Level {
	case "debug":
		slogLevel = slog.LevelDebug
	case "info":
		slogLevel = slog.LevelInfo
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	gethlog.SetDefault(gethlog.NewLogger(gethlog.NewTerminalHandlerWithLevel(os.Stdout, slogLevel, true)))

	Info("logger initialized", "level", cfg.Level)
}

func With(ctx ...interface{}) gethlog.Logger {
	return gethlog.Root().With(ctx...)
}

func Debug(format string, v ...interface{}) {
	gethlog.Debug(format, v...)
}

func Info(format string, v ...interface{}) {
	gethlog.Info(format, v...)
}

func Warn(format string, v ...interface{}) {
	gethlog.Warn(format, v...)
}

func Error(format string, v ...interface{}) {
	gethlog.Error(format, v...)
}
