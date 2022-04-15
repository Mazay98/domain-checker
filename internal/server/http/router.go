package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"gitlab.lucky-team.pro/luckyads/go.domain-checker/internal/environment"
)

func (s *Server) router(ctx context.Context) http.Handler {
	mux := chi.NewRouter()

	mux.Use(
		middleware.Recoverer,
		middleware.Heartbeat("/check"),
	)

	mux.Get("/deploy/info", deployInfoHandlerFunc(ctx))

	return mux
}

func deployInfoHandlerFunc(ctx context.Context) http.HandlerFunc {
	info := map[string]string{
		"service":     environment.ServiceName,
		"environment": environment.EnvFromCtx(ctx).String(),
		"version":     environment.VersionFromCtx(ctx),
		"build_time":  environment.BuildTimeFromCtx(ctx),
	}

	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(info) //nolint:errcheck,gosec
	}
}
