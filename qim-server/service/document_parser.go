package service

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/ledongthuc/pdf"
)

// maxDocumentSize 文档解析的单文件大小上限（20MB）
// 防止 zip bomb / 超大文件导致 OOM
const maxDocumentSize = 20 * 1024 * 1024

// maxZipEntrySize ZIP 内单个成员的解压读取上限（20MB）
// 防止压缩比极高的恶意 zip bomb 解压后撑爆内存
const maxZipEntrySize = 20 * 1024 * 1024

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
	case "pptx":
		return p.parsePptx(filePath)
	case "xlsx":
		return p.parseXlsx(filePath)
	default:
		return p.parseText(filePath)
	}
}

// parseText 解析纯文本文件
// 先 stat 检查文件大小，防止超大文件一次性读入内存
func (p *DocumentParser) parseText(filePath string) (string, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("获取文本文件信息 %s 失败: %w", filePath, err)
	}
	if info.Size() > maxDocumentSize {
		return "", fmt.Errorf("文本文件 %s 过大（%d 字节，上限 %d 字节）", filePath, info.Size(), maxDocumentSize)
	}
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
			// 限制解压读取大小，防止 zip bomb
			data, err := io.ReadAll(io.LimitReader(rc, maxZipEntrySize+1))
			if err != nil {
				return "", fmt.Errorf("读取 document.xml 内容失败: %w", err)
			}
			if len(data) > maxZipEntrySize {
				return "", fmt.Errorf("document.xml 解压后过大（超过 %d 字节，疑似 zip bomb）", maxZipEntrySize)
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

// parsePptx 解析 PPTX 文件（ZIP 内的 ppt/slides/slideN.xml，提取 <a:t> 文本）
func (p *DocumentParser) parsePptx(filePath string) (string, error) {
	zr, err := zip.OpenReader(filePath)
	if err != nil {
		return "", fmt.Errorf("打开 PPTX %s 失败: %w", filePath, err)
	}
	defer zr.Close()

	// 收集所有 slide XML 并按编号排序
	type slideFile struct {
		num  int
		name string
	}
	var slides []slideFile
	for _, f := range zr.File {
		if !strings.HasPrefix(f.Name, "ppt/slides/slide") || !strings.HasSuffix(f.Name, ".xml") {
			continue
		}
		numStr := strings.TrimSuffix(strings.TrimPrefix(f.Name, "ppt/slides/slide"), ".xml")
		num, err := strconv.Atoi(numStr)
		if err != nil {
			continue
		}
		slides = append(slides, slideFile{num: num, name: f.Name})
	}
	sort.Slice(slides, func(i, j int) bool { return slides[i].num < slides[j].num })

	if len(slides) == 0 {
		return "", fmt.Errorf("PPTX %s 中未找到幻灯片", filePath)
	}

	var slideTexts []string
	for _, sf := range slides {
		f, _ := zr.Open(sf.name)
		if f == nil {
			continue
		}
		// 限制解压读取大小，防止 zip bomb
		data, err := io.ReadAll(io.LimitReader(f, maxZipEntrySize+1))
		f.Close()
		if err != nil {
			continue
		}
		if len(data) > maxZipEntrySize {
			return "", fmt.Errorf("幻灯片 %d 解压后过大（超过 %d 字节，疑似 zip bomb）", sf.num, maxZipEntrySize)
		}
		text := extractPptxText(data)
		if text != "" {
			slideTexts = append(slideTexts, fmt.Sprintf("--- 幻灯片 %d ---\n%s", sf.num, text))
		}
	}

	if len(slideTexts) == 0 {
		return "", fmt.Errorf("PPTX %s 无法提取文本内容", filePath)
	}
	return strings.Join(slideTexts, "\n\n"), nil
}

// extractPptxText 从 slide XML 中提取所有 <a:t>...</a:t> 文本
func extractPptxText(data []byte) string {
	content := string(data)
	var texts []string
	searchContent := content
	for {
		start := strings.Index(searchContent, "<a:t")
		if start == -1 {
			break
		}
		endTag := strings.Index(searchContent[start:], ">")
		if endTag == -1 {
			break
		}
		textStart := start + endTag + 1
		closeTag := strings.Index(searchContent[textStart:], "</a:t>")
		if closeTag == -1 {
			break
		}
		text := strings.TrimSpace(searchContent[textStart : textStart+closeTag])
		if text != "" {
			texts = append(texts, text)
		}
		searchContent = searchContent[textStart+closeTag+len("</a:t>"):]
	}
	return strings.Join(texts, "\n")
}

// parseXlsx 解析 XLSX 文件（sharedStrings + worksheets，输出为文本）
func (p *DocumentParser) parseXlsx(filePath string) (string, error) {
	zr, err := zip.OpenReader(filePath)
	if err != nil {
		return "", fmt.Errorf("打开 XLSX %s 失败: %w", filePath, err)
	}
	defer zr.Close()

	// 1. 读取共享字符串表
	sharedStrings, err := readXlsxSharedStrings(&zr.Reader)
	if err != nil {
		sharedStrings = nil // 可能没有共享字符串（纯数字表格）
	}

	// 2. 收集并排序 worksheet
	type sheetFile struct {
		num  int
		name string
	}
	var sheets []sheetFile
	for _, f := range zr.File {
		if !strings.HasPrefix(f.Name, "xl/worksheets/sheet") || !strings.HasSuffix(f.Name, ".xml") {
			continue
		}
		numStr := strings.TrimSuffix(strings.TrimPrefix(f.Name, "xl/worksheets/sheet"), ".xml")
		num, err := strconv.Atoi(numStr)
		if err != nil {
			continue
		}
		sheets = append(sheets, sheetFile{num: num, name: f.Name})
	}
	sort.Slice(sheets, func(i, j int) bool { return sheets[i].num < sheets[j].num })

	if len(sheets) == 0 {
		return "", fmt.Errorf("XLSX %s 中未找到工作表", filePath)
	}

	// 3. 解析每个工作表
	var sheetTexts []string
	for idx, sf := range sheets {
		f, err := zr.Open(sf.name)
		if err != nil {
			continue
		}
		// 限制解压读取大小，防止 zip bomb
		data, err := io.ReadAll(io.LimitReader(f, maxZipEntrySize+1))
		f.Close()
		if err != nil {
			continue
		}
		if len(data) > maxZipEntrySize {
			return "", fmt.Errorf("工作表 %d 解压后过大（超过 %d 字节，疑似 zip bomb）", idx+1, maxZipEntrySize)
		}
		text := parseXlsxSheet(data, sharedStrings)
		if text != "" {
			sheetTexts = append(sheetTexts, fmt.Sprintf("--- 工作表 %d ---\n%s", idx+1, text))
		}
	}

	if len(sheetTexts) == 0 {
		return "", fmt.Errorf("XLSX %s 无法提取文本内容", filePath)
	}
	return strings.Join(sheetTexts, "\n\n"), nil
}

// xlsxSST 共享字符串表结构
type xlsxSST struct {
	Items []xlsxSI `xml:"si"`
}

type xlsxSI struct {
	Text string    `xml:"t"`      // 简单文本
	Runs []xlsxRun `xml:"r"`      // 富文本片段
}

type xlsxRun struct {
	Text string `xml:"t"`
}

// readXlsxSharedStrings 从 ZIP 中读取并解析 xl/sharedStrings.xml
func readXlsxSharedStrings(zr *zip.Reader) ([]string, error) {
	var sstFile *zip.File
	for _, f := range zr.File {
		if f.Name == "xl/sharedStrings.xml" {
			sstFile = f
			break
		}
	}
	if sstFile == nil {
		return nil, fmt.Errorf("sharedStrings.xml not found")
	}

	rc, err := sstFile.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	// 限制解压读取大小，防止 zip bomb
	data, err := io.ReadAll(io.LimitReader(rc, maxZipEntrySize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxZipEntrySize {
		return nil, fmt.Errorf("sharedStrings.xml 解压后过大（超过 %d 字节，疑似 zip bomb）", maxZipEntrySize)
	}

	var sst xlsxSST
	if err := xml.Unmarshal(data, &sst); err != nil {
		return nil, err
	}

	result := make([]string, len(sst.Items))
	for i, si := range sst.Items {
		if si.Text != "" {
			result[i] = si.Text
		} else {
			// 富文本：拼接所有 run 的文本
			var parts []string
			for _, r := range si.Runs {
				if r.Text != "" {
					parts = append(parts, r.Text)
				}
			}
			result[i] = strings.Join(parts, "")
		}
	}
	return result, nil
}

// xlsxWorksheet 工作表结构
type xlsxWorksheet struct {
	SheetData xlsxSheetData `xml:"sheetData"`
}

type xlsxSheetData struct {
	Rows []xlsxRow `xml:"row"`
}

type xlsxRow struct {
	Cells []xlsxCell `xml:"c"`
}

type xlsxCell struct {
	Ref       string           `xml:"r,attr"`        // 单元格引用如 "A1"
	Type      string           `xml:"t,attr"`        // "s"=共享字符串, "n"/""=数字, "inlineStr"=内联
	Value     string           `xml:"v"`             // 值
	InlineStr xlsxInlineString `xml:"is"`            // 内联字符串
}

type xlsxInlineString struct {
	Text string `xml:"t"`
}

// parseXlsxSheet 解析工作表 XML，将单元格映射为文本
func parseXlsxSheet(data []byte, sharedStrings []string) string {
	var ws xlsxWorksheet
	if err := xml.Unmarshal(data, &ws); err != nil {
		return ""
	}

	var rows []string
	for _, row := range ws.SheetData.Rows {
		var cells []string
		for _, cell := range row.Cells {
			text := cellValueToString(cell, sharedStrings)
			cells = append(cells, text)
		}
		rows = append(rows, strings.Join(cells, "\t"))
	}
	return strings.Join(rows, "\n")
}

// cellValueToString 将单元格值转换为文本
func cellValueToString(cell xlsxCell, sharedStrings []string) string {
	switch cell.Type {
	case "s":
		// 共享字符串引用
		idx, err := strconv.Atoi(cell.Value)
		if err != nil || idx < 0 || idx >= len(sharedStrings) {
			return ""
		}
		return sharedStrings[idx]
	case "inlineStr":
		return cell.InlineStr.Text
	default:
		// 数字或空
		return cell.Value
	}
}
