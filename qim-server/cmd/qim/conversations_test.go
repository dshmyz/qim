package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestConversationsListUsesPageSizeQuery(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"list": []any{}},
		})
	}))
	t.Cleanup(srv.Close)
	writeTestConfig(t, srv.URL, "qbot_test_token")

	cmd := newConversationsListCmd()
	cmd.SetArgs([]string{"--limit", "50"})
	cmd.Run(cmd, nil)

	if !strings.Contains(gotQuery, "page_size=50") {
		t.Fatalf("expected page_size=50 query, got %q", gotQuery)
	}
	if strings.Contains(gotQuery, "limit=50") {
		t.Fatalf("server does not consume limit query, got %q", gotQuery)
	}
}
