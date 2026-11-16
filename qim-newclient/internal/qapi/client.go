package qapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Client 是 qim 服务端的 HTTP 客户端。
//
// 并发安全：Token 会被 HTTP 请求和 WS 重连同时读，refresh 还可能被多个
// 并发请求同时触发，所以 token 有读写锁、refresh 有单飞锁。
type Client struct {
	base string
	http *http.Client

	// Version / Platform 会作为 X-App-Version / X-App-Platform 上报。
	// 后端用它们做版本门槛校验（太旧的客户端不允许发消息），
	// 缺了这两个头可能出现「能登录但发不出消息」的怪现象。
	Version  string
	Platform string

	mu           sync.RWMutex
	token        string
	refreshToken string
	// onSession 在凭据变化时通知界面层刷新持久化（refresh 会轮换
	// refresh_token，只存登录那一刻的值会在下次启动时失效）。
	onSession func(token, refresh string)

	refreshMu sync.Mutex // 保证并发 401 只刷新一次
}

// New 创建一个客户端。base 形如 http://localhost:8080，
// 尾部斜杠会被去掉，避免拼出 //api/v1 这种路径。
// clientVersion 与 Electron 客户端保持一致：后端的发消息版本门槛按这个
// 字符串比较，报一个偏低的自造版本号会被挡在门外。
// platformOf 把 Go 的 GOOS 换成后端 NormalizePlatform 认识的 macos/linux/windows，
// 传 "desktop" 会被归一化成 unknown，平台专属门槛就会判错。
const clientVersion = "2.0.39"

func platformOf(goos string) string {
	switch goos {
	case "darwin":
		return "macos"
	case "windows":
		return "windows"
	case "linux":
		return "linux"
	default:
		return goos
	}
}

func New(base string) *Client {
	return &Client{
		base:     strings.TrimRight(base, "/"),
		http:     &http.Client{Timeout: 30 * time.Second},
		Version:  clientVersion,
		Platform: platformOf(runtime.GOOS),
	}
}

// BaseURL 返回服务端地址。
func (c *Client) BaseURL() string { return c.base }

// SetBaseURL 允许在运行时改服务器地址（对应网页侧的「服务器设置」）。
func (c *Client) SetBaseURL(base string) { c.base = strings.TrimRight(base, "/") }

// OnSessionChange 注册凭据变化回调。回调在锁外执行，避免它反过来
// 读 Session() 时死锁；同一个 Client 只需注册一次。
func (c *Client) OnSessionChange(fn func(token, refresh string)) {
	c.mu.Lock()
	c.onSession = fn
	c.mu.Unlock()
}

// SetSession 写入凭据。
func (c *Client) SetSession(token, refresh string) {
	c.mu.Lock()
	c.token, c.refreshToken = token, refresh
	fn := c.onSession
	c.mu.Unlock()
	if fn != nil {
		fn(token, refresh)
	}
}

// Session 读出当前凭据，WS 连接时要用 token。
func (c *Client) Session() (token, refresh string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.token, c.refreshToken
}

// HasToken 判断是否已经登录。
func (c *Client) HasToken() bool {
	t, _ := c.Session()
	return t != ""
}

// Login 登录并返回凭据。
//
// 二次验证（2FA）会返回 *APIError 且 IsTwoFactorRequired 为真，
// 调用方据此弹验证码输入，而不是当登录失败处理。
func (c *Client) Login(ctx context.Context, username, password string) (*Session, error) {
	body := map[string]any{
		"username": username,
		"password": password,
		"version":  c.Version,
	}
	var s Session
	if err := c.do(ctx, http.MethodPost, "/api/v1/auth/login", body, &s); err != nil {
		return nil, err
	}
	c.SetSession(s.Token, s.RefreshToken)
	return &s, nil
}

// Verify2FA 用二次验证会话码完成登录。
func (c *Client) Verify2FA(ctx context.Context, session, code string) (*Session, error) {
	body := map[string]any{"session": session, "code": code}
	var s Session
	if err := c.do(ctx, http.MethodPost, "/api/v1/auth/2fa/verify", body, &s); err != nil {
		return nil, err
	}
	c.SetSession(s.Token, s.RefreshToken)
	return &s, nil
}

