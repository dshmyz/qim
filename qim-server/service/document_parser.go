package service

import (
	"fmt"
	"os"
	"strings"
)

// DocumentParser 文档内容解析器
type DocumentParser struct{}

// NewDocumentParser 创建文档解析器实例
func NewDocumentParser() *DocumentParser {
	return &DocumentParser{}
}

// Parse 根据文件扩展名解析文档内容
func (p *DocumentParser) Parse(filePath string) (string, error) {
	ext := strings.ToLower(filePath[strings.LastIndex(filePath, ".")+1:])

	switch ext {
	case "txt", "md", "markdown":
		return p.parseText(filePath)
	case "pdf":
		return p.parsePDF(filePath)
	case "docx":
		return p.parseDocx(filePath)
	default:
		// 默认尝试按文本解析
		return p.parseText(filePath)
	}
}

// parseText 解析纯文本文件
func (p *DocumentParser) parseText(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("读取文本文件 %s 失败: %w", filePath, err)
	}
	return string(data), nil
}

// parsePDF 解析 PDF 文件
func (p *DocumentParser) parsePDF(filePath string) (string, error) {
	// 简化实现：直接读取文件内容（需要引入 ledongthuc/pdf 库完善）
	// 目前先返回空字符串，后续可以接入 PDF 解析库
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("读取 PDF 文件 %s 失败: %w", filePath, err)
	}
	// 简单返回文件内容作为占位
	return string(data), nil
}

// parseDocx 解析 DOCX 文件
func (p *DocumentParser) parseDocx(filePath string) (string, error) {
	// 简化实现：需要引入 github.com/unidoc/unioffice/document 库
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("读取 DOCX 文件 %s 失败: %w", filePath, err)
	}
	// 简单返回文件内容作为占位
	return string(data), nil
}
