package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dshmyz/qim/qim-server/ai"
	"github.com/dshmyz/qim/qim-server/pkg/aiprompt"
)

// DescribeImage 用视觉模型识别/描述图片（/ai/describe-image 端点的业务层实现）。
// handler 完成参数校验、图片读取与「视觉理解」路由门控后调用本函数；
// 默认指令、system prompt 组装与模型 JSON 回复解析属于业务逻辑，收敛在 service 层。
func DescribeImage(aiSvc *ai.AIService, instruction, dataURL string) (string, error) {
	instruction = strings.TrimSpace(instruction)
	if instruction == "" {
		instruction = "识别图片内容并详细描述"
	}

	systemPrompt := fmt.Sprintf(`%s

你是一个图片识别助手。请基于图片内容完成用户指定的任务（识别/描述/提取信息等）。
请严格按以下 JSON 格式输出，不要包含任何其他内容：
{"description": "对图片的识别/描述结果"}

注意：只输出图片中实际存在的信息，不要编造。`, aiprompt.CurrentTimeLine())

	messages := []ai.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: instruction, ImageURL: dataURL},
	}

	// 通过路由选择视觉 Provider / 模型，不传 Override，由 ModelRouter 按「视觉理解」路由解析
	result, err := aiSvc.GetCompletion(ai.TaskTypeVision, messages)
	if err != nil {
		return "", err
	}

	return parseDescribeResult(result), nil
}

// parseDescribeResult 从模型回复中提取描述文本。
// 模型有时把 JSON 包在 ```json 代码围栏里或带前言/尾注，直接 Unmarshal 会失败、
// 把整段 JSON 当描述吐出来（弹窗里显示一坨 JSON）。整段试解失败后用
// extractJSONObject 抽取第一个 {...} 再试（磨掉 markdown 围栏/前言，与
// avatar_trigger_service / rerank 同一抽取函数）；仍未按 JSON 返回时直接取全文
//（识别/描述场景可容忍自由文本）。
func parseDescribeResult(result string) string {
	var parsed struct {
		Description string `json:"description"`
	}
	raw := strings.TrimSpace(result)
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		if sub := extractJSONObject(raw); sub != "" {
			if err2 := json.Unmarshal([]byte(sub), &parsed); err2 == nil && parsed.Description != "" {
				return parsed.Description
			}
		}
	}
	if parsed.Description == "" {
		return raw
	}
	return parsed.Description
}
