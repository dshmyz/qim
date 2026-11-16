package client

// 登录态持久化：token / refresh_token 由系统钥匙串加密保存，不明文落盘。
// 界面层只在几个明确时机写；凭据在 refresh 时被轮换，所以 qapi.Client
// 的 OnSessionChange 会把新值同步进来，避免重启后拿到过期的 refresh_token。

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/egoist/mygo"
)

const sessionSecretKey = "qim.session"

// savedSession 是重启免登录需要的全部信息。
// 不存密码：只存可撤销的 token，用户随时可以退出登录清掉。
type savedSession struct {
	ServerURL    string `json:"server"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh"`
	UserID       string `json:"user_id"`
	UserName     string `json:"user_name"`
}

func (s savedSession) valid() bool {
	return s.ServerURL != "" && s.Token != ""
}

// sessionStore 抽象掉系统凭据存储，headless 测试注入内存实现。
// 没有运行中的 app 时钥匙串不可用，测试不该依赖它。
type sessionStore interface {
	Load() (savedSession, error)
	Save(savedSession) error
	Clear() error
}

// secretStore 用 mygo.Secrets：macOS 钥匙串 / Windows 凭据管理器 / libsecret。
type secretStore struct{}

func (secretStore) Load() (savedSession, error) {
	var s savedSession
	b, err := mygo.Secrets.Get(sessionSecretKey)
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return savedSession{}, err
	}
	return s, nil
}

func (secretStore) Save(s savedSession) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return mygo.Secrets.Set(sessionSecretKey, b)
}

func (secretStore) Clear() error {
	return mygo.Secrets.Delete(sessionSecretKey)
}

// memStore 只给测试用。
type memStore struct {
	mu   sync.Mutex
	val  savedSession
	set  bool
	size int // Save 次数，测试用来验证「refresh 后有没有回写」
}

func (m *memStore) Load() (savedSession, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !m.set {
		return savedSession{}, mygo.ErrSecretNotFound
	}
	return m.val, nil
}

func (m *memStore) Save(s savedSession) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.val, m.set = s, true
	m.size++
	return nil
}

func (m *memStore) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.val, m.set = savedSession{}, false
	m.size++
	return nil
}

func (m *memStore) current() (savedSession, bool, int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.val, m.set, m.size
}

// onSessionChange 由 qapi.Client 在 SetSession 时回调。
// 退出登录会写空 token，这里直接跳过——清理由 forgetSession 负责，
// 否则会把刚删掉的凭据再写回去。
func (a *app) onSessionChange(token, refresh string) {
	if token == "" || a.store == nil {
		return
	}
	a.savedMu.Lock()
	a.saved.Token, a.saved.RefreshToken = token, refresh
	s := a.saved
	a.savedMu.Unlock()
	if !s.valid() {
		return
	}
	if err := a.store.Save(s); err != nil {
		log.Printf("[app] 保存登录态失败: %v", err)
	}
}

// rememberSession 在登录成功后写入完整信息（服务器地址、用户资料）。
func (a *app) rememberSession(server, userID, userName string) {
	if a.store == nil {
		return
	}
	token, refresh := a.client.Session()
	a.savedMu.Lock()
	a.saved = savedSession{
		ServerURL: server, Token: token, RefreshToken: refresh,
		UserID: userID, UserName: userName,
	}
	s := a.saved
	a.savedMu.Unlock()
	if err := a.store.Save(s); err != nil {
		log.Printf("[app] 保存登录态失败: %v", err)
	}
}

// forgetSession 清掉本地凭据（用户主动退出，或启动时发现已失效）。
func (a *app) forgetSession() {
	a.savedMu.Lock()
	a.saved = savedSession{}
	a.savedMu.Unlock()
	if a.store == nil {
		return
	}
	if err := a.store.Clear(); err != nil {
		log.Printf("[app] 清除登录态失败: %v", err)
	}
}
