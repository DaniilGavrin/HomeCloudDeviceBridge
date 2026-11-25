package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const (
	defaultHealthAddr = "127.0.0.1:8094"
	shutdownTimeout   = 5 * time.Second
)

type healthResponse struct {
	Status         string `json:"status"`
	Implementation string `json:"implementation"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("Device Bridge stopped with an error", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	healthAddr := os.Getenv("HC_DEVICE_BRIDGE_HEALTH_ADDR")
	if healthAddr == "" {
		healthAddr = defaultHealthAddr
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(healthResponse{
			Status:         "ok",
			Implementation: "stub",
		})
	})

	listener, err := net.Listen("tcp", healthAddr)
	if err != nil {
		return err
	}

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("Device Bridge health endpoint started", "address", listener.Addr().String())
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	// The Backend supervisor closes the child's stdin to request graceful shutdown.
	go func() {
		_, _ = io.Copy(io.Discard, os.Stdin)
		stop()
	}()

	logger.Info(
		"HomeCloud Device Bridge started",
		"implementation", "stub",
		"backend_connection", "not_implemented",
		"device_discovery", "not_implemented",
	)

	select {
	case <-ctx.Done():
		logger.Info("Device Bridge shutdown requested")
	case err := <-serverErrors:
		stop()
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		_ = server.Close()
		return err
	}

	logger.Info("Device Bridge stopped")
	return nil
}
