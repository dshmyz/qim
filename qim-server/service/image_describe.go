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
// 模型未按 JSON 格式返回时直接取全文（识别/描述场景可容忍自由文本）。
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

	var parsed struct {
		Description string `json:"description"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(result)), &parsed); err != nil || parsed.Description == "" {
		parsed.Description = strings.TrimSpace(result)
	}
	return parsed.Description, nil
}
