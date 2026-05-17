package ai

import (
	"encoding/json"
	"fmt"
	"strings"

	"qim-server/pkg/logger"
)

// IntelligentTroubleshootingTool 智能故障排查工具
type IntelligentTroubleshootingTool struct {
	aiService *AIService
}

func NewIntelligentTroubleshootingTool(aiService *AIService) *IntelligentTroubleshootingTool {
	return &IntelligentTroubleshootingTool{aiService: aiService}
}

func (t *IntelligentTroubleshootingTool) Name() string {
	return "intelligent_troubleshooting"
}

func (t *IntelligentTroubleshootingTool) Description() string {
	return "智能故障排查工具，用于分析服务器故障"
}

func (t *IntelligentTroubleshootingTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"symptom": map[string]interface{}{
			"type":        "string",
			"description": "故障症状描述",
			"required":    true,
		},
		"server": map[string]interface{}{
			"type":        "string",
			"description": "服务器信息",
			"required":    false,
		},
		"logs": map[string]interface{}{
			"type":        "string",
			"description": "相关日志",
			"required":    false,
		},
	}
}

func (t *IntelligentTroubleshootingTool) Execute(params map[string]interface{}, ctx *CallerContext) (interface{}, error) {
	symptom, ok := params["symptom"].(string)
	if !ok {
		return nil, fmt.Errorf("symptom parameter is required")
	}

	server := ""
	if s, ok := params["server"].(string); ok {
		server = s
	}

	logs := ""
	if l, ok := params["logs"].(string); ok {
		logs = l
	}

	if t.aiService != nil {
		result, err := t.analyzeWithLLM(symptom, server, logs)
		if err != nil {
			logger.WithModule("IntelligentTroubleshootingTool").Error("LLM analysis failed, falling back to rule-based", "error", err)
		} else {
			return result, nil
		}
	}

	analysis := t.analyzeSymptom(symptom, logs)
	solutions := t.generateSolutions(analysis)

	return map[string]interface{}{
		"symptom":    symptom,
		"server":     server,
		"analysis":   analysis,
		"solutions":  solutions,
		"recommended": solutions[0],
	}, nil
}

type troubleshootingResult struct {
	Analysis       string   `json:"analysis"`
	PossibleCauses []string `json:"possible_causes"`
	Solutions      []string `json:"solutions"`
	Commands       []string `json:"commands"`
	Urgency        string   `json:"urgency"`
}

func (t *IntelligentTroubleshootingTool) analyzeWithLLM(symptom, server, logs string) (interface{}, error) {
	systemPrompt := `你是一个资深运维工程师，有丰富的故障排查经验。请分析用户描述的故障症状和相关日志，给出专业的诊断和解决方案。

严格按以下 JSON 格式输出，不要输出其他内容：
{
  "analysis": "故障根因分析（2-3句话，基于真实运维经验）",
  "possible_causes": ["可能原因1", "可能原因2"],
  "solutions": ["具体解决方案1（包含可执行命令）", "具体解决方案2"],
  "commands": ["可直接执行的命令1", "可直接执行的命令2"],
  "urgency": "critical|high|medium|low"
}`

	userPrompt := fmt.Sprintf("故障症状：%s\n", symptom)
	if server != "" {
		userPrompt += fmt.Sprintf("服务器信息：%s\n", server)
	}
	if logs != "" {
		userPrompt += fmt.Sprintf("相关日志：\n%s\n", logs)
	}

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	response, err := t.aiService.GetCompletion(messages)
	if err != nil {
		return nil, fmt.Errorf("LLM completion failed: %w", err)
	}

	var result troubleshootingResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w", err)
	}

	return map[string]interface{}{
		"symptom":        symptom,
		"server":         server,
		"analysis":       result.Analysis,
		"possible_causes": result.PossibleCauses,
		"solutions":      result.Solutions,
		"commands":       result.Commands,
		"urgency":        result.Urgency,
		"recommended":    result.Solutions[0],
		"source":         "llm",
	}, nil
}

