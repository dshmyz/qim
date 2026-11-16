package qapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// pingInterval 是应用层心跳间隔。后端回 {"type":"pong"}，
	// 不是 WebSocket 控制帧，所以只能用 Application 层判断。
	pingInterval = 30 * time.Second
	// readWait 是多久收不到任何帧就判定连接已死。
	// 必须比 pingInterval 大，否则正常心跳期间就会被误杀。
	readWait = 90 * time.Second

	minReconnectDelay = time.Second
	maxReconnectDelay = 30 * time.Second
	// 连接存活超过这个时长再断开，认为链路本来是稳定的，退避归零重来；
	// 否则按指数退避，避免服务端没起来时每秒重连刷屏。
	stableConnection = 30 * time.Second
)

// nextBackoff 返回下一次重连的等待时长，封顶 maxReconnectDelay。
// 单独抽成纯函数，退避曲线可以直接单测，不用起 WS 服务。
func nextBackoff(cur time.Duration) time.Duration {
	if cur < minReconnectDelay {
		return minReconnectDelay
	}
	if cur >= maxReconnectDelay/2 {
		return maxReconnectDelay
	}
	return cur * 2
}

// Event 是服务端推来的一条消息。
//
// Data 保持原始 JSON：事件类型很多（新消息、已读回执、会话更新、
// 成员变动……），在 Go 侧全部建模既没必要也容易过时。上层按需解。
type Event struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// Raw 把 Data 解成具体类型。
func (e Event) Raw(v any) error { return json.Unmarshal(e.Data, v) }

// Status 是连接状态，UI 用它显示「连接中 / 已连接 / 断开」。
type Status int

const (
	StatusDisconnected Status = iota
	StatusConnecting
	StatusConnected
)

func (s Status) String() string {
	switch s {
	case StatusConnecting:
		return "连接中"
	case StatusConnected:
		return "已连接"
	}
	return "已断开"
}

// Realtime 维护与服务端的 WebSocket 长连接。
//
// 职责边界刻意收窄：只负责「连上、认证、保活、断线重连、把事件交出去」。
// 发消息走 REST（和网页侧一致），因为 HTTP 有明确返回值和错误码，
// 比在 WS 上等 ack 好排查。
type Realtime struct {
	client *Client

	mu     sync.Mutex
	conn   *websocket.Conn
	status Status

	handlers    map[string][]func(Event)
	anyHandlers []func(Event)
	onStatus    []func(Status)

	stopOnce sync.Once
	stop     chan struct{}
	running  bool
}

// NewRealtime 创建实时连接管理器。
func NewRealtime(c *Client) *Realtime {
	return &Realtime{
		client:   c,
		status:   StatusDisconnected,
		handlers: make(map[string][]func(Event)),
		stop:     make(chan struct{}),
	}
}

// On 注册某类事件的处理函数。type 用后端的事件名，如 "new_message"。
func (r *Realtime) On(typ string, fn func(Event)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[typ] = append(r.handlers[typ], fn)
}

// OnAny 注册兜底处理函数，收到所有事件都会调一次（用于日志和调试）。
func (r *Realtime) OnAny(fn func(Event)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.anyHandlers = append(r.anyHandlers, fn)
}

// OnStatus 注册连接状态变化回调。
func (r *Realtime) OnStatus(fn func(Status)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onStatus = append(r.onStatus, fn)
}

// Status 返回当前连接状态。
func (r *Realtime) Status() Status {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.status
}

func (r *Realtime) setStatus(s Status) {
	r.mu.Lock()
	if r.status == s {
		r.mu.Unlock()
		return
	}
	r.status = s
	subs := append([]func(Status){}, r.onStatus...)
	r.mu.Unlock()
	for _, fn := range subs {
		fn(s)
	}
}

// Start 在后台维持连接，直到 Stop 或 ctx 取消。
// 返回后不会阻塞调用方。
func (r *Realtime) Start(ctx context.Context) {
	r.mu.Lock()
	if r.running {
		r.mu.Unlock()
		return
	}
	r.running = true
	r.mu.Unlock()

	go r.loop(ctx)
}

func (r *Realtime) loop(ctx context.Context) {
	backoff := minReconnectDelay

	for {
		select {
		case <-ctx.Done():
			r.shutdown()
			return
		case <-r.stop:
			r.shutdown()
			return
		default:
		}

		started := time.Now()
		if err := r.connectOnce(ctx); err != nil {
			log.Printf("[ws] %v", err)
		}
		// 之前的实现在这里把 backoff 在 1s/2s 之间来回切换，
		// maxBackoff 永远到不了；改成「连接本来就稳定才归零」。
		if time.Since(started) >= stableConnection {
			backoff = minReconnectDelay
		}

		select {
		case <-ctx.Done():
			r.shutdown()
			return
		case <-r.stop:
			r.shutdown()
			return
		case <-time.After(backoff):
		}

		backoff = nextBackoff(backoff)
	}
}

