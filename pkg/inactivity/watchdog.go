// Package inactivity provides middleware and logic for gracefully shutting
// down a service after a period of inactivity, while allowing active
// requests to complete and resetting on any new incoming request.
package inactivity

import (
	"context"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Watchdog manages the inactivity shutdown behavior.
type Watchdog struct {
	timeout     time.Duration
	resetCh     chan struct{}
	shutdownFn  func()
	shutdownCtx context.Context
	cancel      context.CancelFunc
	activeReqs  sync.WaitGroup
	once        sync.Once
	logger      *slog.Logger
}

// New creates a new Watchdog. The shutdownFn will be called
// only after the timeout *and* all active requests have finished.
func NewWatchdog(timeout time.Duration, h slog.Handler, shutdownFn func()) *Watchdog {
	ctx, cancel := context.WithCancel(context.Background())
	w := &Watchdog{
		timeout:     timeout,
		resetCh:     make(chan struct{}, 1),
		shutdownFn:  shutdownFn,
		shutdownCtx: ctx,
		cancel:      cancel,
		logger:      slog.New(h),
	}
	go w.watch()
	return w
}

// Middleware returns an http.Handler that resets the timer
// and tracks in-flight requests.
func (w *Watchdog) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		// select {
		// case w.resetCh <- struct{}{}:
		// default:
		// }

		w.resetCh <- struct{}{}

		w.activeReqs.Add(1)
		defer w.activeReqs.Done()
		next.ServeHTTP(rw, r)
	})
}

// Stop cancels the background watchdog goroutine.
func (w *Watchdog) Stop() {
	w.once.Do(func() {
		w.cancel()
	})
}

// watch monitors inactivity and triggers shutdown when appropriate.
func (w *Watchdog) watch() {
	timer := time.NewTimer(w.timeout)
	defer timer.Stop()

	for {
		select {
		case <-w.shutdownCtx.Done():
			return

		case <-w.resetCh:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			w.logger.InfoContext(w.shutdownCtx, "[inactivity] resetting timeout", slog.Int("timeout", int(w.timeout.Seconds())))
			timer.Reset(w.timeout)

		case <-timer.C:
			w.logger.InfoContext(w.shutdownCtx, "[inactivity] timeout reached - waiting for active requests to finish", slog.Int("timeout", int(w.timeout.Seconds())))
			w.activeReqs.Wait()
			w.logger.InfoContext(w.shutdownCtx, "[inactivity] all requests done - shutting down")
			w.shutdownFn()
			return
		}
	}
}
