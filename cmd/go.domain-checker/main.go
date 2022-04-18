package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"gitlab.lucky-team.pro/luckyads/go.domain-checker/internal/config"
	"gitlab.lucky-team.pro/luckyads/go.domain-checker/internal/environment"
	ll "gitlab.lucky-team.pro/luckyads/go.domain-checker/internal/logger"
	"gitlab.lucky-team.pro/luckyads/go.domain-checker/internal/server/http"
	"gitlab.lucky-team.pro/luckyads/go.domain-checker/internal/service/domain"
	"gitlab.lucky-team.pro/luckyads/go.domain-checker/internal/storage/postgres"
)

//nolint:gochecknoglobals
var (
	version   = "unknown"
	buildTime = "unknown"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	appConfig, err := config.New()
	if err != nil {
		if errors.Is(err, config.ErrHelp) {
			os.Exit(0)
		}
		log.Fatalf("failed to read app config: %v", err)
	}

	logger, err := ll.New(version, appConfig.Env, appConfig.Logger.Level)
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	defer logger.Sync() //nolint:errcheck

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	ctx = environment.CtxWithEnv(ctx, appConfig.Env)
	ctx = environment.CtxWithVersion(ctx, version)
	ctx = environment.CtxWithBuildTime(ctx, buildTime)

	pgStorage, err := postgres.New(ctx, logger, &appConfig.Postgres)
	if err != nil {
		logger.Error("failed to connect to postgres", zap.Error(err))
		return
	}
	defer pgStorage.Close() //nolint:errcheck

	httpServer, err := http.NewServer(logger, appConfig)
	if err != nil {
		logger.Error("failed to create http server", zap.Error(err))
		return
	}

	gr, appctx := errgroup.WithContext(ctx)
	gr.Go(func() error {
		return httpServer.Serve(appctx)
	})

	domainCheckerService := domain.New(&pgStorage, logger, appConfig.Region)

	gr.Go(func() error {
		return updateDomainsInBackground(appctx, logger, domainCheckerService, appConfig.Tickers.SSLChecker)
	})

	if appConfig.EnableEasylist {
		gr.Go(func() error {
			return banDomainsInBackground(appctx, logger, domainCheckerService, appConfig.Tickers.EasyListChecker)
		})
	}

	if err := gr.Wait(); err != nil {
		logger.Error("application exited with error", zap.Error(err))
	}
}

func updateDomainsInBackground(
	ctx context.Context,
	logger *zap.Logger,
	domainCheckerService domain.Service,
	duration time.Duration,
) error {
	if err := updateDomains(ctx, logger, domainCheckerService); err != nil {
		return err
	}

	tch := time.NewTicker(duration)
	defer tch.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tch.C:
			if err := updateDomains(ctx, logger, domainCheckerService); err != nil {
				return err
			}
		}
	}
}

func updateDomains(ctx context.Context, logger *zap.Logger, domainCheckerService domain.Service) error {
	start := time.Now()

	if err := domainCheckerService.UpdateDomains(ctx); err != nil {
		return fmt.Errorf("failed to update domains ssl %w", err)
	}
	logger.Info("update ssl", zap.Duration("duration", time.Since(start)))

	return nil
}

func banDomainsInBackground(
	ctx context.Context,
	logger *zap.Logger,
	domainCheckerService domain.Service,
	duration time.Duration,
) error {
	if err := banDomains(ctx, logger, domainCheckerService); err != nil {
		return err
	}

	tch := time.NewTicker(duration)
	defer tch.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tch.C:
			if err := banDomains(ctx, logger, domainCheckerService); err != nil {
				return err
			}
		}
	}
}

func banDomains(ctx context.Context, logger *zap.Logger, domainCheckerService domain.Service) error {
	start := time.Now()

	count, err := domainCheckerService.BanDomains(ctx)
	if err != nil {
		return fmt.Errorf("failed to ban domains %w", err)
	}
	logger.Info("baned domains",
		zap.Duration("duration", time.Since(start)),
		zap.Int("count", count),
	)

	return nil
}
