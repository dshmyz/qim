package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDownloadAsDataURLRejectsOversizedImageWithoutTruncating(t *testing.T) {
	payload := strings.Repeat("a", maxImageTranslateSize+1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte(payload))
	}))
	t.Cleanup(srv.Close)

	if got := downloadAsDataURL(srv.URL); got != "" {
		t.Fatalf("oversized image should be rejected, got data URL length %d", len(got))
	}
}
