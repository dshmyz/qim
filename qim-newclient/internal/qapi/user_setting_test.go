package qapi

// 通用用户设置契约测试（GET/PUT /api/v1/user-settings/:key）。
// 后端 user_setting_handler.go：GET 未设置返回 value=null + has_value=false；
// PUT 请求体 {"value": <任意 JSON>}，整存整取。

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUserSettingGetUnset(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasSuffix(r.URL.Path, "/user-settings/quick_replies") {
			t.Errorf("请求不对: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"code":0,"message":"ok","data":{"value":null,"has_value":false}}`)
	}))
	defer srv.Close()

	c := New(srv.URL)
	v, err := c.GetUserSetting(context.Background(), "quick_replies")
	if err != nil {
		t.Fatalf("读取失败: %v", err)
	}
	if v.HasValue {
		t.Fatalf("未设置时应 has_value=false: %+v", v)
	}
	s := strings.TrimSpace(string(v.Value))
	if s != "" && s != "null" {
		t.Fatalf("value 应为 null: %s", v.Value)
	}
}

func TestUserSettingGetArray(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"code":0,"data":{"value":["收到","好的"],"has_value":true}}`)
	}))
	defer srv.Close()

	c := New(srv.URL)
	v, err := c.GetUserSetting(context.Background(), "quick_replies")
	if err != nil {
		t.Fatalf("读取失败: %v", err)
	}
	if !v.HasValue {
		t.Fatal("自定义后应 has_value=true")
	}
	var items []string
	if err := json.Unmarshal(v.Value, &items); err != nil {
		t.Fatalf("value 不是字符串数组: %v (%s)", err, v.Value)
	}
	if len(items) != 2 || items[0] != "收到" {
		t.Fatalf("数组解析错: %v", items)
	}
}

func TestUserSettingPut(t *testing.T) {
	var gotBody struct {
		Value []string `json:"value"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || !strings.HasSuffix(r.URL.Path, "/user-settings/quick_replies") {
			t.Errorf("请求不对: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"code":0,"message":"保存成功","data":null}`)
	}))
	defer srv.Close()

	c := New(srv.URL)
	if err := c.PutUserSetting(context.Background(), "quick_replies", []string{"嗯", "马上"}); err != nil {
		t.Fatalf("保存失败: %v", err)
	}
	if len(gotBody.Value) != 2 || gotBody.Value[0] != "嗯" || gotBody.Value[1] != "马上" {
		t.Fatalf("PUT body 错: %+v", gotBody)
	}
}
