package main

import (
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

// healthHandler returns an HTTP handler for the health check endpoint.
// It responds with a simple JSON object indicating the service is operational.
func healthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}
}

// createItemHandler returns an HTTP handler for creating new items.
// It accepts a JSON payload with an item name and stores it in the database.
// The handler requires a logger for structured logging and a database connection.
func createItemHandler(logger *slog.Logger, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req CreateItemRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.ErrorContext(ctx, "invalid request body",
				slog.String("error", err.Error()),
			)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		if req.Name == "" {
			logger.ErrorContext(ctx, "name is required")
			http.Error(w, "Name is required", http.StatusBadRequest)
			return
		}

		var item Item
		err := db.QueryRowContext(ctx,
			"INSERT INTO items (name, created_at) VALUES ($1, $2) RETURNING id, name, created_at",
			req.Name,
			time.Now(),
		).Scan(&item.ID, &item.Name, &item.CreatedAt)

		if err != nil {
			logger.ErrorContext(ctx, "error creating item",
				slog.String("error", err.Error()),
			)
			http.Error(w, "Error creating item", http.StatusInternalServerError)
			return
		}

		logger.InfoContext(ctx, "item created",
			slog.Int("id", item.ID),
			slog.String("name", item.Name),
		)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(item)
	}
}

// listItemsHandler returns an HTTP handler for retrieving all items.
// It returns a JSON array of items sorted by creation date in descending order.
// The handler requires a logger for structured logging and a database connection.
func listItemsHandler(logger *slog.Logger, db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		rows, err := db.QueryContext(ctx, "SELECT id, name, created_at FROM items ORDER BY created_at DESC")
		if err != nil {
			logger.ErrorContext(ctx, "error querying items",
				slog.String("error", err.Error()),
			)
			http.Error(w, "Error retrieving items", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var items []Item
		for rows.Next() {
			var item Item
			if err := rows.Scan(&item.ID, &item.Name, &item.CreatedAt); err != nil {
				logger.ErrorContext(ctx, "error scanning item",
					slog.String("error", err.Error()),
				)
				http.Error(w, "Error retrieving items", http.StatusInternalServerError)
				return
			}
			items = append(items, item)
		}

		if err := rows.Err(); err != nil {
			logger.ErrorContext(ctx, "error iterating items",
				slog.String("error", err.Error()),
			)
			http.Error(w, "Error retrieving items", http.StatusInternalServerError)
			return
		}

		logger.InfoContext(ctx, "items retrieved",
			slog.Int("count", len(items)),
		)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(items)
	}
}
