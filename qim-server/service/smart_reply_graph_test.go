package service

import (
	"strings"
	"testing"

	"github.com/cloudwego/eino/schema"
)

// TestBuildContextBlocks_SuccessOnly 仅在 Quoted.Kind=QuotedFile 非空时，产出「已读到」应答，且无「未能读取」应答。
func TestBuildContextBlocks_SuccessOnly(t *testing.T) {
	blocks := buildContextBlocks(&SmartReplyContext{
		Quoted: &QuotedContext{Kind: QuotedFile, Name: "a.txt", Text: "📄 被引用文件「a.txt」的内容：\nhello world"},
	})

	roles, contents := flattenRoles(blocks)
	if !anyContains(contents, "我已读到被引用的文件内容") {
		t.Fatalf("成功场景应包含「已读到」应答，got contents=%v", contents)
	}
	if anyContains(contents, "未能读取") {
		t.Fatalf("成功场景不应出现「未能读取」应答，got contents=%v", contents)
	}
	if !anyContains(contents, "hello world") {
		t.Fatalf("成功场景应注入文件正文，got contents=%v", contents)
	}
	if anyContains(roles, string(schema.System)) {
		t.Fatalf("buildContextBlocks 不应产 System 消息（System 由外层构建），got roles=%v", roles)
	}
}

// TestBuildContextBlocks_FailureOnly 仅在 Quoted.Kind=QuotedFailed 非空时，产出「未能读取」应答，且无「已读到」应答。
func TestBuildContextBlocks_FailureOnly(t *testing.T) {
	blocks := buildContextBlocks(&SmartReplyContext{
		Quoted: &QuotedContext{Kind: QuotedFailed, Name: "b.pdf", Text: "📄 你引用了一条文件消息「b.pdf」，但其体积过大（超过 20MB），无法一次性读入上下文。"},
	})

	roles, contents := flattenRoles(blocks)
	if !anyContains(contents, "未能读取该文件") {
		t.Fatalf("失败场景应包含「未能读取」应答，got contents=%v", contents)
	}
	if anyContains(contents, "我已读到") {
		t.Fatalf("失败场景不应出现「已读到」应答，got contents=%v", contents)
	}
	if !anyContains(contents, "体积过大") {
		t.Fatalf("失败场景应注入失败说明，got contents=%v", contents)
	}
	_ = roles
}

// TestBuildContextBlocks_None 两者皆空时，不产出任何被引用文件相关的上下文消息块。
func TestBuildContextBlocks_None(t *testing.T) {
	blocks := buildContextBlocks(&SmartReplyContext{})

	if len(blocks) != 0 {
		t.Fatalf("无被引用对象时不应有任何上下文块，got %d blocks", len(blocks))
	}
}

// TestBuildContextBlocks_KnowledgeMemory 知识库/记忆上下文不受被引用对象拆分影响，各自正常产出。
func TestBuildContextBlocks_KnowledgeMemory(t *testing.T) {
	blocks := buildContextBlocks(&SmartReplyContext{
		KnowledgeCtx: "KB内容",
		MemoryCtx:    "记忆内容",
		Quoted:       &QuotedContext{Kind: QuotedFile, Name: "a.txt", Text: "📄 文件正文"},
	})

	roles, contents := flattenRoles(blocks)
	if !anyContains(contents, "优先参考这些内容") {
		t.Fatalf("应产出知识库应答，got %v", contents)
	}
	if !anyContains(contents, "我记住了这些历史信息") {
		t.Fatalf("应产出记忆应答，got %v", contents)
	}
	if anyContains(roles, string(schema.System)) {
		t.Fatalf("不应产出 System 消息，got %v", roles)
	}
}

func flattenRoles(blocks []*schema.Message) ([]string, []string) {
	var roles, contents []string
	for _, b := range blocks {
		roles = append(roles, string(b.Role))
		contents = append(contents, b.Content)
	}
	return roles, contents
}

func anyContains(ss []string, sub string) bool {
	for _, s := range ss {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}
