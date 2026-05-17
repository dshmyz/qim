package service

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ledongthuc/pdf"
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

// parsePDF 使用 ledongthuc/pdf 提取 PDF 文本内容
func (p *DocumentParser) parsePDF(filePath string) (string, error) {
	f, r, err := pdf.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("打开 PDF %s 失败: %w", filePath, err)
	}
	defer f.Close()

	var texts []string
	numPages := r.NumPage()
	for i := 1; i <= numPages; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			continue
		}
		content, err := page.GetPlainText(nil)
		if err != nil {
			continue
		}
		text := strings.TrimSpace(content)
		if text != "" {
			texts = append(texts, text)
		}
	}

	if len(texts) == 0 {
		return "", fmt.Errorf("PDF %s 无法提取文本内容", filePath)
	}
	return strings.Join(texts, "\n\n"), nil
}

// parseDocx 解析 DOCX 文件（ZIP 内的 word/document.xml）
func (p *DocumentParser) parseDocx(filePath string) (string, error) {
	zr, err := zip.OpenReader(filePath)
	if err != nil {
		return "", fmt.Errorf("打开 DOCX %s 失败: %w", filePath, err)
	}
	defer zr.Close()

	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", fmt.Errorf("读取 document.xml 失败: %w", err)
			}
			defer rc.Close()
			data, err := io.ReadAll(rc)
			if err != nil {
				return "", fmt.Errorf("读取 document.xml 内容失败: %w", err)
			}
			return extractDocxText(data), nil
		}
	}

	return "", fmt.Errorf("DOCX %s 中未找到 word/document.xml", filePath)
}

// docxXML 解析 DOCX XML 所需的结构
type docxXML struct {
	XMLName xml.Name `xml:"document"`
	Body    docxBody `xml:"body"`
}

type docxBody struct {
	Paragraphs []docxParagraph `xml:"p"`
}

type docxParagraph struct {
	RunItems []docxRun `xml:"r"`
}

type docxRun struct {
	Text string `xml:"t"`
}

func extractDocxText(data []byte) string {
	var doc docxXML
	if err := xml.Unmarshal(data, &doc); err != nil {
		// 降级：用正则提取 <w:t> 内容
		return regexExtractDocxText(data)
	}

	var paragraphs []string
	for _, p := range doc.Body.Paragraphs {
		var runs []string
		for _, r := range p.RunItems {
			if r.Text != "" {
				runs = append(runs, r.Text)
			}
		}
		if len(runs) > 0 {
			paragraphs = append(paragraphs, strings.Join(runs, ""))
		}
	}
	return strings.Join(paragraphs, "\n")
}

func regexExtractDocxText(data []byte) string {
	// 简单提取所有 <w:t ...>text</w:t> 内容
	content := string(data)
	var texts []string
	for {
		start := strings.Index(content, "<w:t")
		if start == -1 {
			break
		}
		endTag := strings.Index(content[start:], ">")
		if endTag == -1 {
			break
		}
		textStart := start + endTag + 1
		closeTag := strings.Index(content[textStart:], "</w:t>")
		if closeTag == -1 {
			break
		}
		text := strings.TrimSpace(content[textStart:textStart+closeTag])
		if text != "" {
			texts = append(texts, text)
		}
		content = content[textStart+closeTag+len("</w:t>"):]
	}
	return strings.Join(texts, "\n")
}