// connectOnce 建立一次连接，阻塞直到连接断开。
func (r *Realtime) connectOnce(ctx context.Context) error {
	token, _ := r.client.Session()
	if token == "" {
		return fmt.Errorf("未登录，不建立长连接")
	}

	r.setStatus(StatusConnecting)
	dialer := websocket.Dialer{
		HandshakeTimeout: 15 * time.Second,
		Proxy:            http.ProxyFromEnvironment,
	}
	conn, resp, err := dialer.DialContext(ctx, r.wsURL(), nil)
	if err != nil {
		if resp != nil {
			return fmt.Errorf("连接失败 HTTP %d: %w", resp.StatusCode, err)
		}
		return fmt.Errorf("连接失败: %w", err)
	}

	r.mu.Lock()
	r.conn = conn
	r.mu.Unlock()
	defer func() {
		r.mu.Lock()
		r.conn = nil
		r.mu.Unlock()
		_ = conn.Close()
		r.setStatus(StatusDisconnected)
	}()

	// 首帧认证：连接建立后立刻发 token，服务端校验通过才回 auth_success。
	// 协议如此，token 不走 URL（避免被写进访问日志）。
	if err := conn.WriteJSON(map[string]any{
		"type": "auth",
		"data": map[string]string{"token": token},
	}); err != nil {
		return fmt.Errorf("发送认证帧: %w", err)
	}

	// 心跳独立 goroutine，读循环不能自己做定时器（会阻塞收消息）
	hbDone := make(chan struct{})
	go func() {
		t := time.NewTicker(pingInterval)
		defer t.Stop()
		for {
			select {
			case <-hbDone:
				return
			case <-t.C:
				r.mu.Lock()
				c := r.conn
				r.mu.Unlock()
				if c == nil {
					return
				}
				_ = c.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := c.WriteJSON(map[string]any{"type": "ping"}); err != nil {
					return
				}
			}
		}
	}()
	defer close(hbDone)

	return r.readLoop(conn)
}

func (r *Realtime) readLoop(conn *websocket.Conn) error {
	var authErr error

	for {
		_ = conn.SetReadDeadline(time.Now().Add(readWait))
		_, raw, err := conn.ReadMessage()
		if err != nil {
			if authErr != nil {
				return authErr
			}
			return fmt.Errorf("连接断开: %w", err)
		}

		var ev Event
		if err := json.Unmarshal(raw, &ev); err != nil {
			log.Printf("[ws] 无法解析的帧: %s", truncate(string(raw), 120))
			continue
		}

		switch ev.Type {
		case "pong":
			continue
		case "auth_success":
			r.setStatus(StatusConnected)
			log.Printf("[ws] 认证成功")
		case "auth_error":
			// token 失效：刷新后由外层重连；失败就是真的需要重新登录
			log.Printf("[ws] 认证失败，尝试刷新 token")
			if rerr := r.client.refresh(context.Background()); rerr != nil {
				authErr = fmt.Errorf("认证失败且刷新无果，需要重新登录: %w", rerr)
				return authErr
			}
			return fmt.Errorf("token 已刷新，重连")
		}

		r.dispatch(ev)
	}
}

func (r *Realtime) dispatch(ev Event) {
	r.mu.Lock()
	subs := append([]func(Event){}, r.handlers[ev.Type]...)
	any := append([]func(Event){}, r.anyHandlers...)
	r.mu.Unlock()

	for _, fn := range any {
		func() {
			defer func() {
				if p := recover(); p != nil {
					log.Printf("[ws] 处理 %s 时 panic: %v", ev.Type, p)
				}
			}()
			fn(ev)
		}()
	}
	for _, fn := range subs {
		func() {
			defer func() {
				if p := recover(); p != nil {
					log.Printf("[ws] 处理 %s 时 panic: %v", ev.Type, p)
				}
			}()
			fn(ev)
		}()
	}
}

// Send 主动向服务端发一帧（已读回执之类）。
func (r *Realtime) Send(typ string, data any) error {
	r.mu.Lock()
	conn := r.conn
	r.mu.Unlock()
	if conn == nil {
		return fmt.Errorf("长连接未建立")
	}
	_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return conn.WriteJSON(map[string]any{"type": typ, "data": data})
}

// Stop 关闭连接并停止重连。
func (r *Realtime) Stop() {
	r.stopOnce.Do(func() { close(r.stop) })
}

func (r *Realtime) shutdown() {
	r.mu.Lock()
	c := r.conn
	r.running = false
	r.mu.Unlock()
	if c != nil {
		_ = c.Close()
	}
}

// wsURL 把 http(s) 服务端地址换算成 ws(s) 的 /api/v1/ws 地址。
func (r *Realtime) wsURL() string {
	base := r.client.BaseURL()
	u, err := url.Parse(base)
	if err != nil {
		// 地址异常时给个能跑的兜底，让报错出现在连接阶段而不是这里
		return strings.Replace(base, "http", "ws", 1) + "/api/v1/ws"
	}
	if u.Scheme == "https" {
		u.Scheme = "wss"
	} else {
		u.Scheme = "ws"
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/api/v1/ws"
	q := u.Query()
	q.Set("version", r.client.Version)
	q.Set("platform", r.client.Platform)
	u.RawQuery = q.Encode()
	return u.String()
}