func (t *IntelligentTroubleshootingTool) analyzeSymptom(symptom, logs string) string {
	// 简单的故障分析逻辑
	if strings.Contains(symptom, "连接") || strings.Contains(symptom, "无法访问") {
		return "网络连接问题，可能是网络配置、防火墙规则或服务未启动"
	} else if strings.Contains(symptom, "CPU") || strings.Contains(symptom, "内存") {
		return "资源使用问题，可能是进程占用过高或内存泄漏"
	} else if strings.Contains(symptom, "磁盘") || strings.Contains(symptom, "空间") {
		return "存储问题，可能是磁盘空间不足或文件系统损坏"
	} else if strings.Contains(symptom, "数据库") {
		return "数据库问题，可能是连接数过多、查询缓慢或服务异常"
	} else {
		return "未知故障类型，需要进一步分析"
	}
}

func (t *IntelligentTroubleshootingTool) generateSolutions(analysis string) []string {
	// 根据分析结果生成解决方案
	if strings.Contains(analysis, "网络") {
		return []string{
			"检查网络连接和防火墙规则",
			"验证服务是否正常运行",
			"检查网络配置和路由设置",
		}
	} else if strings.Contains(analysis, "资源") {
		return []string{
			"检查进程使用情况，结束异常进程",
			"增加服务器资源或优化应用",
			"检查内存泄漏问题",
		}
	} else if strings.Contains(analysis, "存储") {
		return []string{
			"清理磁盘空间，删除不必要的文件",
			"扩展磁盘容量",
			"检查文件系统完整性",
		}
	} else if strings.Contains(analysis, "数据库") {
		return []string{
			"优化数据库查询和索引",
			"增加数据库连接池容量",
			"检查数据库服务状态",
		}
	} else {
		return []string{
			"收集更多故障信息",
			"检查系统日志和应用日志",
			"重启相关服务",
		}
	}
}

// CommandGenerationTool 命令生成工具
type CommandGenerationTool struct {
	aiService *AIService
}

func NewCommandGenerationTool(aiService *AIService) *CommandGenerationTool {
	return &CommandGenerationTool{aiService: aiService}
}

func (t *CommandGenerationTool) Name() string {
	return "command_generation"
}

func (t *CommandGenerationTool) Description() string {
	return "命令生成工具，根据描述生成运维命令"
}

func (t *CommandGenerationTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"description": map[string]interface{}{
			"type":        "string",
			"description": "命令功能描述",
			"required":    true,
		},
		"platform": map[string]interface{}{
			"type":        "string",
			"description": "目标平台: linux, windows, macos",
			"required":    false,
			"default":     "linux",
		},
		"format": map[string]interface{}{
			"type":        "string",
			"description": "输出格式: single, script",
			"required":    false,
			"default":     "single",
		},
	}
}

func (t *CommandGenerationTool) Execute(params map[string]interface{}, ctx *CallerContext) (interface{}, error) {
	description, ok := params["description"].(string)
	if !ok {
		return nil, fmt.Errorf("description parameter is required")
	}

	platform := "linux"
	if p, ok := params["platform"].(string); ok {
		platform = p
	}

	format := "single"
	if f, ok := params["format"].(string); ok {
		format = f
	}

	if t.aiService != nil {
		result, err := t.generateWithLLM(description, platform, format)
		if err != nil {
			logger.WithModule("CommandGenerationTool").Error("LLM generation failed, falling back to rule-based", "error", err)
		} else {
			return result, nil
		}
	}

	command := t.generateCommand(description, platform, format)

	return map[string]interface{}{
		"description": description,
		"platform":    platform,
		"format":      format,
		"command":     command,
		"explanation": t.explainCommand(command),
	}, nil
}

type commandGenerationResult struct {
	Command      string   `json:"command"`
	Explanation  string   `json:"explanation"`
	Alternatives []string `json:"alternatives"`
	Warnings     []string `json:"warnings"`
}

