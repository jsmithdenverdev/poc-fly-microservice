package app

import (
	"net/http"
	"sync"
	"time"
)

// Middleware to reset the inactivity timeout timer on every request
func shutdownTimerResetMiddleware(mu *sync.Mutex, timer *time.Timer, timeout time.Duration, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		timer.Reset(timeout)
		mu.Unlock()
		next.ServeHTTP(w, r)
	})
}
