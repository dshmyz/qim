package qapi

import "encoding/json"

// 三类结构化消息的解析：card（bot 交互卡片）、news（资讯卡）、
// miniApp（小程序分享卡）。对齐 qim-client 的 CardMessage / NewsMessage /
// MiniAppMessage 的 payload。按钮点击、小程序打开属于 bot/应用侧能力，
// 客户端这里只做可读展示：标题 + 摘要，news 保留可点链接。

type CardInfo struct {
	Title   string `json:"title"`
	Text    string `json:"text"`
	Buttons []struct {
		ID    string `json:"id"`
		Text  string `json:"text"`
		Style string `json:"style"`
	} `json:"buttons"`
	// 已点击的按钮 id（服务端派生，跨设备一致）；空表示未提交过。
	ActionID string `json:"card_action_id"`
}

func ParseCardInfo(content string) *CardInfo {
	if content == "" {
		return nil
	}
	var ci CardInfo
	if err := json.Unmarshal([]byte(content), &ci); err != nil {
		return nil
	}
	if ci.Title == "" && ci.Text == "" {
		return nil
	}
	return &ci
}

type NewsInfo struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	URL     string `json:"url"`
	Image   string `json:"image"`
}

func ParseNewsInfo(content string) *NewsInfo {
	if content == "" {
		return nil
	}
	var ni NewsInfo
	if err := json.Unmarshal([]byte(content), &ni); err != nil {
		return nil
	}
	if ni.Title == "" && ni.Summary == "" && ni.URL == "" {
		return nil
	}
	return &ni
}

type MiniAppInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Icon string `json:"icon"`
	Desc string `json:"description"`
	Path string `json:"path"`
	// display_name / title 是网页端的兼容字段，优先级在 name 之后。
	DisplayName string `json:"display_name"`
	Title       string `json:"title"`
}

// Label 返回卡片主标题：display_name > name > title > 未命名。
func (mi *MiniAppInfo) Label() string {
	for _, s := range []string{mi.DisplayName, mi.Name, mi.Title} {
		if s != "" {
			return s
		}
	}
	return "未命名小程序"
}

func ParseMiniAppInfo(content string) *MiniAppInfo {
	if content == "" {
		return nil
	}
	var mi MiniAppInfo
	if err := json.Unmarshal([]byte(content), &mi); err != nil {
		return nil
	}
	if mi.Label() == "未命名小程序" && mi.Desc == "" && mi.ID == "" {
		return nil
	}
	return &mi
}
