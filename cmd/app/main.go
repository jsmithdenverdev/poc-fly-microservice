package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/caarlos0/env"
	"github.com/jake/poc-fly-microservice/internal/app"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "run failed: %s\n", err.Error())
	}
}

func run(ctx context.Context) error {
	var cfg app.Config

	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt)
	defer cancel()

	if err := env.Parse(&cfg); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	svr := app.NewServer(ctx, cancel, cfg, logger)
	httpServer := &http.Server{
		Addr:    net.JoinHostPort(cfg.AppHost, cfg.AppPort),
		Handler: svr,
	}

	// Start the server in a separate goroutine
	go func() {
		logger.InfoContext(
			context.Background(),
			"server started",
			slog.String("address", httpServer.Addr))

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "error listening and serving: %s\n", err)
		}
	}()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		<-ctx.Done()
		shutdownCtx := context.Background()
		shutdownCtx, cancel := context.WithTimeout(shutdownCtx, 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			fmt.Fprintf(os.Stderr, "error shutting down http server: %s\n", err)
		}
	}()

	wg.Wait()

	return nil
}
