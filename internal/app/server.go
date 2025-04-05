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
	mux := http.NewServeMux()

	t := template.Must(template.ParseFS(resources, "templates/*"))

	addRoutes(mux, cfg, logger, t)

	var handler http.Handler = mux

	// Add inactivity timeout middleware
	if cfg.EnableInactivityTimeout {
		var mu sync.Mutex
		duration := time.Duration(cfg.InactivityTimeout) * time.Second

		// Create a timer that will trigger after the timeout period
		// This period can be reset (see middleware)
		timer := time.AfterFunc(duration, func() {
			logger.InfoContext(
				ctx,
				"no activity for timeout period — shutting down",
				slog.Int("timeout_period", cfg.InactivityTimeout))
			stop()
		})

		handler = shutdownTimerResetMiddleware(&mu, timer, duration, mux)
	}

	return handler
}

// addRoutes adds the application routes to the mux
func addRoutes(mux *http.ServeMux, cfg Config, logger *slog.Logger, t *template.Template) {
	mux.Handle("GET /health", healthHandler(cfg, logger, t))
}
