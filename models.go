package main

import (
	"context"
	"database/sql"
	"log/slog"
	"time"
)

// Item represents a stored item in the database.
// It contains an ID, name, and timestamp of when it was created.
type Item struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateItemRequest represents the JSON payload for creating a new item.
// It only requires a name field, as other fields are generated server-side.
type CreateItemRequest struct {
	Name string `json:"name"`
}

// initDB initializes the database by creating the required tables if they don't exist.
// It uses the provided context for cancellation and timeout control.
// The function logs the initialization process using the provided structured logger.
func initDB(ctx context.Context, logger *slog.Logger, db *sql.DB) error {
	logger.InfoContext(ctx, "initializing database")
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS items (
			id SERIAL PRIMARY KEY,
			name TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL
		)
	`)
	if err != nil {
		logger.ErrorContext(ctx, "failed to create table",
			slog.String("error", err.Error()),
		)
		return err
	}
	logger.InfoContext(ctx, "database initialized")
	return nil
}
