package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"pikabot/internal/config"
	"pikabot/internal/logx"
	"pikabot/internal/matrix"
)

func main() {
	logger := logx.New(parseLogLevel(os.Getenv("LOG_LEVEL")))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("invalid configuration", "err", err.Error())
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	bot, err := matrix.New(ctx, cfg, logger)
	if err != nil {
		logger.Error("failed creating bot", "err", err.Error())
		os.Exit(1)
	}
	defer func() {
		if closeErr := bot.Close(); closeErr != nil {
			logger.Warn("failed closing docker client", "err", closeErr.Error())
		}
	}()

	if err := bot.Run(ctx); err != nil && !errors.Is(ctx.Err(), context.Canceled) {
		logger.Error("bot stopped with error", "err", err.Error())
		os.Exit(1)
	}

	logger.Info("bot shutdown complete")
}

func parseLogLevel(raw string) logx.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return logx.Debug
	case "warn", "warning":
		return logx.Warn
	case "error":
		return logx.Error
	default:
		return logx.Info
	}
}
