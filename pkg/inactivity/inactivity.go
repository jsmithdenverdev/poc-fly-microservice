package inactivity

import (
	"net/http"
	"sync"
	"time"
)

type Inactivity struct {
	timeout    time.Duration
	onInactive func()

	mu          sync.Mutex
	active      int
	timer       *time.Timer
	timerActive bool
}

func New(timeout time.Duration, onInactive func()) *Inactivity {
	inactivity := &Inactivity{
		timeout:    timeout,
		onInactive: onInactive,
	}
	inactivity.timerActive = true
	inactivity.timer = time.AfterFunc(timeout, func() {
		inactivity.mu.Lock()
		defer inactivity.mu.Unlock()
		if inactivity.active == 0 {
			inactivity.onInactive()
		}
		inactivity.timerActive = false
		inactivity.timer = nil
	})
	return inactivity
}

func (i *Inactivity) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		i.increment()
		defer i.decrement()
		next.ServeHTTP(w, r)
	})
}

func (i *Inactivity) Shutdown() {
	i.mu.Lock()
	defer i.mu.Unlock()
	if i.timer != nil {
		i.timer.Stop()
	}
}

func (i *Inactivity) increment() {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.active++
	if i.timer != nil {
		i.timer.Stop()
		i.timer = nil
		i.timerActive = false
	}
}

func (i *Inactivity) decrement() {
	i.mu.Lock()
	defer i.mu.Unlock()

	i.active--
	if i.active == 0 && !i.timerActive {
		i.timerActive = true
		i.timer = time.AfterFunc(i.timeout, func() {
			i.mu.Lock()
			defer i.mu.Unlock()
			if i.active == 0 {
				i.onInactive()
			}
			i.timerActive = false
			i.timer = nil
		})
	}
}
