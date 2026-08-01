package handler

import (
	"bytes"
	"encoding/base64"
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

func TestDownloadAsDataURLSuccess(t *testing.T) {
	png := bytes.Repeat([]byte{0x89, 0x50, 0x4E, 0x47}, 10)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(png)
	}))
	t.Cleanup(srv.Close)

	got := downloadAsDataURL(srv.URL)
	if !strings.HasPrefix(got, "data:image/png;base64,") {
		t.Fatalf("expected data URL with image/png, got %q", got[:min(len(got), 40)])
	}
}

func TestDownloadAsDataURLHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	if got := downloadAsDataURL(srv.URL); got != "" {
		t.Fatalf("non-200 should return empty, got %q", got)
	}
}

func TestDownloadAsDataURLMissingContentType(t *testing.T) {
	// PNG 魔数开头，http.DetectContentType 应识别为 image/png
	png := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 故意不设置 Content-Type
		_, _ = w.Write(png)
	}))
	t.Cleanup(srv.Close)

	got := downloadAsDataURL(srv.URL)
	if !strings.HasPrefix(got, "data:image/png;base64,") {
		t.Fatalf("missing Content-Type should fall back to DetectContentType, got %q", got[:min(len(got), 40)])
	}
}

func TestReadImageWithLimitAtBoundary(t *testing.T) {
	// 恰好等于上限，应成功（不 tooLarge）
	data := bytes.Repeat([]byte("x"), maxImageTranslateSize)
	got, tooLarge, err := readImageWithLimit(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tooLarge {
		t.Fatal("data at boundary should not be too large")
	}
	if len(got) != maxImageTranslateSize {
		t.Fatalf("expected %d bytes, got %d", maxImageTranslateSize, len(got))
	}
}

func TestReadImageWithLimitEmpty(t *testing.T) {
	got, tooLarge, err := readImageWithLimit(bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tooLarge {
		t.Fatal("empty input should not be too large")
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 bytes, got %d", len(got))
	}
}

func TestDataURLTooLargeNoComma(t *testing.T) {
	// 无逗号：直接按总长度判断
	small := "data:text/plainabcdefgh"
	if dataURLTooLarge(small) {
		t.Fatal("small dataURL without comma should not be too large")
	}
	huge := "data:" + strings.Repeat("a", maxImageTranslateSize+1)
	if !dataURLTooLarge(huge) {
		t.Fatal("huge dataURL without comma should be too large")
	}
}

func TestDataURLTooLargeBase64(t *testing.T) {
	// base64 payload 解码后超过上限 → too large
	payload := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte("x"), maxImageTranslateSize+1))
	dataURL := "data:image/png;base64," + payload
	if !dataURLTooLarge(dataURL) {
		t.Fatal("base64 dataURL with decoded size over limit should be too large")
	}

	// base64 payload 解码后未超过上限 → not too large
	smallPayload := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte("x"), 100))
	smallDataURL := "data:image/png;base64," + smallPayload
	if dataURLTooLarge(smallDataURL) {
		t.Fatal("small base64 dataURL should not be too large")
	}
}

func TestDataURLTooLargePlain(t *testing.T) {
	// 非 base64 的 dataURL：按 payload 原始长度判断
	small := "data:text/plain,hello"
	if dataURLTooLarge(small) {
		t.Fatal("small plain dataURL should not be too large")
	}
	huge := "data:text/plain," + strings.Repeat("a", maxImageTranslateSize+1)
	if !dataURLTooLarge(huge) {
		t.Fatal("huge plain dataURL should be too large")
	}
}
