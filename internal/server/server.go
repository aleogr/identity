// SPDX-License-Identifier: AGPL-3.0-only

// Package server serves the platform's HTTP endpoints and shuts down
// gracefully.
package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// Limits of every connection (docs/threat-model.md, X-04).
const (
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 30 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 120 * time.Second
	maxHeaderBytes    = 64 << 10
)

// ErrShutdownTimeout reports that requests in flight did not finish within the
// shutdown timeout; their connections were closed.
var ErrShutdownTimeout = errors.New("server: requests in flight did not finish before the shutdown timeout")

// NewHandler returns the handler for every route the server answers.
func NewHandler() http.Handler {
	mux := http.NewServeMux()
	// A GET pattern also serves HEAD; other methods get 405 with Allow: GET, HEAD.
	mux.HandleFunc("GET /health", health)
	return mux
}

// health reports that the process is up. It carries no build identifier:
// that is the operator-only version endpoint's job (F3).
func health(w http.ResponseWriter, _ *http.Request) {
	h := w.Header()
	h.Set("Content-Type", "application/json")
	h.Set("Cache-Control", "no-store")
	h.Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "{\"status\":\"ok\"}\n")
}

// New returns a server for handler with the platform's limits, logging its
// own errors through logger.
func New(handler http.Handler, logger *slog.Logger) *http.Server {
	return &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
	}
}

// Run serves on ln until ctx is done, then stops accepting connections and
// waits up to shutdownTimeout for requests in flight. It returns nil after a
// clean shutdown, ErrShutdownTimeout when requests were cut off, and the
// serving error when serving stopped by itself.
func Run(ctx context.Context, srv *http.Server, ln net.Listener, shutdownTimeout time.Duration) error {
	served := make(chan error, 1)
	go func() { served <- srv.Serve(ln) }()

	select {
	case err := <-served:
		return fmt.Errorf("server: %w", err)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		_ = srv.Close()
		if errors.Is(err, context.DeadlineExceeded) {
			return ErrShutdownTimeout
		}
		return fmt.Errorf("server: shutdown: %w", err)
	}
	if err := <-served; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server: %w", err)
	}
	return nil
}
