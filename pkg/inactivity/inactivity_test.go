package inactivity_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jake/poc-fly-microservice/pkg/inactivity"
)

func TestInactivityMiddleware(t *testing.T) {
	tests := []struct {
		name            string
		timeout         time.Duration
		requests        []time.Duration // times (since start) to send each request
		handlerDuration time.Duration   // how long each request takes
		wait            time.Duration   // total wait duration after requests
		wantTriggered   bool
	}{
		{
			name:            "triggers after inactivity",
			timeout:         2 * time.Second,
			requests:        []time.Duration{0},
			handlerDuration: 100 * time.Millisecond,
			wait:            3 * time.Second,
			wantTriggered:   true,
		},
		{
			name:            "does not trigger if requests keep coming",
			timeout:         2 * time.Second,
			requests:        []time.Duration{0, 1 * time.Second, 2 * time.Second},
			handlerDuration: 100 * time.Millisecond,
			wait:            1 * time.Second,
			wantTriggered:   false,
		},
		{
			name:            "resets after long request finishes",
			timeout:         2 * time.Second,
			requests:        []time.Duration{0},
			handlerDuration: 3 * time.Second,
			wait:            1 * time.Second,
			wantTriggered:   false,
		},
		{
			name:            "triggers after long request finishes and no new ones",
			timeout:         2 * time.Second,
			requests:        []time.Duration{0},
			handlerDuration: 3 * time.Second,
			wait:            6 * time.Second, // 6s > 3s handler + 2s timeout
			wantTriggered:   true,
		},
		{
			name:            "no requests",
			timeout:         2 * time.Second,
			requests:        []time.Duration{},
			handlerDuration: 100 * time.Millisecond,
			wait:            3 * time.Second,
			wantTriggered:   true,
		},
		{
			name:            "rapid successive requests",
			timeout:         2 * time.Second,
			requests:        []time.Duration{0, 100 * time.Millisecond, 200 * time.Millisecond},
			handlerDuration: 50 * time.Millisecond,
			wait:            1 * time.Second,
			wantTriggered:   false,
		},
		{
			name:            "delayed first request",
			timeout:         2 * time.Second,
			requests:        []time.Duration{3 * time.Second},
			handlerDuration: 100 * time.Millisecond,
			wait:            3 * time.Second,
			wantTriggered:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			done := make(chan struct{})

			mw := inactivity.New(tt.timeout, func() {
				close(done)
			})

			handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(tt.handlerDuration)
			}))

			server := httptest.NewServer(handler)
			defer server.Close()

			start := time.Now()
			for _, offset := range tt.requests {
				sleep := time.Until(start.Add(offset))
				if sleep > 0 {
					time.Sleep(sleep)
				}
				go func() {
					resp, err := http.Get(server.URL)
					if err != nil {
						t.Error(err)
						return
					}
					resp.Body.Close()
				}()
			}

			select {
			case <-done:
				if !tt.wantTriggered {
					t.Errorf("unexpected trigger")
				}
			case <-time.After(tt.wait):
				if tt.wantTriggered {
					t.Errorf("expected trigger but did not fire in time")
				}
			}
		})
	}
}