func (t *CommandGenerationTool) generateWithLLM(description, platform, format string) (interface{}, error) {
	systemPrompt := `你是一个资深运维工程师，精通各种系统命令。请根据用户描述生成最合适的运维命令。

严格按以下 JSON 格式输出，不要输出其他内容：
{
  "command": "主要命令（可直接执行）",
  "explanation": "命令解释（说明命令的作用和参数）",
  "alternatives": ["备选命令1", "备选命令2"],
  "warnings": ["注意事项1", "注意事项2"]
}

要求：
1. 命令必须准确、安全、高效
2. 解释要清晰易懂
3. 提供合理的备选方案
4. 列出可能的风险和注意事项`

	userPrompt := fmt.Sprintf("请生成命令：\n描述：%s\n平台：%s\n格式：%s", description, platform, format)

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	response, err := t.aiService.GetCompletion(messages)
	if err != nil {
		return nil, fmt.Errorf("LLM completion failed: %w", err)
	}

	var result commandGenerationResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w", err)
	}

	return map[string]interface{}{
		"description":   description,
		"platform":      platform,
		"format":        format,
		"command":       result.Command,
		"explanation":   result.Explanation,
		"alternatives":  result.Alternatives,
		"warnings":      result.Warnings,
		"source":        "llm",
	}, nil
}

func (t *CommandGenerationTool) generateCommand(description, platform, format string) string {
	// 简单的命令生成逻辑
	description = strings.ToLower(description)

	if strings.Contains(description, "重启") && strings.Contains(description, "nginx") {
		if platform == "linux" {
			if format == "script" {
				return "#!/bin/bash\nsudo systemctl restart nginx\nsudo systemctl status nginx"
			} else {
				return "sudo systemctl restart nginx"
			}
		}
	} else if strings.Contains(description, "查看") && strings.Contains(description, "进程") {
		if platform == "linux" {
			if format == "script" {
				return "#!/bin/bash\nps aux | grep nginx\ntop -b -n 1 | head -20"
			} else {
				return "ps aux | grep nginx"
			}
		}
	} else if strings.Contains(description, "磁盘") && strings.Contains(description, "空间") {
		if platform == "linux" {
			if format == "script" {
				return "#!/bin/bash\ndf -h\ndu -h --max-depth=1 /"
			} else {
				return "df -h"
			}
		}
	} else if strings.Contains(description, "网络") && strings.Contains(description, "连接") {
		if platform == "linux" {
			if format == "script" {
				return "#!/bin/bash\nip addr\nnetstat -tuln\nping -c 4 google.com"
			} else {
				return "netstat -tuln"
			}
		}
	}

	// 默认命令
	return "echo \"Command not found for: " + description + "\""
}

func (t *CommandGenerationTool) explainCommand(command string) string {
	// 命令解释
	if strings.Contains(command, "systemctl restart nginx") {
		return "重启Nginx服务并查看状态"
	} else if strings.Contains(command, "ps aux | grep nginx") {
		return "查看Nginx进程信息"
	} else if strings.Contains(command, "df -h") {
		return "查看磁盘空间使用情况"
	} else if strings.Contains(command, "netstat -tuln") {
		return "查看网络连接和监听端口"
	} else {
		return "命令执行操作"
	}
}

// LogAnalysisTool 日志分析工具
type LogAnalysisTool struct {
	aiService *AIService
}

func NewLogAnalysisTool(aiService *AIService) *LogAnalysisTool {
	return &LogAnalysisTool{aiService: aiService}
}

func (t *LogAnalysisTool) Name() string {
	return "log_analysis"
}

func (t *LogAnalysisTool) Description() string {
	return "日志分析工具，用于分析服务器日志"
}

func (t *LogAnalysisTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"log_content": map[string]interface{}{
			"type":        "string",
			"description": "日志内容",
			"required":    true,
		},
		"service": map[string]interface{}{
			"type":        "string",
			"description": "服务名称",
			"required":    false,
		},
		"severity": map[string]interface{}{
			"type":        "string",
			"description": "严重程度: error, warning, info",
			"required":    false,
		},
	}
}

