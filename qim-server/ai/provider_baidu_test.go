package ai

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBaiduChatStreamWithContextCancelsAccessTokenRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/oauth/2.0/token") {
			select {
			case <-r.Context().Done():
				return
			case <-time.After(200 * time.Millisecond):
				http.Error(w, "token request was not canceled", http.StatusGatewayTimeout)
				return
			}
		}

		t.Fatalf("unexpected request path: %s", r.URL.Path)
	}))
	defer server.Close()

	provider := NewBaiduProvider(ProviderConfig{
		APIKey:      "client-id",
		APISecret:   "client-secret",
		Model:       "ernie-test",
		BaseURL:     server.URL,
		ExtraParams: map[string]interface{}{},
	})
	provider.Client.Timeout = 250 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := provider.ChatStreamWithContext(ctx, []Message{{Role: "user", Content: "hello"}}, func(chunk StreamChunk) error {
		return nil
	})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected context deadline error")
	}
	if elapsed > 100*time.Millisecond {
		t.Fatalf("expected token request to be canceled quickly, took %s: %v", elapsed, err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected context deadline exceeded, got %v", err)
	}
}
