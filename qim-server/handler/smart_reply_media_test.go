package handler

import (
	"testing"

	"github.com/dshmyz/qim/qim-server/model"
)

func TestDecideDirectMediaReply_FileNeverTriggers(t *testing.T) {
	autoCfg := &model.GroupAIConfig{Enabled: true, ReplyMode: "auto"}
	if decideDirectMediaReply(autoCfg, `{"url":"x"}`, false, "file") {
		t.Error("auto 模式直发文件不得触发 AI 接话")
	}
	if !decideDirectMediaReply(autoCfg, `{"url":"x"}`, false, "image") {
		t.Error("auto 模式直发图片应保留触发（多模态有语义）")
	}
	// 其他门仍生效
	if decideDirectMediaReply(&model.GroupAIConfig{Enabled: false}, "{}", false, "image") {
		t.Error("未启用群不得触发")
	}
	if decideDirectMediaReply(autoCfg, "{}", true, "image") {
		t.Error("防刷屏拦截时不得触发")
	}
	if decideDirectMediaReply(&model.GroupAIConfig{Enabled: true, ReplyMode: "mention_only"}, "{}", false, "image") {
		t.Error("mention_only 模式不得触发")
	}
}
