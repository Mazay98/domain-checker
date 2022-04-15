package logger

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"gitlab.lucky-team.pro/luckyads/go.domain-checker/internal/environment"
)

// New returns new zap logger for the given service that is launched
// in the given environment. Optionally, non-empty log level will be set.
func New(version string, env environment.Env, level string) (*zap.Logger, error) {
	config := zap.NewDevelopmentConfig()
	if env.IsProduction() {
		config = zap.NewProductionConfig()
	}

	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	if err := config.Level.UnmarshalText([]byte(level)); err != nil && level != "" {
		log.Printf("failed to set log level %q: %v", level, err)
	}

	logger, err := config.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	logger = logger.With(
		zap.String("version", version),
		zap.String("environment", env.String()),
	)

	go func() {
		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
		defer cancel()

		usr1 := make(chan os.Signal, 1)
		signal.Notify(usr1, syscall.SIGUSR1)
		initialLevel := config.Level.Level()

		for {
			select {
			case <-ctx.Done():
				return
			case <-usr1:
				logger.Info("caught SIGUSR1 signal, toggling log level")
				var nextLevel zapcore.Level
				if config.Level.Level() == zap.DebugLevel {
					nextLevel = initialLevel
				} else {
					nextLevel = zap.DebugLevel
				}
				config.Level.SetLevel(nextLevel)
				logger.Info("log level changed", zap.String("level", nextLevel.String()))
			}
		}
	}()

	return logger, nil
}
