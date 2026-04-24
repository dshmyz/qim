package ai

import (
	"fmt"
	"strings"
)

// OpsTool 运维工具接口
type OpsTool interface {
	Name() string
	Description() string
	Parameters() map[string]interface{}
	Execute(params map[string]interface{}) (interface{}, error)
}

// IntelligentTroubleshootingTool 智能故障排查工具
type IntelligentTroubleshootingTool struct{}

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

func (t *IntelligentTroubleshootingTool) Execute(params map[string]interface{}) (interface{}, error) {
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

	// 智能故障分析逻辑
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
type CommandGenerationTool struct{}

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

func (t *CommandGenerationTool) Execute(params map[string]interface{}) (interface{}, error) {
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

	// 生成命令
	command := t.generateCommand(description, platform, format)

	return map[string]interface{}{
		"description": description,
		"platform":    platform,
		"format":      format,
		"command":     command,
		"explanation": t.explainCommand(command),
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
type LogAnalysisTool struct{}

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

func (t *LogAnalysisTool) Execute(params map[string]interface{}) (interface{}, error) {
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

	// 分析日志
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
type IntelligentAlertTool struct{}

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

func (t *IntelligentAlertTool) Execute(params map[string]interface{}) (interface{}, error) {
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

	// 分析告警
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
type OpsKnowledgeTool struct{}

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

func (t *OpsKnowledgeTool) Execute(params map[string]interface{}) (interface{}, error) {
	question, ok := params["question"].(string)
	if !ok {
		return nil, fmt.Errorf("question parameter is required")
	}

	category := ""
	if c, ok := params["category"].(string); ok {
		category = c
	}

	// 回答问题
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
