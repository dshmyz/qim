package qapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestP3ConversationDetailParse 验证群会话详情里的 members 与 announcement 解析。
func TestP3ConversationDetailParse(t *testing.T) {
	raw := `{
		"id": 106,
		"type": "group",
		"name": "测试群",
		"announcement": "本周五团建",
		"creator_id": 7,
		"members": [
			{"user_id": 7, "role": "owner", "user": {"id": 7, "nickname": "我", "username": "me", "avatar": ""}},
			{"user_id": 8, "role": "member", "user": {"id": 8, "nickname": "张三", "username": "zhangsan", "avatar": ""}}
		]
	}`
	var conv Conversation
	if err := json.Unmarshal([]byte(raw), &conv); err != nil {
		t.Fatalf("解析会话详情失败: %v", err)
	}
	if conv.ID.String() != "106" {
		t.Errorf("id 错: %s", conv.ID.String())
	}
	if conv.Announcement != "本周五团建" {
		t.Errorf("announcement 错: %q", conv.Announcement)
	}
	if conv.GroupID != 0 {
		t.Errorf("group_id 不应从详情顶层解析: %d", conv.GroupID)
	}
	if len(conv.Members) != 2 {
		t.Fatalf("members 数量错: %d", len(conv.Members))
	}
	if conv.Members[0].UserID != 7 || conv.Members[0].Role != "owner" {
		t.Errorf("成员0 错: %+v", conv.Members[0])
	}
	if conv.Members[1].DisplayName() != "张三" {
		t.Errorf("成员1 名字错: %q", conv.Members[1].DisplayName())
	}
}

// TestP3SetPinRequest 验证置顶切换发出正确的 PUT 与请求体。
func TestP3SetPinRequest(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || !strings.HasSuffix(r.URL.Path, "/conversations/106/pin") {
			w.WriteHeader(404)
			return
		}
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &body)
		io.WriteString(w, `{"code":0,"message":"ok","data":{"is_pinned":true}}`)
	}))
	defer srv.Close()
	c := New(srv.URL)
	if err := c.SetConversationPin(context.Background(), ID("106"), true); err != nil {
		t.Fatalf("置顶失败: %v", err)
	}
	if body["is_pinned"] != true {
		t.Errorf("请求体 is_pinned 错: %+v", body)
	}
}

// TestP3SetMuteRequest 验证免打扰切换发出正确的 PUT 与请求体。
func TestP3SetMuteRequest(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || !strings.HasSuffix(r.URL.Path, "/conversations/106/mute") {
			w.WriteHeader(404)
			return
		}
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &body)
		io.WriteString(w, `{"code":0}`)
	}))
	defer srv.Close()
	c := New(srv.URL)
	if err := c.SetConversationMute(context.Background(), ID("106"), true); err != nil {
		t.Fatalf("免打扰失败: %v", err)
	}
	if body["muted"] != true {
		t.Errorf("请求体 muted 错: %+v", body)
	}
}

// TestP3SetAnnouncementRequest 验证群公告写入发出正确的 PUT 与请求体。
// 对齐后端：PUT /groups/:id/announcement，:id 是会话 ID，body 只有 announcement。
func TestP3SetAnnouncementRequest(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || !strings.HasSuffix(r.URL.Path, "/groups/106/announcement") {
			w.WriteHeader(404)
			return
		}
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &body)
		io.WriteString(w, `{"code":0,"message":"ok"}`)
	}))
	defer srv.Close()
	c := New(srv.URL)
	if err := c.SetGroupAnnouncement(context.Background(), ID("106"), "本周五团建"); err != nil {
		t.Fatalf("公告保存失败: %v", err)
	}
	if body["announcement"] != "本周五团建" {
		t.Errorf("请求体 announcement 错: %+v", body)
	}
}

// TestP3MessageReadUsers 验证已读详情端点与响应解析：
// GET /messages/:id/read-users → {read_users, read_count, total_members}。
func TestP3MessageReadUsers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasSuffix(r.URL.Path, "/messages/55/read-users") {
			w.WriteHeader(404)
			return
		}
		io.WriteString(w, `{"code":0,"message":"ok","data":{
			"read_users":[
				{"id":8,"username":"zhangsan","nickname":"张三","avatar":""},
				{"id":9,"username":"","nickname":"","avatar":""}
			],
			"read_count":2,
			"total_members":5}}`)
	}))
	defer srv.Close()
	c := New(srv.URL)
	info, err := c.MessageReadUsers(context.Background(), ID("55"))
	if err != nil {
		t.Fatalf("拉取已读失败: %v", err)
	}
	if info.ReadCount != 2 || info.TotalMembers != 5 {
		t.Errorf("计数错: read=%d total=%d", info.ReadCount, info.TotalMembers)
	}
	if len(info.ReadUsers) != 2 {
		t.Fatalf("read_users 数量错: %d", len(info.ReadUsers))
	}
	if info.ReadUsers[0].DisplayName() != "张三" {
		t.Errorf("用户名错: %q", info.ReadUsers[0].DisplayName())
	}
	if info.ReadUsers[1].DisplayName() != "用户9" {
		t.Errorf("空名兜底错: %q", info.ReadUsers[1].DisplayName())
	}
}

// TestP3ConvListPinMuted 验证会话列表里的 is_pinned / muted / announcement 解析。
func TestP3ConvListPinMuted(t *testing.T) {
	raw := `[
		{"id":1,"type":"group","name":"A","is_pinned":true,"muted":false,"announcement":"公告A"},
		{"id":2,"type":"single","name":"B","is_pinned":false,"muted":true}
	]`
	var list []Conversation
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		t.Fatalf("解析列表失败: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("列表长度错: %d", len(list))
	}
	if !list[0].PinnedNow() {
		t.Errorf("列表 is_pinned 未解析: %+v", list[0])
	}
	if list[0].Announcement != "公告A" {
		t.Errorf("列表 announcement 未解析: %q", list[0].Announcement)
	}
	if !list[1].Muted {
		t.Errorf("列表 muted 未解析: %+v", list[1])
	}
}