func (t *LogAnalysisTool) Execute(params map[string]interface{}, ctx *CallerContext) (interface{}, error) {
	logContent, ok := params["log_content"].(string)
	if !ok {
		return nil, fmt.Errorf("log_content parameter is required")
	}

	service := ""
	if s, ok := params["service"].(string); ok {
		service = s
	}

	severity := ""
	if sev, ok := params["severity"].(string); ok {
		severity = sev
	}

	if t.aiService != nil {
		result, err := t.analyzeWithLLM(logContent, service, severity)
		if err != nil {
			logger.WithModule("LogAnalysisTool").Error("LLM analysis failed, falling back to rule-based", "error", err)
		} else {
			return result, nil
		}
	}

	errorCount := t.countErrors(logContent)
	warningCount := t.countWarnings(logContent)
	patterns := t.extractPatterns(logContent)
	recommendations := t.generateRecommendations(patterns)

	return map[string]interface{}{
		"service":         service,
		"severity":        severity,
		"error_count":     errorCount,
		"warning_count":   warningCount,
		"patterns":        patterns,
		"recommendations": recommendations,
	}, nil
}

type logAnalysisResult struct {
	Summary         string   `json:"summary"`
	Anomalies       []string `json:"anomalies"`
	Recommendations []string `json:"recommendations"`
	ErrorCount      int      `json:"error_count"`
	WarningCount    int      `json:"warning_count"`
}

func (t *LogAnalysisTool) analyzeWithLLM(logContent, service, severity string) (interface{}, error) {
	systemPrompt := `你是一个资深运维工程师，精通日志分析。请分析提供的日志内容，识别问题并给出建议。

严格按以下 JSON 格式输出，不要输出其他内容：
{
  "summary": "日志分析摘要（2-3句话概括主要发现）",
  "anomalies": ["异常1", "异常2"],
  "recommendations": ["建议1", "建议2"],
  "error_count": 错误数量,
  "warning_count": 警告数量
}

要求：
1. 准确识别日志中的错误和警告
2. 分析异常模式和根因
3. 给出具体可操作的建议
4. 统计要准确`

	userPrompt := fmt.Sprintf("请分析日志：\n服务：%s\n严重程度：%s\n日志内容：\n%s", service, severity, logContent)

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	response, err := t.aiService.GetCompletion(messages)
	if err != nil {
		return nil, fmt.Errorf("LLM completion failed: %w", err)
	}

	var result logAnalysisResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w", err)
	}

	return map[string]interface{}{
		"service":         service,
		"severity":        severity,
		"summary":         result.Summary,
		"anomalies":       result.Anomalies,
		"recommendations": result.Recommendations,
		"error_count":     result.ErrorCount,
		"warning_count":   result.WarningCount,
		"source":          "llm",
	}, nil
}

func (t *LogAnalysisTool) countErrors(logContent string) int {
	return strings.Count(logContent, "ERROR")
}

func (t *LogAnalysisTool) countWarnings(logContent string) int {
	return strings.Count(logContent, "WARNING")
}

func (t *LogAnalysisTool) extractPatterns(logContent string) []string {
	patterns := []string{}

	if strings.Contains(logContent, "connection refused") {
		patterns = append(patterns, "连接被拒绝")
	}
	if strings.Contains(logContent, "timeout") {
		patterns = append(patterns, "超时")
	}
	if strings.Contains(logContent, "out of memory") {
		patterns = append(patterns, "内存不足")
	}
	if strings.Contains(logContent, "disk full") {
		patterns = append(patterns, "磁盘空间不足")
	}

	return patterns
}

func (t *LogAnalysisTool) generateRecommendations(patterns []string) []string {
	recommendations := []string{}

	for _, pattern := range patterns {
		switch pattern {
		case "连接被拒绝":
			recommendations = append(recommendations, "检查服务是否运行，端口是否开放")
		case "超时":
			recommendations = append(recommendations, "检查网络连接和服务响应时间")
		case "内存不足":
			recommendations = append(recommendations, "增加内存或优化应用内存使用")
		case "磁盘空间不足":
			recommendations = append(recommendations, "清理磁盘空间或扩展磁盘容量")
		}
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "无明显问题，建议继续监控")
	}

	return recommendations
}

