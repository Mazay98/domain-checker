package http

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"go.uber.org/zap"

	"gitlab.lucky-team.pro/luckyads/go.domain-checker/internal/config"
)

// Server knows how to serve http requests.
type Server struct {
	logger *zap.Logger
	config *config.Server
}

// NewServer returns new Server that will use passed config and
// consulClient during setup. To start serving requests call Server.Serve.
func NewServer(logger *zap.Logger, config *config.AppConfig) (*Server, error) {
	return &Server{
		logger: logger,
		config: &config.HTTP,
	}, nil
}

// Serve starts HTTP server. This is a blocking call.
// To stop serving, cancel the passed context.
func (s *Server) Serve(ctx context.Context) error {
	srv := &http.Server{
		Addr:    ":" + strconv.Itoa(s.config.Port),
		Handler: s.router(ctx),
	}

	e := make(chan error, 1)
	go func() {
		e <- srv.ListenAndServe()
	}()

	s.logger.Info(
		"HTTP server is running",
		zap.String("host", s.config.Host),
		zap.Int("port", s.config.Port),
	)

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Millisecond*200)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); !errors.Is(err, context.Canceled) {
			return err
		}
		return nil
	case err := <-e:
		return err
	}
}
