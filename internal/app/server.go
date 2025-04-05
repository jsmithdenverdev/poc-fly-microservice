package app

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"text/template"
	"time"
)

// NewServer creates a new HTTP server
func NewServer(ctx context.Context, stop context.CancelFunc, cfg Config, logger *slog.Logger) http.Handler {
	t := template.Must(template.ParseFS(resources, "templates/*"))
	mux := http.NewServeMux()

	addRoutes(mux, cfg, logger, t)

	var handler http.Handler = mux

	// Add inactivity timeout middleware
	if cfg.EnableInactivityTimeout {
		handler = configureInactivityTimeout(ctx, stop, cfg, logger, handler)
	}

	return handler
}

// configureInactivityTimeout configures the inactivity timeout middleware
// on the handler
func configureInactivityTimeout(
	ctx context.Context,
	stop context.CancelFunc,
	cfg Config,
	logger *slog.Logger,
	handler http.Handler) http.Handler {
	var (
		mu         sync.Mutex
		activeReqs sync.WaitGroup
	)

	duration := time.Duration(cfg.InactivityTimeout) * time.Second

	// Create a timer that will trigger after the timeout period
	// This period can be reset (see middleware)
	// If a request is actively processing this will also wait for the
	// request to complete
	timer := time.AfterFunc(duration, func() {
		activeReqs.Wait()
		logger.InfoContext(
			ctx,
			"no activity for timeout period — shutting down",
			slog.Int("timeout_period", cfg.InactivityTimeout))
		stop()
	})

	resetTimerMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			activeReqs.Add(1)
			defer activeReqs.Done()

			mu.Lock()
			timer.Reset(duration)
			mu.Unlock()

			next.ServeHTTP(w, r)
		})

	}

	return resetTimerMiddleware(handler)
}

// addRoutes adds the application routes to the mux
func addRoutes(mux *http.ServeMux, cfg Config, logger *slog.Logger, t *template.Template) {
	mux.Handle("GET /health", healthHandler(cfg, logger, t))
}
