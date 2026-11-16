package qapi

// 单条消息的操作。三个端点都在 /api/v1/messages/:id 下，注意撤回和提醒
// 是 POST（不是 DELETE），删除才是 DELETE：和网页侧 api/message.ts 一致。

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// RecallMessage 撤回自己发的消息。后端有 120s 时限和「只能撤回自己消息」
// 的校验，超时/越权会返回业务错误，由界面层显示原因。
// 成功时返回撤回后的完整消息：content 已换成占位，原内容在
// Extra.original_content（供「撤回后重新编辑」回填输入框）。
func (c *Client) RecallMessage(ctx context.Context, messageID string) (*Message, error) {
	if messageID == "" {
		return nil, fmt.Errorf("消息 ID 为空")
	}
	path := "/api/v1/messages/" + strings.TrimSpace(messageID) + "/recall"
	// 后端这个端点的 data 是 {"message":"消息撤回成功","data":msg} 双层包裹，
	// 这里剥掉外层拿到真正的消息对象。
	var inner struct {
		Message string   `json:"message"`
		Data    *Message `json:"data"`
	}
	if err := c.do(ctx, http.MethodPost, path, nil, &inner); err != nil {
		return nil, err
	}
	if inner.Data == nil {
		return nil, fmt.Errorf("撤回响应缺少消息数据")
	}
	return inner.Data, nil
}

// DeleteMessage 从自己的视图里删除一条消息（后端按用户做软删）。
func (c *Client) DeleteMessage(ctx context.Context, messageID string) error {
	if messageID == "" {
		return fmt.Errorf("消息 ID 为空")
	}
	path := "/api/v1/messages/" + strings.TrimSpace(messageID)
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// RemindMessage 给单聊里对方还没读的消息发一次提醒。
// 后端会校验「发送者本人 + 单聊 + 消息未读」，界面层提前把这些条件
// 过滤掉只是为了不显示点了必然失败的菜单项。
func (c *Client) RemindMessage(ctx context.Context, messageID string) error {
	if messageID == "" {
		return fmt.Errorf("消息 ID 为空")
	}
	path := "/api/v1/messages/" + strings.TrimSpace(messageID) + "/remind"
	return c.do(ctx, http.MethodPost, path, nil, nil)
}
