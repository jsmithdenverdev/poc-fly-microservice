package app

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"text/template"
)

// healthHandler handles health checks
func healthHandler(cfg Config, h slog.Handler, t *template.Template) http.Handler {
	type healthResponse struct {
		Message string `json:"message"`
		Region  string `json:"region"`
	}

	logger := slog.New(h)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.InfoContext(r.Context(), "health check")
		acceptHeader := r.Header.Get("Accept")

		response := healthResponse{
			Message: "Hello from Fly.io!",
			Region:  cfg.FlyRegion,
		}

		if acceptHeader == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			b, err := json.Marshal(response)

			if err != nil {
				http.Error(w, "error marshaling health response", http.StatusInternalServerError)
				return
			}

			w.Write(b)
		} else {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)

			err := t.ExecuteTemplate(w, "health.html.tmpl", response)

			if err != nil {
				http.Error(w, "Error rendering template", http.StatusInternalServerError)
			}
		}
	})
}