// IntelligentAlertTool 智能告警处理工具
type IntelligentAlertTool struct {
	aiService *AIService
}

func NewIntelligentAlertTool(aiService *AIService) *IntelligentAlertTool {
	return &IntelligentAlertTool{aiService: aiService}
}

func (t *IntelligentAlertTool) Name() string {
	return "intelligent_alert"
}

func (t *IntelligentAlertTool) Description() string {
	return "智能告警处理工具，用于分析和处理告警"
}

func (t *IntelligentAlertTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"alert_content": map[string]interface{}{
			"type":        "string",
			"description": "告警内容",
			"required":    true,
		},
		"severity": map[string]interface{}{
			"type":        "string",
			"description": "告警级别: critical, warning, info",
			"required":    false,
		},
		"service": map[string]interface{}{
			"type":        "string",
			"description": "相关服务",
			"required":    false,
		},
	}
}

func (t *IntelligentAlertTool) Execute(params map[string]interface{}, ctx *CallerContext) (interface{}, error) {
	alertContent, ok := params["alert_content"].(string)
	if !ok {
		return nil, fmt.Errorf("alert_content parameter is required")
	}

	severity := "warning"
	if sev, ok := params["severity"].(string); ok {
		severity = sev
	}

	service := ""
	if s, ok := params["service"].(string); ok {
		service = s
	}

	if t.aiService != nil {
		result, err := t.analyzeWithLLM(alertContent, severity, service)
		if err != nil {
			logger.WithModule("IntelligentAlertTool").Error("LLM analysis failed, falling back to rule-based", "error", err)
		} else {
			return result, nil
		}
	}

	category := t.categorizeAlert(alertContent)
	priority := t.calculatePriority(severity, category)
	actions := t.generateActions(category, priority)

	return map[string]interface{}{
		"alert_content": alertContent,
		"severity":      severity,
		"service":       service,
		"category":      category,
		"priority":      priority,
		"actions":       actions,
		"auto_resolve":  t.canAutoResolve(category),
	}, nil
}

type alertAnalysisResult struct {
	Severity    string   `json:"severity"`
	Category    string   `json:"category"`
	Action      string   `json:"action"`
	RelatedLogs []string `json:"related_logs"`
	Priority    string   `json:"priority"`
}

func (t *IntelligentAlertTool) analyzeWithLLM(alertContent, severity, service string) (interface{}, error) {
	systemPrompt := `你是一个资深运维工程师，精通告警分析和处理。请分析告警内容，给出处理建议。

严格按以下 JSON 格式输出，不要输出其他内容：
{
  "severity": "critical|high|medium|low",
  "category": "告警分类（如：CPU使用过高、内存使用过高、磁盘空间不足、服务宕机、网络错误等）",
  "action": "建议的处理动作",
  "related_logs": ["相关日志文件路径1", "相关日志文件路径2"],
  "priority": "高|中|低"
}

要求：
1. 准确判断告警严重程度
2. 正确分类告警类型
3. 给出具体可执行的处理建议
4. 列出需要检查的相关日志`

	userPrompt := fmt.Sprintf("请分析告警：\n告警内容：%s\n原始级别：%s\n相关服务：%s", alertContent, severity, service)

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	response, err := t.aiService.GetCompletion(messages)
	if err != nil {
		return nil, fmt.Errorf("LLM completion failed: %w", err)
	}

	var result alertAnalysisResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w", err)
	}

	return map[string]interface{}{
		"alert_content": alertContent,
		"severity":      result.Severity,
		"service":       service,
		"category":      result.Category,
		"priority":      result.Priority,
		"action":        result.Action,
		"related_logs":  result.RelatedLogs,
		"auto_resolve":  result.Severity == "low" || result.Severity == "medium",
		"source":        "llm",
	}, nil
}

