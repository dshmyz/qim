package qapi

// 搜索与会话创建。三个搜索接口在 qim-server 里的参数名/返回形状各不相同：
//   - GET /conversations/search?query=  → data 是数组 {id,type,name,avatar}
//   - GET /messages/search?keyword=&page=&pageSize= → data.list，且每条多带 conversation_meta
//   - GET /users/search?q= → data 是数组，用户和群混在一起
// 这些差异全部收在这个文件里，UI 层只看统一的 SearchHits。

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// SearchConvHit 是会话搜索结果。后端只给展示用的最小字段。
type SearchConvHit struct {
	ID     ID     `json:"id"`
	Type   string `json:"type"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

// SearchUserHit 是用户搜索结果。type=user 时 ID 是用户 id；
// 其它值（group/discussion…）时 ID 其实是会话 id，不该拿去创建单聊。
type SearchUserHit struct {
	ID       ID     `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
	Status   string `json:"status"`
	IsMember bool   `json:"isMember"`
}

func (u SearchUserHit) Label() string {
	if u.Name != "" {
		return u.Name
	}
	return u.Username
}

// SearchMessageHit 是一条命中的聊天记录，外加它所属会话的展示信息。
// 消息本身用 Message 解析（含 created_at / is_read 归一化），
// conversation_meta 单独解一次，避免污染主消息模型。
type SearchMessageHit struct {
	Message  Message
	ConvID   string
	ConvName string
	ConvType string
}

// SearchHits 是一次搜索的三路结果。
type SearchHits struct {
	Conversations []SearchConvHit
	Messages      []SearchMessageHit
	Users         []SearchUserHit
	MessageTotal  int
}

func (h SearchHits) Empty() bool {
	return len(h.Conversations) == 0 && len(h.Messages) == 0 && len(h.Users) == 0
}

// SearchConversations 按群名或单聊对方昵称搜会话。
func (c *Client) SearchConversations(ctx context.Context, query string) ([]SearchConvHit, error) {
	path := "/api/v1/conversations/search?query=" + url.QueryEscape(query)
	var out []SearchConvHit
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SearchUsers 搜联系人/群。scope=users 时只返回用户，避免把群当成可发起单聊的对象。
func (c *Client) SearchUsers(ctx context.Context, query string) ([]SearchUserHit, error) {
	path := "/api/v1/users/search?q=" + url.QueryEscape(query) + "&scope=users"
	var out []SearchUserHit
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SearchMessages 搜聊天记录。注意后端的分页参数叫 pageSize（驼峰），
// 不是其它接口用的 page_size；传错会静默退回默认 20 条。
func (c *Client) SearchMessages(ctx context.Context, query, convID string, page, pageSize int) ([]SearchMessageHit, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	q := url.Values{}
	q.Set("keyword", query)
	q.Set("page", strconv.Itoa(page))
	q.Set("pageSize", strconv.Itoa(pageSize))
	if convID != "" {
		q.Set("conv_id", convID)
	}
	var raw struct {
		List  []json.RawMessage `json:"list"`
		Total int               `json:"total"`
	}
	if err := c.do(ctx, http.MethodGet, "/api/v1/messages/search?"+q.Encode(), nil, &raw); err != nil {
		return nil, 0, err
	}
	hits := make([]SearchMessageHit, 0, len(raw.List))
	for _, item := range raw.List {
		var m Message
		if err := json.Unmarshal(item, &m); err != nil {
			continue // 坏帧跳过，不让一条脏数据毁掉整个搜索
		}
		var meta struct {
			ConversationMeta struct {
				Name string `json:"name"`
				Type string `json:"type"`
			} `json:"conversation_meta"`
		}
		_ = json.Unmarshal(item, &meta)
		hits = append(hits, SearchMessageHit{
			Message:  m,
			ConvID:   m.ConversationID.String(),
			ConvName: meta.ConversationMeta.Name,
			ConvType: meta.ConversationMeta.Type,
		})
	}
	return hits, raw.Total, nil
}

// CreateSingleConversation 与某个用户开一段单聊。后端对已存在的单聊会复用，
// 返回体里 id / conversation_id 两种命名都出现过，所以都收。
func (c *Client) CreateSingleConversation(ctx context.Context, userID string) (string, error) {
	uid, err := strconv.ParseUint(userID, 10, 64)
	if err != nil || uid == 0 {
		return "", fmt.Errorf("无效的用户 ID %q", userID)
	}
	var out struct {
		ID             ID `json:"id"`
		ConversationID ID `json:"conversation_id"`
	}
	body := map[string]any{"type": "single", "recipient_id": uid}
	if err := c.do(ctx, http.MethodPost, "/api/v1/conversations", body, &out); err != nil {
		return "", err
	}
	if id := out.ID.String(); id != "" {
		return id, nil
	}
	if id := out.ConversationID.String(); id != "" {
		return id, nil
	}
	return "", nil // 有的版本只回 code:0，让调用方重拉列表
}
