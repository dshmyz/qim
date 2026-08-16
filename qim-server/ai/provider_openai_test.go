package ai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

// TestOpenAIEmbeddingModelSelection 锁定 embedding 模型的单源语义：
// embedding 模型没有独立配置，统一来自 config.Model——
//   - 经 WithModel（router 显式路由 embedding）→ 用 router 传入的模型；
//   - 直接调 Embedding()（无 embedding 路由的直连回退）→ 用 provider 自身配置的 Model。
func TestOpenAIEmbeddingModelSelection(t *testing.T) {
	var (
		mu    sync.Mutex
		model string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/embeddings" {
			http.Error(w, "unexpected path: "+r.URL.Path, http.StatusNotFound)
			return
		}
		var req struct {
			Model string `json:"model"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		mu.Lock()
		model = req.Model
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"model": req.Model,
			"data": []map[string]interface{}{
				{"embedding": []float32{0.1, 0.2, 0.3}, "index": 0},
			},
		})
	}))
	defer server.Close()

	provider := NewOpenAIProvider(ProviderConfig{
		APIKey:      "test-key",
		Model:       "chat-default",
		BaseURL:     server.URL,
		ExtraParams: map[string]interface{}{},
	})

	t.Run("router 路由的 WithModel 优先", func(t *testing.T) {
		if _, err := provider.WithModel("router-embed").Embedding("hello"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		mu.Lock()
		defer mu.Unlock()
		if model != "router-embed" {
			t.Fatalf("WithModel 路由后应发 router 模型，实际发了 %q", model)
		}
	})

	t.Run("直连路径回退到配置的 Model", func(t *testing.T) {
		if _, err := provider.Embedding("hello"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		mu.Lock()
		defer mu.Unlock()
		if model != "chat-default" {
			t.Fatalf("直连路径应回退配置的 Model，实际发了 %q", model)
		}
	})
}
