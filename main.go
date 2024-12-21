package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
)

// Constants for server configuration
const (
	// shutdownTimeout is the duration of inactivity after which the server will shut down
	shutdownTimeout = 5 * time.Minute
	// port is the default port number for the HTTP server
	port = "8080"
)

// setupServer initializes and configures the HTTP server with all routes and middleware.
// It takes a context for cancellation, a structured logger, database connection, and activity tracker.
// Returns a configured http.Server ready to be started.
func setupServer(ctx context.Context, logger *slog.Logger, db *sql.DB, tracker *activityTracker) *http.Server {
	// Create a new HTTP request multiplexer
	mux := http.NewServeMux()

	// Add routes for health check, item creation, and item listing
	mux.Handle("/health", healthHandler())
	mux.Handle("/items", createItemHandler(logger, db))
	mux.Handle("/items/", listItemsHandler(logger, db))

	// Wrap all handlers with activity tracking
	handler := withActivityTracking(ctx, logger, tracker, mux)

	// Return a configured HTTP server
	return &http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}
}

// main is the entry point of the application. It:
// - Initializes the logger and database connection
// - Sets up the HTTP server with all routes
// - Starts the activity monitoring
// - Handles graceful shutdown on signals
func main() {
	// Create a background context for the application
	ctx := context.Background()

	// Initialize a JSON logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Get the database URL from the environment variable
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		// Log an error and exit if the database URL is not provided
		logger.ErrorContext(ctx, "DATABASE_URL environment variable is required")
		os.Exit(1)
	}

	// Open a database connection
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		// Log an error and exit if the database connection fails
		logger.ErrorContext(ctx, "failed to connect to database", 
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
	defer db.Close()

	// Ping the database to verify the connection
	if err := db.Ping(); err != nil {
		// Log an error and exit if the database ping fails
		logger.ErrorContext(ctx, "failed to ping database", 
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	// Initialize the database schema
	if err := initDB(ctx, logger, db); err != nil {
		// Log an error and exit if the database initialization fails
		logger.ErrorContext(ctx, "failed to initialize database", 
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	// Initialize activity tracking
	tracker := newActivityTracker(logger)
	shutdown := make(chan struct{})

	// Setup and start activity monitoring
	go monitorActivity(ctx, logger, tracker, shutdown)

	// Initialize the HTTP server
	srv := setupServer(ctx, logger, db, tracker)

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start a goroutine to handle shutdown
	go func() {
		select {
		case <-sigChan:
			// Log a message when a shutdown signal is received
			logger.InfoContext(ctx, "received shutdown signal")
		case <-shutdown:
			// Log a message when the server is shutting down due to inactivity
			logger.InfoContext(ctx, "shutting down due to inactivity")
		}

		// Create a shutdown context with a timeout
		shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		// Shut down the HTTP server
		if err := srv.Shutdown(shutdownCtx); err != nil {
			// Log an error if the shutdown fails
			logger.ErrorContext(ctx, "error during shutdown",
				slog.String("error", err.Error()),
			)
		}
	}()

	// Log a message when the server is starting
	logger.InfoContext(ctx, "server starting", slog.String("port", port))

	// Start the HTTP server
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		// Log an error and exit if the server fails to start
		logger.ErrorContext(ctx, "server error",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
}
