package service

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendRemind_Success(t *testing.T) {
	var receivedBody string
	var receivedSignature string
	var receivedEvent string
	var receivedDelivery string
	var receivedTimestamp string
	var receivedCustomHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		receivedBody = string(buf[:n])
		receivedSignature = r.Header.Get("X-QIM-Signature")
		receivedEvent = r.Header.Get("X-QIM-Event")
		receivedDelivery = r.Header.Get("X-QIM-Delivery")
		receivedTimestamp = r.Header.Get("X-QIM-Timestamp")
		receivedCustomHeader = r.Header.Get("X-Custom")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &WebhookConfig{
		Enabled:      true,
		URL:          server.URL,
		Method:       "POST",
		Secret:       "test-secret",
		Headers:      map[string]string{"X-Custom": "test"},
		BodyTemplate: `{"text":"{{.SenderNickname}} 提醒你：{{.MessageContentPreview}}"}`,
	}
	data := RemindData{
		SenderNickname:        "Alice",
		MessageContentPreview: "会议纪要见附件",
	}

	err := SendRemind(cfg, data)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if receivedEvent != "message.remind" {
		t.Errorf("expected event message.remind, got %s", receivedEvent)
	}
	if receivedDelivery == "" {
		t.Error("expected non-empty delivery ID")
	}
	if receivedTimestamp == "" {
		t.Error("expected non-empty timestamp")
	}
	if receivedSignature == "" {
		t.Error("expected non-empty signature")
	}
	if receivedCustomHeader != "test" {
		t.Errorf("expected custom header test, got %s", receivedCustomHeader)
	}
	if !strings.Contains(receivedBody, "Alice") {
		t.Errorf("expected body to contain Alice, got %s", receivedBody)
	}
	if !strings.Contains(receivedBody, "会议纪要见附件") {
		t.Errorf("expected body to contain message preview, got %s", receivedBody)
	}
}

func TestSendRemind_NoSecret_NoSignatureHeader(t *testing.T) {
	var receivedSignature string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSignature = r.Header.Get("X-QIM-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &WebhookConfig{
		Enabled:      true,
		URL:          server.URL,
		Method:       "POST",
		Secret:       "",
		BodyTemplate: `{"text":"test"}`,
	}
	err := SendRemind(cfg, RemindData{})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if receivedSignature != "" {
		t.Errorf("expected empty signature when no secret, got %s", receivedSignature)
	}
}

func TestSendRemind_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	cfg := &WebhookConfig{
		Enabled:      true,
		URL:          server.URL,
		Method:       "POST",
		BodyTemplate: `{"text":"test"}`,
	}
	err := SendRemind(cfg, RemindData{})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
	if !strings.Contains(err.Error(), "HTTP 500") {
		t.Errorf("expected HTTP 500 in error, got %v", err)
	}
}

func TestSendRemind_HTTP404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := &WebhookConfig{
		Enabled:      true,
		URL:          server.URL,
		Method:       "POST",
		BodyTemplate: `{"text":"test"}`,
	}
	err := SendRemind(cfg, RemindData{})
	if err == nil {
		t.Fatal("expected error for HTTP 404")
	}
	if !strings.Contains(err.Error(), "HTTP 404") {
		t.Errorf("expected HTTP 404 in error, got %v", err)
	}
}

func TestSendRemind_TemplateParseError(t *testing.T) {
	cfg := &WebhookConfig{
		Enabled:      true,
		URL:          "http://localhost",
		Method:       "POST",
		BodyTemplate: `{{.InvalidSyntax`,
	}
	err := SendRemind(cfg, RemindData{})
	if err == nil {
		t.Fatal("expected template parse error")
	}
	if !strings.Contains(err.Error(), "body_template") {
		t.Errorf("expected body_template error, got %v", err)
	}
}

func TestSendRemind_ConnectionError(t *testing.T) {
	cfg := &WebhookConfig{
		Enabled:      true,
		URL:          "http://127.0.0.1:1", // 不可达端口
		Method:       "POST",
		BodyTemplate: `{"text":"test"}`,
	}
	err := SendRemind(cfg, RemindData{})
	if err == nil {
		t.Fatal("expected connection error")
	}
	if !strings.Contains(err.Error(), "webhook 调用失败") {
		t.Errorf("expected webhook 调用失败, got %v", err)
	}
}

func TestSendRemind_MessageContentPreviewTruncated(t *testing.T) {
	var receivedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 8192)
		n, _ := r.Body.Read(buf)
		receivedBody = string(buf[:n])
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &WebhookConfig{
		Enabled:      true,
		URL:          server.URL,
		Method:       "POST",
		BodyTemplate: `{"text":"{{.MessageContentPreview}}"}`,
	}
	longContent := strings.Repeat("一", 200) // 200 个中文字符
	data := RemindData{
		MessageContentPreview: longContent,
	}
	err := SendRemind(cfg, data)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	// 截断后应为 100 字符 + "..."
	if strings.Contains(receivedBody, strings.Repeat("一", 101)) {
		t.Error("expected content to be truncated to 100 characters")
	}
	if !strings.Contains(receivedBody, "...") {
		t.Error("expected truncated content to end with ...")
	}
}

func TestTruncateString_ShortString(t *testing.T) {
	if got := truncateString("hello", 10); got != "hello" {
		t.Errorf("expected hello, got %s", got)
	}
}

func TestTruncateString_ExactLength(t *testing.T) {
	if got := truncateString("hello", 5); got != "hello" {
		t.Errorf("expected hello, got %s", got)
	}
}

func TestTruncateString_LongString(t *testing.T) {
	if got := truncateString("hello world", 5); got != "hello..." {
		t.Errorf("expected hello..., got %s", got)
	}
}

func TestTruncateString_ChineseCharacters(t *testing.T) {
	// 中文字符按 rune 计数
	long := strings.Repeat("一", 10)
	got := truncateString(long, 5)
	if !strings.HasPrefix(got, "一一一一一") {
		t.Errorf("expected 5 Chinese chars prefix, got %s", got)
	}
	if !strings.HasSuffix(got, "...") {
		t.Errorf("expected suffix ..., got %s", got)
	}
}