func (t *IntelligentAlertTool) categorizeAlert(alertContent string) string {
	alertContent = strings.ToLower(alertContent)

	if strings.Contains(alertContent, "cpu") && strings.Contains(alertContent, "high") {
		return "CPU使用过高"
	} else if strings.Contains(alertContent, "memory") && strings.Contains(alertContent, "high") {
		return "内存使用过高"
	} else if strings.Contains(alertContent, "disk") && strings.Contains(alertContent, "full") {
		return "磁盘空间不足"
	} else if strings.Contains(alertContent, "service") && strings.Contains(alertContent, "down") {
		return "服务宕机"
	} else if strings.Contains(alertContent, "network") && strings.Contains(alertContent, "error") {
		return "网络错误"
	} else {
		return "其他告警"
	}
}

func (t *IntelligentAlertTool) calculatePriority(severity, category string) string {
	if severity == "critical" {
		return "高"
	} else if severity == "warning" && (category == "服务宕机" || category == "磁盘空间不足") {
		return "高"
	} else if severity == "warning" {
		return "中"
	} else {
		return "低"
	}
}

func (t *IntelligentAlertTool) generateActions(category, priority string) []string {
	actions := []string{}

	switch category {
	case "CPU使用过高":
		actions = append(actions, "检查占用CPU的进程", "优化应用性能", "考虑增加CPU资源")
	case "内存使用过高":
		actions = append(actions, "检查内存使用情况", "清理内存缓存", "考虑增加内存")
	case "磁盘空间不足":
		actions = append(actions, "清理磁盘空间", "删除不必要的文件", "扩展磁盘容量")
	case "服务宕机":
		actions = append(actions, "重启服务", "检查服务日志", "分析宕机原因")
	case "网络错误":
		actions = append(actions, "检查网络连接", "验证网络配置", "重启网络服务")
	default:
		actions = append(actions, "分析告警内容", "采取相应措施")
	}

	if priority == "高" {
		actions = append([]string{"立即处理"}, actions...)
	}

	return actions
}

func (t *IntelligentAlertTool) canAutoResolve(category string) bool {
	// 可以自动解决的告警类型
	autoResolveCategories := []string{
		"CPU使用过高",
		"内存使用过高",
	}

	for _, cat := range autoResolveCategories {
		if cat == category {
			return true
		}
	}

	return false
}

// OpsKnowledgeTool 运维知识问答工具
type OpsKnowledgeTool struct {
	aiService *AIService
}

func NewOpsKnowledgeTool(aiService *AIService) *OpsKnowledgeTool {
	return &OpsKnowledgeTool{aiService: aiService}
}

func (t *OpsKnowledgeTool) Name() string {
	return "ops_knowledge"
}

func (t *OpsKnowledgeTool) Description() string {
	return "运维知识问答工具，回答运维相关问题"
}

func (t *OpsKnowledgeTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"question": map[string]interface{}{
			"type":        "string",
			"description": "问题内容",
			"required":    true,
		},
		"category": map[string]interface{}{
			"type":        "string",
			"description": "问题类别: linux, network, database, security",
			"required":    false,
		},
	}
}

func (t *OpsKnowledgeTool) Execute(params map[string]interface{}, ctx *CallerContext) (interface{}, error) {
	question, ok := params["question"].(string)
	if !ok {
		return nil, fmt.Errorf("question parameter is required")
	}

	category := ""
	if c, ok := params["category"].(string); ok {
		category = c
	}

	if t.aiService != nil {
		result, err := t.answerWithLLM(question, category)
		if err != nil {
			logger.WithModule("OpsKnowledgeTool").Error("LLM answer failed, falling back to rule-based", "error", err)
		} else {
			return result, nil
		}
	}

	answer := t.answerQuestion(question, category)
	references := t.getReferences(question, category)

	return map[string]interface{}{
		"question":    question,
		"category":    category,
		"answer":      answer,
		"references":  references,
		"recommended": t.getRecommendedActions(question),
	}, nil
}

type opsKnowledgeResult struct {
	Answer        string   `json:"answer"`
	References    []string `json:"references"`
	RelatedTopics []string `json:"related_topics"`
	Commands      []string `json:"commands"`
}

