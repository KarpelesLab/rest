package rest

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

// TestDoHonorsContext checks that a query is aborted when its context
// expires, instead of waiting for the server.
func TestDoHonorsContext(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()
	defer close(release)

	u, _ := url.Parse(srv.URL)
	ctx := context.WithValue(context.Background(), BackendURL, u)
	ctx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := Do(ctx, "Slow:call", "POST", map[string]any{})
	if err == nil {
		t.Fatal("expected an error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Errorf("query took %s, context was not honored", d)
	}
}
