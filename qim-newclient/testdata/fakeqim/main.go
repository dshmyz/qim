// 一个仅供开发自测的假 qim 服务端。放在 testdata 下，go build 会忽略它。
//
//	go run ./testdata/fakeqim
//	go run . -probe-api -server http://127.0.0.1:18080 -user grace -pass pw
//
// 它实现登录、会话列表、历史消息、WS 认证与推送四件事，用来在没有真实
// 后端的情况下验证 Go 客户端的协议实现是否正确。真实后端可能还有
// 版本门槛、2FA、多设备互踢等行为，这里一概不模拟。
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Password != "pw" {
			writeJSON(w, map[string]any{"code": 1001, "message": "用户名或密码错误"})
			return
		}
		writeJSON(w, map[string]any{"code": 0, "message": "ok", "data": map[string]any{
			"token":         "fake_token_abc",
			"refresh_token": "fake_refresh_abc",
			"user":          map[string]any{"id": 7, "username": body.Username, "nickname": "测试用户"},
		}})
	})

	mux.HandleFunc("/api/v1/conversations", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
			writeJSON(w, map[string]any{"code": 401, "message": "未授权"})
			return
		}
		writeJSON(w, map[string]any{"code": 0, "data": []map[string]any{
			{"id": 101, "name": "技术交流群", "type": "group", "unread_count": 3, "timestamp": time.Now().Unix(),
				"lastMessage": map[string]any{"id": 9001, "type": "text", "content": "假服务端的最后一条消息", "conversation_id": 101}},
			{"id": "c2", "name": "系统通知", "type": "single", "unread_count": 0, "timestamp": time.Now().Unix() - 60,
				"lastMessage": map[string]any{"id": 9002, "type": "image", "content": "", "conversation_id": "c2"}},
		}})
	})

	mux.HandleFunc("/api/v1/conversations/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/read") {
			writeJSON(w, map[string]any{"code": 0})
			return
		}
		writeJSON(w, map[string]any{"code": 0, "data": map[string]any{
			"messages": []map[string]any{
				{"id": 1, "conversation_id": 101, "sender_id": 8, "type": "text", "content": "早上好", "timestamp": 1700000001},
				{"id": 2, "conversation_id": 101, "sender_id": 7, "type": "text", "content": "早，今天的构建挂了", "timestamp": 1700000002, "isSelf": true},
				{"id": 3, "conversation_id": 101, "sender_id": 8, "type": "file", "content": "", "file_name": "build.log", "timestamp": 1700000003},
			},
			"pagination": map[string]any{"current_page": 1, "total_pages": 1, "total": 3, "page_size": 20},
		}})
	})

	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	mux.HandleFunc("/api/v1/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		var first struct {
			Type string `json:"type"`
			Data struct {
				Token string `json:"token"`
			} `json:"data"`
		}
		if err := conn.ReadJSON(&first); err != nil || first.Type != "auth" {
			return
		}
		log.Printf("WS 认证: token=%s", first.Data.Token)
		_ = conn.WriteJSON(map[string]any{"type": "auth_success", "data": map[string]any{"user_id": 7}})

		// 隔一会儿推两条，验证客户端能收
		time.Sleep(500 * time.Millisecond)
		_ = conn.WriteJSON(map[string]any{"type": "new_message", "data": map[string]any{
			"id": 9100, "conversation_id": 101, "sender_id": 8, "type": "text",
			"content": "这是假服务端推送的新消息", "timestamp": time.Now().Unix(),
		}})
		time.Sleep(300 * time.Millisecond)
		_ = conn.WriteJSON(map[string]any{"type": "message_read", "data": map[string]any{
			"conversation_id": 101, "user_id": 8, "message_ids": []int{2},
		}})

		// 保持连接，等客户端断开
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	})

	addr := "127.0.0.1:18080"
	log.Printf("假 qim 服务端监听 http://%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
