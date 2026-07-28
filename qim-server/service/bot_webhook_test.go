package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSendBotWebhook_PostsSignedPayload(t *testing.T) {
	secret := "wh_secret_abc"

	var received BotWebhookPayload
	var gotSig, gotEvent, gotDelivery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &received)
		gotSig = r.Header.Get("X-QIM-Signature")
		gotEvent = r.Header.Get("X-QIM-Event")
		gotDelivery = r.Header.Get("X-QIM-Delivery")

		// 校验签名：HMAC-SHA256(body, secret)
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		assert.Equal(t, hex.EncodeToString(mac.Sum(nil)), gotSig)

		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	payload := BotWebhookPayload{
		BotID:     7,
		ThreadID:  456,
		MessageID: 1001,
		UserID:    123,
		Content:   "先别回滚",
		MsgType:   "text",
	}
	err := SendBotWebhook(srv.URL, secret, payload)
	assert.NoError(t, err)
	assert.Equal(t, "bot.message", gotEvent)
	assert.Equal(t, uint(7), received.BotID)
	assert.Equal(t, uint(456), received.ThreadID)
	assert.Equal(t, "先别回滚", received.Content)
	assert.NotEmpty(t, received.DeliveryID)
	assert.Equal(t, received.DeliveryID, gotDelivery)
}

func TestSendBotWebhook_NoSecretOmitsSignature(t *testing.T) {
	var gotSig string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSig = r.Header.Get("X-QIM-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	err := SendBotWebhook(srv.URL, "", BotWebhookPayload{BotID: 1})
	assert.NoError(t, err)
	assert.Empty(t, gotSig)
}

func TestSendBotWebhook_ReturnsErrorOnNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	err := SendBotWebhook(srv.URL, "secret", BotWebhookPayload{BotID: 1})
	assert.Error(t, err)
}

func TestSendBotWebhook_MissingURL(t *testing.T) {
	err := SendBotWebhook("", "secret", BotWebhookPayload{BotID: 1})
	assert.Error(t, err)
}