func (t *OpsKnowledgeTool) answerWithLLM(question, category string) (interface{}, error) {
	systemPrompt := `你是一个资深运维工程师，有丰富的运维经验和知识。请回答用户的运维相关问题。

严格按以下 JSON 格式输出，不要输出其他内容：
{
  "answer": "详细回答（包含原理说明和操作步骤）",
  "references": ["参考资料1", "参考资料2"],
  "related_topics": ["相关主题1", "相关主题2"],
  "commands": ["相关命令1", "相关命令2"]
}

要求：
1. 回答要专业、准确、详细
2. 提供实际可操作的命令或步骤
3. 列出相关的参考资料
4. 提供相关主题供用户深入学习`

	userPrompt := fmt.Sprintf("请回答运维问题：\n问题：%s\n类别：%s", question, category)

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	response, err := t.aiService.GetCompletion(messages)
	if err != nil {
		return nil, fmt.Errorf("LLM completion failed: %w", err)
	}

	var result opsKnowledgeResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response as JSON: %w", err)
	}

	return map[string]interface{}{
		"question":       question,
		"category":       category,
		"answer":         result.Answer,
		"references":     result.References,
		"related_topics": result.RelatedTopics,
		"commands":       result.Commands,
		"source":         "llm",
	}, nil
}

func (t *OpsKnowledgeTool) answerQuestion(question, category string) string {
	// 简单的知识问答逻辑
	question = strings.ToLower(question)

	if strings.Contains(question, "如何重启") && strings.Contains(question, "nginx") {
		return "可以使用以下命令重启Nginx服务：\n1. sudo systemctl restart nginx\n2. 或者 sudo service nginx restart\n重启后可以使用 sudo systemctl status nginx 检查服务状态"
	} else if strings.Contains(question, "如何查看") && strings.Contains(question, "磁盘空间") {
		return "可以使用以下命令查看磁盘空间使用情况：\n1. df -h  # 查看所有挂载点的磁盘使用情况\n2. du -h --max-depth=1 /  # 查看根目录下各目录的空间使用情况"
	} else if strings.Contains(question, "如何查看") && strings.Contains(question, "进程") {
		return "可以使用以下命令查看进程信息：\n1. ps aux  # 查看所有进程\n2. top  # 实时查看进程资源使用情况\n3. htop  # 交互式进程查看工具"
	} else if strings.Contains(question, "如何配置") && strings.Contains(question, "防火墙") {
		return "可以使用以下命令配置防火墙：\n1. 对于iptables：sudo iptables -A INPUT -p tcp --dport 80 -j ACCEPT\n2. 对于firewalld：sudo firewall-cmd --add-port=80/tcp --permanent && sudo firewall-cmd --reload"
	} else {
		return "抱歉，我无法回答这个问题。请提供更具体的运维相关问题。"
	}
}

func (t *OpsKnowledgeTool) getReferences(question, category string) []string {
	// 提供参考资料
	references := []string{}

	if strings.Contains(question, "nginx") {
		references = append(references, "Nginx官方文档: https://nginx.org/en/docs/")
	} else if strings.Contains(question, "磁盘") {
		references = append(references, "Linux磁盘管理指南: https://linux.die.net/man/")
	} else if strings.Contains(question, "进程") {
		references = append(references, "Linux进程管理: https://man7.org/linux/man-pages/")
	} else if strings.Contains(question, "防火墙") {
		references = append(references, "Linux防火墙配置: https://www.linux.com/")
	}

	return references
}

func (t *OpsKnowledgeTool) getRecommendedActions(question string) []string {
	// 推荐操作
	actions := []string{}

	if strings.Contains(question, "重启") {
		actions = append(actions, "在重启服务前，确保已保存所有配置更改", "重启后验证服务是否正常运行")
	} else if strings.Contains(question, "查看") {
		actions = append(actions, "定期检查系统状态", "设置监控告警")
	} else if strings.Contains(question, "配置") {
		actions = append(actions, "在修改配置前备份原始配置", "测试配置更改是否生效")
	}

	return actions
}
