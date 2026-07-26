// Command echo-webhook 是一个本地测试用 webhook 接收端，用于联调 bot 卡片交互。
//
// 它监听 :9100，接收 qim-server 的 bot webhook 投递（event=bot.card_action 等），
// 打印请求头与 JSON body，便于观察"用户点击卡片按钮 -> 后端转发到 agent"的闭环。
//
// 仅用于本地测试，勿用于生产。
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	addr := ":9100"
	if a := os.Getenv("ECHO_WEBHOOK_ADDR"); a != "" {
		addr = a
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("[echo] read body err: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// 签名头（qim-server 投递时附带，便于核对 webhook_secret 是否一致）
		// 签名算法：HMAC-SHA256(webhook_secret, raw_body)，纯 body 不含 timestamp
		sig := r.Header.Get("X-QIM-Signature")
		event := r.Header.Get("X-QIM-Event")
		ts := r.Header.Get("X-QIM-Timestamp")
		delivery := r.Header.Get("X-QIM-Delivery")

		// 尝试美化 JSON；失败则原样打印
		pretty := body
		var buf map[string]any
		if json.Unmarshal(body, &buf) == nil {
			p, err := json.MarshalIndent(buf, "", "  ")
			if err == nil {
				pretty = p
			}
		}

		fmt.Println("\n========== webhook received ==========")
		fmt.Printf("time:     %s\n", time.Now().Format("15:04:05.000"))
		fmt.Printf("method:   %s %s\n", r.Method, r.URL.Path)
		fmt.Printf("event:    %s\n", event)
		fmt.Printf("delivery: %s\n", delivery)
		fmt.Printf("sig:      %s\n", sig)
		fmt.Printf("ts:       %s\n", ts)
		fmt.Printf("payload:\n%s\n", string(pretty))
		fmt.Println("======================================\n")

		// 返回 200，让 qim-server 标记投递成功；否则会走重试/死信
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	// 健康检查，确认服务在跑
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	log.Printf("[echo-webhook] listening on %s (POST / -> echo; GET /healthz -> ok)", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("listen failed: %v", err)
	}
}
