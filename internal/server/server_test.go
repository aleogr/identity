// SPDX-License-Identifier: AGPL-3.0-only

package server_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aleogr/identity/internal/server"
)

func TestHealth(t *testing.T) {
	h := server.NewHandler()
	for _, target := range []string{"/health", "/health?probe=1"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil))
		res := rec.Result()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("%s: status = %d", target, res.StatusCode)
		}
		for k, v := range map[string]string{
			"Content-Type":           "application/json",
			"Cache-Control":          "no-store",
			"X-Content-Type-Options": "nosniff",
		} {
			if got := res.Header.Get(k); got != v {
				t.Errorf("%s: %s = %q, want %q", target, k, got, v)
			}
		}
		if body := rec.Body.String(); body != "{\"status\":\"ok\"}\n" {
			t.Errorf("%s: body = %q", target, body)
		}
	}
}

func TestHealthHead(t *testing.T) {
	srv := httptest.NewServer(server.NewHandler())
	defer srv.Close()
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodHead, srv.URL+"/health", nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK || len(body) != 0 || res.Header.Get("Content-Type") != "application/json" {
		t.Errorf("HEAD: status %d, body %q, headers %v", res.StatusCode, body, res.Header)
	}
}

func TestOtherMethodsAndPaths(t *testing.T) {
	h := server.NewHandler()
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), m, "/health", nil))
		if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != "GET, HEAD" {
			t.Errorf("%s: status %d, Allow %q", m, rec.Code, rec.Header().Get("Allow"))
		}
	}
	for _, p := range []string{"/", "/health/", "/healthz", "/version"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, p, nil))
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s: status %d", p, rec.Code)
		}
	}
}

func TestNewSetsLimits(t *testing.T) {
	s := server.New(server.NewHandler(), slog.New(slog.DiscardHandler))
	if s.ReadHeaderTimeout != 10*time.Second || s.ReadTimeout != 30*time.Second ||
		s.WriteTimeout != 30*time.Second || s.IdleTimeout != 120*time.Second ||
		s.MaxHeaderBytes != 64<<10 || s.ErrorLog == nil {
		t.Errorf("limits = %+v", s)
	}
}

func listen(t *testing.T) net.Listener {
	t.Helper()
	ln, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return ln
}

func get(ctx context.Context, url string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, res.Body)
	return res.StatusCode, nil
}

func TestRunServesUntilCancelled(t *testing.T) {
	ln := listen(t)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- server.Run(ctx, server.New(server.NewHandler(), slog.New(slog.DiscardHandler)), ln, time.Second)
	}()
	if code, err := get(t.Context(), "http://"+ln.Addr().String()+"/health"); err != nil || code != http.StatusOK {
		t.Fatalf("health: %d, %v", code, err)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run = %v", err)
	}
}

func TestRunLetsRequestsInFlightFinish(t *testing.T) {
	started := make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /slow", func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})
	ln := listen(t)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- server.Run(ctx, server.New(mux, slog.New(slog.DiscardHandler)), ln, 5*time.Second) }()
	result := make(chan int, 1)
	go func() {
		code, _ := get(context.Background(), "http://"+ln.Addr().String()+"/slow")
		result <- code
	}()
	<-started
	cancel()
	if code := <-result; code != http.StatusOK {
		t.Errorf("request in flight got %d", code)
	}
	if err := <-done; err != nil {
		t.Fatalf("Run = %v", err)
	}
}

func TestRunReportsShutdownTimeout(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /stuck", func(_ http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
	})
	ln := listen(t)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() {
		done <- server.Run(ctx, server.New(mux, slog.New(slog.DiscardHandler)), ln, 50*time.Millisecond)
	}()
	go func() { _, _ = get(context.Background(), "http://"+ln.Addr().String()+"/stuck") }()
	<-started
	cancel()
	err := <-done
	close(release)
	if !errors.Is(err, server.ErrShutdownTimeout) {
		t.Fatalf("Run = %v, want ErrShutdownTimeout", err)
	}
}

func TestRunReturnsServeError(t *testing.T) {
	ln := listen(t)
	_ = ln.Close()
	err := server.Run(t.Context(), server.New(server.NewHandler(), slog.New(slog.DiscardHandler)), ln, time.Second)
	if err == nil {
		t.Fatal("Run on a closed listener returned nil")
	}
}
