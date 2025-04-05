package app

import (
	"context"
	"log/slog"
	"net/http"
	"text/template"
	"time"

	"github.com/jake/poc-fly-microservice/pkg/inactivity"
)

// NewServer creates a new HTTP server
func NewServer(ctx context.Context, stop context.CancelFunc, cfg Config, h slog.Handler) http.Handler {
	t := template.Must(template.ParseFS(resources, "templates/*"))
	mux := http.NewServeMux()

	addRoutes(mux, cfg, h, t)

	var handler http.Handler = mux

	// Add inactivity timeout middleware
	if cfg.EnableInactivityTimeout {
		w := inactivity.NewWatchdog(time.Duration(cfg.InactivityTimeout)*time.Second, h, func() {
			stop()
		})
		handler = w.Middleware(handler)
	}

	return handler
}

// addRoutes adds the application routes to the mux
func addRoutes(mux *http.ServeMux, cfg Config, h slog.Handler, t *template.Template) {
	mux.Handle("GET /health", healthHandler(cfg, h, t))
}