// Conversations 拉取会话列表。
//
// 真实 qim-server 把数组包在 data.list 里（还带 total/page/has_more），
// 而早期接口与假服务端直接把数组放在 data。两种都收，避免换后端就全空。
func (c *Client) Conversations(ctx context.Context) ([]Conversation, error) {
	var raw json.RawMessage
	if err := c.do(ctx, http.MethodGet, "/api/v1/conversations", nil, &raw); err != nil {
		return nil, err
	}
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	var list []Conversation
	if err := json.Unmarshal(raw, &list); err == nil {
		return list, nil
	}
	var wrapped struct {
		List []Conversation `json:"list"`
	}
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return nil, fmt.Errorf("解析会话列表: %w", err)
	}
	return wrapped.List, nil
}

// Messages 拉取某个会话的历史消息。
// page 从 1 开始；pageSize 传 0 时用后端默认值。
func (c *Client) Messages(ctx context.Context, convID ID, page, pageSize int) (*MessagePage, error) {
	path := fmt.Sprintf("/api/v1/conversations/%s/messages", convID)
	if page > 0 || pageSize > 0 {
		q := make([]string, 0, 2)
		if page > 0 {
			q = append(q, fmt.Sprintf("page=%d", page))
		}
		if pageSize > 0 {
			q = append(q, fmt.Sprintf("page_size=%d", pageSize))
		}
		path += "?" + strings.Join(q, "&")
	}
	var p MessagePage
	if err := c.do(ctx, http.MethodGet, path, nil, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// MessagesBefore 按游标加载更早的一页历史消息。
//
// 不用 page+offset：聊天时新消息会不断插入，offset 会随总数漂移，
// 翻页容易重复或漏消息。后端 before_id 取「该 id 之前的最新 N 条」，
// 天然免疫插入。返回值仍是 MessagePage，pagination 用来判断有没有更早的。
func (c *Client) MessagesBefore(ctx context.Context, convID ID, beforeID string, pageSize int) (*MessagePage, error) {
	if pageSize <= 0 {
		pageSize = 30
	}
	path := fmt.Sprintf("/api/v1/conversations/%s/messages?before_id=%s&page_size=%d",
		convID, url.QueryEscape(beforeID), pageSize)
	var p MessagePage
	if err := c.do(ctx, http.MethodGet, path, nil, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// SendMessage 发送消息。
func (c *Client) SendMessage(ctx context.Context, convID ID, req SendRequest) (*Message, error) {
	var m Message
	path := fmt.Sprintf("/api/v1/conversations/%s/messages", convID)
	if err := c.do(ctx, http.MethodPost, path, req, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// MessagesAfter 增量拉取 afterID 之后的消息，用于断线补偿（WS 的 sync_hint）。
// 后端 after_id 是自增数字，返回结构与分页接口一致，都是 data.messages。
func (c *Client) MessagesAfter(ctx context.Context, convID ID, afterID string, pageSize int) (*MessagePage, error) {
	if pageSize <= 0 {
		pageSize = 50
	}
	path := fmt.Sprintf("/api/v1/conversations/%s/messages?after_id=%s&page_size=%d",
		convID, url.QueryEscape(afterID), pageSize)
	var p MessagePage
	if err := c.do(ctx, http.MethodGet, path, nil, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// MarkRead 把会话标记为已读。
func (c *Client) MarkRead(ctx context.Context, convID ID) error {
	path := fmt.Sprintf("/api/v1/conversations/%s/read", convID)
	return c.do(ctx, http.MethodPost, path, nil, nil)
}

// GetConversation 拉取单个会话详情。群聊会在 data 里带 members 与 announcement，
// 单聊/机器人会话不带 members。data 被 do 直接解进 *Conversation。
func (c *Client) GetConversation(ctx context.Context, convID ID) (*Conversation, error) {
	var conv Conversation
	path := fmt.Sprintf("/api/v1/conversations/%s", convID)
	if err := c.do(ctx, http.MethodGet, path, nil, &conv); err != nil {
		return nil, err
	}
	return &conv, nil
}

// SetConversationPin 置顶/取消置顶会话。后端用 is_pinned 布尔，body 必须带这个字段。
func (c *Client) SetConversationPin(ctx context.Context, convID ID, pinned bool) error {
	path := fmt.Sprintf("/api/v1/conversations/%s/pin", convID)
	return c.do(ctx, http.MethodPut, path, map[string]any{"is_pinned": pinned}, nil)
}

// SetConversationMute 设置/取消免打扰。后端用 muted 布尔。
func (c *Client) SetConversationMute(ctx context.Context, convID ID, muted bool) error {
	path := fmt.Sprintf("/api/v1/conversations/%s/mute", convID)
	return c.do(ctx, http.MethodPut, path, map[string]any{"muted": muted}, nil)
}

// DeleteConversation 把会话从「我的列表」里移除：DELETE /conversations/:id。
// 后端语义是 per-user 隐藏（HideConversation），不会解散群、不删历史；
// 想真正离开群聊要走 ExitGroup。
func (c *Client) DeleteConversation(ctx context.Context, convID ID) error {
	path := fmt.Sprintf("/api/v1/conversations/%s", convID)
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// ExitGroup 退出群聊：POST /groups/:id/exit，:id 是会话 ID（后端用会话 ID 反查群）。
// 群主不能直接退群，后端会返回「群主退出前请先转让群主或解散群聊」。
func (c *Client) ExitGroup(ctx context.Context, convID ID) error {
	path := fmt.Sprintf("/api/v1/groups/%s/exit", convID)
	return c.do(ctx, http.MethodPost, path, nil, nil)
}

// SetGroupAnnouncement 更新群/讨论组公告。对齐 qim-client：
// 端点是 PUT /api/v1/groups/:id/announcement，:id 是「会话 ID」
// （后端 handler 内部用会话 ID 反查群），body 只有 announcement 字符串。
// 权限由后端校验：仅群主/管理员可写。
func (c *Client) SetGroupAnnouncement(ctx context.Context, convID ID, text string) error {
	path := fmt.Sprintf("/api/v1/groups/%s/announcement", convID)
	return c.do(ctx, http.MethodPut, path, map[string]any{"announcement": text}, nil)
}

// SetGroupName 修改群/讨论组名称：PUT /api/v1/groups/:convID，body 只有
// name（后端 UpdateGroupInfo 其余字段留空表示不改）。:id 是会话 ID。
// 对齐 qim-client ChatHeaderActions 的「修改群名称」。
func (c *Client) SetGroupName(ctx context.Context, convID ID, name string) error {
	path := fmt.Sprintf("/api/v1/groups/%s", convID)
	return c.do(ctx, http.MethodPut, path, map[string]any{"name": name}, nil)
}

// CreateGroup 建群会话（POST /conversations type=group），创建者为群主。
// 返回新会话详情。
func (c *Client) CreateGroup(ctx context.Context, name string, memberIDs []uint) (*Conversation, error) {
	var conv Conversation
	err := c.do(ctx, http.MethodPost, "/api/v1/conversations",
		map[string]any{"type": "group", "name": name, "member_ids": memberIDs}, &conv)
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

// DissolveGroup 解散群（DELETE /groups/:convID），:id 是会话 ID，仅群主可操作。
func (c *Client) DissolveGroup(ctx context.Context, convID ID) error {
	path := fmt.Sprintf("/api/v1/groups/%s", convID)
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// MessageReadUsers 拉某条消息的已读用户列表（GET /messages/:id/read-users）。
// 对齐 qim-client：仅对「自己发的消息」有意义；read_users 不含发送者，
// total_members 是会话总人数，未读数 = total - read_count。
func (c *Client) MessageReadUsers(ctx context.Context, msgID ID) (*MessageReadInfo, error) {
	path := fmt.Sprintf("/api/v1/messages/%s/read-users", msgID)
	var info MessageReadInfo
	if err := c.do(ctx, http.MethodGet, path, nil, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// Notifications 拉通知列表（GET /notifications）。
// 注意分页参数名是 pageSize（camelCase）——handler 只认这个写法，
// API.md 里的 page_size 实际不生效。
func (c *Client) Notifications(ctx context.Context, page, pageSize int) (*NotificationPage, error) {
	path := "/api/v1/notifications"
	q := make([]string, 0, 2)
	if page > 0 {
		q = append(q, fmt.Sprintf("page=%d", page))
	}
	if pageSize > 0 {
		q = append(q, fmt.Sprintf("pageSize=%d", pageSize))
	}
	if len(q) > 0 {
		path += "?" + strings.Join(q, "&")
	}
	var p NotificationPage
	if err := c.do(ctx, http.MethodGet, path, nil, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// MarkNotificationRead 标记单条通知已读。
func (c *Client) MarkNotificationRead(ctx context.Context, id ID) error {
	path := fmt.Sprintf("/api/v1/notifications/%s/read", id)
	return c.do(ctx, http.MethodPut, path, nil, nil)
}

// MarkNotificationsAllRead 一键全部已读（PUT /notifications/read-all）。
func (c *Client) MarkNotificationsAllRead(ctx context.Context) error {
	return c.do(ctx, http.MethodPut, "/api/v1/notifications/read-all", nil, nil)
}

// ClearNotifications 清空所有通知（DELETE /notifications）。
func (c *Client) ClearNotifications(ctx context.Context) error {
	return c.do(ctx, http.MethodDelete, "/api/v1/notifications", nil, nil)
}

// DeleteNotification 删除单条通知。
func (c *Client) DeleteNotification(ctx context.Context, id ID) error {
	path := fmt.Sprintf("/api/v1/notifications/%s", id)
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// PatchNotification 统一更新通知（PATCH /notifications/:id）：
// read 标已读；pinned / important 传 true 触发 toggle（返回 toggled 布尔，
// 不含 notification）；action 走 accept|ignore|confirm|reschedule|confirm_create|todo_ignore，
// 成功后返回更新过的 notification（handled=true）。
func (c *Client) PatchNotification(ctx context.Context, id ID, req NotificationPatch) (*Notification, error) {
	path := fmt.Sprintf("/api/v1/notifications/%s", id)
	var out struct {
		Notification *Notification `json:"notification"`
	}
	if err := c.do(ctx, http.MethodPatch, path, req, &out); err != nil {
		return nil, err
	}
	return out.Notification, nil
}

// ApproveGroupJoin 同意入群申请（POST /groups/:convID/members）。
// qim-client 的 approve_reject 通知不走 PATCH action 分发，而是直接调这两个接口。
// 服务端 member_ids 是 []uint（数字数组），传字符串会 1001 参数错误。
func (c *Client) ApproveGroupJoin(ctx context.Context, convID ID, userID ID) error {
	return c.AddGroupMembers(ctx, convID.String(), []ID{userID})
}

// AddGroupMembers 邀请成员入群（POST /groups/:convID/members）。
// 服务端契约：member_ids 为 []uint 数字数组（handler/group_handler.go）。
func (c *Client) AddGroupMembers(ctx context.Context, convID string, ids []ID) error {
	nums := make([]uint, 0, len(ids))
	for _, id := range ids {
		if n, err := strconv.ParseUint(strings.TrimSpace(id.String()), 10, 64); err == nil && n > 0 {
			nums = append(nums, uint(n))
		}
	}
	path := fmt.Sprintf("/api/v1/groups/%s/members", convID)
	return c.do(ctx, http.MethodPost, path, map[string]any{"member_ids": nums}, nil)
}

// RejectGroupJoin 拒绝入群申请（DELETE /groups/:convID/join-requests/:userID）。
func (c *Client) RejectGroupJoin(ctx context.Context, convID ID, userID ID) error {
	path := fmt.Sprintf("/api/v1/groups/%s/join-requests/%s", convID, userID)
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// ApplyGroupJoin 申请加入群聊（POST /groups/:convID/apply）。
// 后端会给群主/管理员发 approve_reject 类型的通知。
func (c *Client) ApplyGroupJoin(ctx context.Context, convID ID) error {
	path := fmt.Sprintf("/api/v1/groups/%s/apply", convID)
	return c.do(ctx, http.MethodPost, path, nil, nil)
}

// Logout 通知服务端本次登出，让其它设备同步下线。
// token 由调用方显式传入：logout 会立刻清掉本地凭据，用快照才能保证
// 请求发出时 Authorization 头还在。登出是尽力而为，不做拆包或 token 刷新。
func (c *Client) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/api/v1/auth/logout", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-App-Version", c.Version)
	req.Header.Set("X-App-Platform", c.Platform)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("登出失败 HTTP %d", resp.StatusCode)
	}
	return nil
}

// do 发出一次请求并把 {code,message,data} 拆开。
//
// 401 时会尝试刷新 token 并重放一次——这是长驻客户端必须有的，
// 否则 token 一过期整个界面就卡死，只能让用户重新登录。
func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	err := c.doOnce(ctx, method, path, body, out)
	// 认证类接口不参与刷新重放：登录失败（含 1002 二次验证）不能拿
	// 上一个账号的旧 token 重试，否则可能把旧会话带进新登录流程。
	if err == nil || strings.HasPrefix(path, "/api/v1/auth/") {
		return err
	}
	if !isUnauthorized(err) {
		return err
	}

	if rerr := c.refresh(ctx); rerr != nil {
		return err // 刷新失败就返回原始 401，让上层跳到登录
	}
	return c.doOnce(ctx, method, path, body, out)
}

// isUnauthorized 判断失败是不是「token 失效」。
// 后端业务码：1001 = 参数错误（登录密码错也用它），1002 = 未授权；
// 1002 同时被登录接口用来表示「需要二次验证」，靠 detail 区分。
// 之前把 1001 当未授权，导致密码错时也走一遍刷新判断；真正的 1002
// 反而漏掉，token 过期不会自动续期。
func isUnauthorized(err error) bool {
	e, ok := err.(*APIError)
	if !ok || e.IsTwoFactorRequired() {
		return false
	}
	return e.Code == 401 || e.Code == 1002
}

func (c *Client) doOnce(ctx context.Context, method, path string, body, out any) error {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("序列化请求体: %w", err)
		}
		rdr = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.base+path, rdr)
	if err != nil {
		return fmt.Errorf("构造请求: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-App-Version", c.Version)
	req.Header.Set("X-App-Platform", c.Platform)
	if t, _ := c.Session(); t != "" {
		req.Header.Set("Authorization", "Bearer "+t)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("请求 %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应: %w", err)
	}

	// 非 2xx 也先按业务错误解析，后端很多错误是 200 + code!=0，
	// 但网关层可能是 4xx/5xx 且带同样的结构。
	if len(raw) == 0 {
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		return &APIError{Code: resp.StatusCode, Message: resp.Status}
	}

	var env struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Detail  json.RawMessage `json:"detail"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			// 不是标准包装，直接尝试按目标类型解（例如直接返回数组）
			if out != nil {
				return json.Unmarshal(raw, out)
			}
			return nil
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(string(raw), 200))
	}

	if env.Code != 0 {
		return &APIError{Code: env.Code, Message: env.Message, Detail: env.Detail}
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return &APIError{Code: 401, Message: "未授权"}
	}
	if out == nil || len(env.Data) == 0 || string(env.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(env.Data, out); err != nil {
		return fmt.Errorf("解析 data: %w", err)
	}
	return nil
}

// refresh 用 refresh_token 换新 token。
// 单飞：并发 401 只发一次刷新请求，其余等结果复用。
func (c *Client) refresh(ctx context.Context) error {
	c.refreshMu.Lock()
	defer c.refreshMu.Unlock()

	_, rt := c.Session()
	if rt == "" {
		return fmt.Errorf("没有 refresh token")
	}

	body, _ := json.Marshal(map[string]string{"refresh_token": rt})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/api/v1/auth/refresh", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+rt)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("刷新 token: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var env struct {
		Code int `json:"code"`
		Data struct {
			Token        string `json:"token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("解析刷新响应: %w", err)
	}
	if env.Code != 0 || env.Data.Token == "" {
		return fmt.Errorf("刷新失败 code=%d", env.Code)
	}
	next := env.Data.RefreshToken
	if next == "" {
		next = rt
	}
	c.SetSession(env.Data.Token, next)
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// ---------- 频道 ----------

// ListChannels 拉频道列表（GET /channels）。返回含 is_subscribed/message_count/
// last_message 扩展字段；默认频道已由后端懒订阅兜底。
func (c *Client) ListChannels(ctx context.Context) ([]Channel, error) {
	var list []Channel
	if err := c.do(ctx, http.MethodGet, "/api/v1/channels", nil, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// CreateChannel 创建频道（POST /channels）。审批开启时 status=pending。
func (c *Client) CreateChannel(ctx context.Context, req ChannelCreateRequest) (*ChannelCreated, error) {
	var out ChannelCreated
	if err := c.do(ctx, http.MethodPost, "/api/v1/channels", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// SubscribeChannel 订阅频道（POST /channels/:id/subscribe）。
func (c *Client) SubscribeChannel(ctx context.Context, channelID ID) error {
	path := fmt.Sprintf("/api/v1/channels/%s/subscribe", channelID)
	return c.do(ctx, http.MethodPost, path, nil, nil)
}

// UnsubscribeChannel 退订频道（DELETE /channels/:id/subscribe）。
// 默认频道不可退订（后端拒绝），错误原样返回。
func (c *Client) UnsubscribeChannel(ctx context.Context, channelID ID) error {
	path := fmt.Sprintf("/api/v1/channels/%s/subscribe", channelID)
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// ChannelMessages 拉频道全部消息（GET /channels/:id/messages）。
// 后端按 created_at DESC 返回，调用方需反转成 ASC 再渲染。
func (c *Client) ChannelMessages(ctx context.Context, channelID ID) ([]ChannelMessage, error) {
	path := fmt.Sprintf("/api/v1/channels/%s/messages", channelID)
	var list []ChannelMessage
	if err := c.do(ctx, http.MethodGet, path, nil, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// SendChannelMessage 在频道发布消息（POST /channels/:id/messages）。
// 权限不足/频道审批中由后端拒绝（403），错误原样返回给上层提示。
func (c *Client) SendChannelMessage(ctx context.Context, channelID ID, content, typ string) (*ChannelMessage, error) {
	path := fmt.Sprintf("/api/v1/channels/%s/messages", channelID)
	if typ == "" {
		typ = "text"
	}
	var msg ChannelMessage
	if err := c.do(ctx, http.MethodPost, path, map[string]any{"content": content, "type": typ}, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

// LikeChannelMessage 点赞（POST /channels/messages/:messageId/like）。重复点赞后端幂等。
func (c *Client) LikeChannelMessage(ctx context.Context, messageID ID) (*ChannelLikeInfo, error) {
	path := fmt.Sprintf("/api/v1/channels/messages/%s/like", messageID)
	var info ChannelLikeInfo
	if err := c.do(ctx, http.MethodPost, path, nil, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// UnlikeChannelMessage 取消点赞（DELETE /channels/messages/:messageId/like）。
func (c *Client) UnlikeChannelMessage(ctx context.Context, messageID ID) (*ChannelLikeInfo, error) {
	path := fmt.Sprintf("/api/v1/channels/messages/%s/like", messageID)
	var info ChannelLikeInfo
	if err := c.do(ctx, http.MethodDelete, path, nil, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// ChannelMessageLikes 查询点赞状态（GET /channels/messages/:messageId/likes）。
func (c *Client) ChannelMessageLikes(ctx context.Context, messageID ID) (*ChannelLikeInfo, error) {
	path := fmt.Sprintf("/api/v1/channels/messages/%s/likes", messageID)
	var info ChannelLikeInfo
	if err := c.do(ctx, http.MethodGet, path, nil, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// CommentChannelMessage 发评论（POST /channels/messages/:messageId/comments）。
func (c *Client) CommentChannelMessage(ctx context.Context, messageID ID, content string) (*ChannelComment, error) {
	path := fmt.Sprintf("/api/v1/channels/messages/%s/comments", messageID)
	var cm ChannelComment
	if err := c.do(ctx, http.MethodPost, path, map[string]any{"content": content}, &cm); err != nil {
		return nil, err
	}
	return &cm, nil
}

// ChannelMessageComments 拉评论列表（GET /channels/messages/:messageId/comments，ASC）。
func (c *Client) ChannelMessageComments(ctx context.Context, messageID ID) ([]ChannelComment, error) {
	path := fmt.Sprintf("/api/v1/channels/messages/%s/comments", messageID)
	var list []ChannelComment
	if err := c.do(ctx, http.MethodGet, path, nil, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// UpdateMe 更新当前用户资料（PUT /users/me），返回更新后的 User。
func (c *Client) UpdateMe(ctx context.Context, req UpdateMeRequest) (*User, error) {
	var u User
	if err := c.do(ctx, http.MethodPut, "/api/v1/users/me", req, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUser 拉单个用户详情（GET /users/:id）。
func (c *Client) GetUser(ctx context.Context, userID ID) (*User, error) {
	var u User
	if err := c.do(ctx, http.MethodGet, "/api/v1/users/"+userID.String(), nil, &u); err != nil {
		return nil, err
	}
	return &u, nil
}

// OrgTree 拉组织架构树（GET /organization/tree），返回顶层部门数组。
// 无部门的用户会挂在一个名为「非标准用户」的虚拟部门下。
func (c *Client) OrgTree(ctx context.Context) ([]Department, error) {
	var wrap struct {
		Departments []Department `json:"departments"`
	}
	if err := c.do(ctx, http.MethodGet, "/api/v1/organization/tree", nil, &wrap); err != nil {
		return nil, err
	}
	return wrap.Departments, nil
}

// UsersStatus 批量查用户连接状态（GET /users/status?user_ids=1,2,3）。
func (c *Client) UsersStatus(ctx context.Context, userIDs []uint) ([]UserStatusItem, error) {
	if len(userIDs) == 0 {
		return nil, nil
	}
	parts := make([]string, len(userIDs))
	for i, id := range userIDs {
		parts[i] = strconv.FormatUint(uint64(id), 10)
	}
	path := "/api/v1/users/status?user_ids=" + strings.Join(parts, ",")
	var list []UserStatusItem
	if err := c.do(ctx, http.MethodGet, path, nil, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// UserSettingValue 通用用户设置读取响应（GET /user-settings/:key）。
// value 是任意 JSON 值，返回原始字节由调用方按 key 的约定解析。
type UserSettingValue struct {
	Value    json.RawMessage `json:"value"`
	HasValue bool            `json:"has_value"`
}

// GetUserSetting 读一项用户设置（GET /user-settings/:key）。
func (c *Client) GetUserSetting(ctx context.Context, key string) (*UserSettingValue, error) {
	var v UserSettingValue
	if err := c.do(ctx, http.MethodGet, "/api/v1/user-settings/"+key, nil, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// PutUserSetting 保存一项用户设置（PUT /user-settings/:key，value 任意 JSON 值整存整取）。
func (c *Client) PutUserSetting(ctx context.Context, key string, value any) error {
	return c.do(ctx, http.MethodPut, "/api/v1/user-settings/"+key, map[string]any{"value": value}, nil)
}

// Me 拉当前用户完整资料（GET /users/me）。
func (c *Client) Me(ctx context.Context) (*User, error) {
	var u User
	if err := c.do(ctx, http.MethodGet, "/api/v1/users/me", nil, &u); err != nil {
		return nil, err
	}
	return &u, nil
}
