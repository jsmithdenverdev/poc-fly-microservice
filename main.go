package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	_ "github.com/lib/pq"
)

const (
	shutdownTimeout = 5 * time.Minute
	port           = "8080"
)

type Item struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateItemRequest struct {
	Name string `json:"name"`
}

type activityTracker struct {
	lastActivity time.Time
	mu          sync.RWMutex
}

func newActivityTracker() *activityTracker {
	return &activityTracker{
		lastActivity: time.Now(),
	}
}

func (t *activityTracker) update() {
	t.mu.Lock()
	t.lastActivity = time.Now()
	t.mu.Unlock()
}

func (t *activityTracker) timeSinceLastActivity() time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return time.Since(t.lastActivity)
}

func withActivityTracking(tracker *activityTracker, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracker.update()
		next.ServeHTTP(w, r)
	})
}

func healthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}

func createItemHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req CreateItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		if req.Name == "" {
			http.Error(w, "Name is required", http.StatusBadRequest)
			return
		}

		var item Item
		err := db.QueryRow(
			"INSERT INTO items (name, created_at) VALUES ($1, $2) RETURNING id, name, created_at",
			req.Name,
			time.Now(),
		).Scan(&item.ID, &item.Name, &item.CreatedAt)

		if err != nil {
			log.Printf("Error creating item: %v", err)
			http.Error(w, "Error creating item", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(item)
	}
}

func listItemsHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		rows, err := db.Query("SELECT id, name, created_at FROM items ORDER BY created_at DESC")
		if err != nil {
			log.Printf("Error querying items: %v", err)
			http.Error(w, "Error retrieving items", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var items []Item
		for rows.Next() {
			var item Item
			if err := rows.Scan(&item.ID, &item.Name, &item.CreatedAt); err != nil {
				log.Printf("Error scanning item: %v", err)
				http.Error(w, "Error retrieving items", http.StatusInternalServerError)
				return
			}
			items = append(items, item)
		}

		if err := rows.Err(); err != nil {
			log.Printf("Error iterating items: %v", err)
			http.Error(w, "Error retrieving items", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	}
}

func initDB(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS items (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL
		)
	`)
	return err
}

func monitorActivity(tracker *activityTracker, shutdown chan<- struct{}) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if tracker.timeSinceLastActivity() > shutdownTimeout {
				log.Printf("No activity for %v, initiating shutdown", shutdownTimeout)
				close(shutdown)
				return
			}
		}
	}
}

func setupServer(db *sql.DB, tracker *activityTracker) *http.Server {
	mux := http.NewServeMux()
	
	// Add routes
	mux.Handle("/health", healthHandler())
	mux.Handle("/items", createItemHandler(db))
	mux.Handle("/items/", listItemsHandler(db))

	// Wrap all handlers with activity tracking
	handler := withActivityTracking(tracker, mux)

	return &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}
}

func main() {
	// TODO: Load configuration from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	if err := initDB(db); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Initialize activity tracking
	tracker := newActivityTracker()
	shutdown := make(chan struct{})

	// Setup and start activity monitoring
	go monitorActivity(tracker, shutdown)

	// Initialize server
	srv := setupServer(db, tracker)

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-sigChan:
			log.Println("Received shutdown signal")
		case <-shutdown:
			log.Println("Shutting down due to inactivity")
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Error during shutdown: %v", err)
		}
	}()

	log.Printf("Server starting on port %s", port)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}
