package main

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// activityTracker tracks the last activity time of the service.
// It is used to implement auto-shutdown functionality after a period of inactivity.
type activityTracker struct {
	lastActivity time.Time     // timestamp of the last recorded activity
	mu           sync.RWMutex  // mutex to protect concurrent access to lastActivity
	logger       *slog.Logger  // structured logger for activity events
}

// newActivityTracker creates a new activity tracker with the current time
// as the initial last activity time. It requires a logger for recording events.
func newActivityTracker(logger *slog.Logger) *activityTracker {
	return &activityTracker{
		lastActivity: time.Now(),
		logger:       logger,
	}
}

// update records a new activity by updating the last activity timestamp.
// This method is thread-safe and can be called concurrently.
func (t *activityTracker) update() {
	t.mu.Lock()
	t.lastActivity = time.Now()
	t.mu.Unlock()
}

// timeSinceLastActivity returns the duration since the last recorded activity.
// This method is thread-safe and can be called concurrently.
func (t *activityTracker) timeSinceLastActivity() time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return time.Since(t.lastActivity)
}

// withActivityTracking wraps an HTTP handler with activity tracking middleware.
// It updates the activity timestamp for each request and logs request details.
// The middleware uses structured logging to record method, path, and request duration.
func withActivityTracking(ctx context.Context, logger *slog.Logger, tracker *activityTracker, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		tracker.update()
		next.ServeHTTP(w, r)
		logger.InfoContext(r.Context(), "request processed",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Duration("duration", time.Since(start)),
		)
	})
}

// monitorActivity continuously monitors the activity tracker and initiates
// shutdown if no activity is detected for longer than the shutdown timeout.
// It respects context cancellation and uses structured logging for events.
func monitorActivity(ctx context.Context, logger *slog.Logger, tracker *activityTracker, shutdown chan<- struct{}) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			inactiveDuration := tracker.timeSinceLastActivity()
			if inactiveDuration > shutdownTimeout {
				logger.InfoContext(ctx, "initiating shutdown due to inactivity",
					slog.Duration("inactive_duration", inactiveDuration),
					slog.Duration("timeout", shutdownTimeout),
				)
				close(shutdown)
				return
			}
			logger.DebugContext(ctx, "activity check",
				slog.Duration("inactive_duration", inactiveDuration),
				slog.Duration("timeout", shutdownTimeout),
			)
		}
	}
}
