package service

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/ledongthuc/pdf"
)

// maxDocumentSize 文档解析的单文件大小上限（20MB）
// 防止 zip bomb / 超大文件导致 OOM
const maxDocumentSize = 20 * 1024 * 1024

// maxZipEntrySize ZIP 内单个成员的解压读取上限（20MB）
// 防止压缩比极高的恶意 zip bomb 解压后撑爆内存
const maxZipEntrySize = 20 * 1024 * 1024

// anydocBackend 抽象的 anydoc 增强后端，供 DocumentParser 调用。
// 生产实现是 *AnydocConverter（调 anydoc CLI）；测试注入假实现以隔离外部二进制。
type anydocBackend interface {
	// Available 报告后端当前是否可用（二进制存在）。
	Available() bool
	// Convert 把 filePath 指向的文档转成 Markdown 返回。
	Convert(filePath string) (string, error)
}

// DocumentParser 文档内容解析器
type DocumentParser struct {
	// anydoc 可选的 anydoc CLI 增强后端（nil 表示未初始化，Parse 时惰性创建）。
	// anydoc 补上老格式（.doc/.xls/.ppt/.rtf/.odt/.ods/.odp/.epub）与更好提取的
	// PDF，且不需要 LibreOffice。二进制缺失/调用失败时回退到下方原生解析器。
	anydoc   anydocBackend
	anydocMu sync.Mutex // 保护 anydoc 的惰性初始化与 SetAnydoc 注入，防并发 Parse 数据竞争
}

// NewDocumentParser 创建文档解析器实例
func NewDocumentParser() *DocumentParser {
	return &DocumentParser{}
}

// SetAnydoc 注入 anydoc 增强后端（供测试注入假实现/关闭）。传 nil 关闭。
func (p *DocumentParser) SetAnydoc(c anydocBackend) {
	p.anydocMu.Lock()
	defer p.anydocMu.Unlock()
	p.anydoc = c
}

// anydocConverter 惰性获取 anydoc 后端：未设置时按 PATH/env 探测一次。
// 返回非 nil 的不可用实例（Available()==false），调用方自然回退原生解析。
// 加锁防并发 Parse 下的 p.anydoc 读写竞争（DocumentParser 为 DI 单例，多 goroutine 共用）。
func (p *DocumentParser) anydocConverter() anydocBackend {
	p.anydocMu.Lock()
	defer p.anydocMu.Unlock()
	if p.anydoc == nil {
		p.anydoc = NewAnydocConverter()
	}
	return p.anydoc
}

// anydocSupportedExts 交给 anydoc 尝试的扩展名集合。
// 覆盖 DocumentParser 原生支持之外的多数办公格式；anydoc 失败时由 Parse 决定
// 回退（原生支持者）或维持「不支持」（原生不支持者）。
func anydocSupportedExts() map[string]bool {
	return map[string]bool{
		// 原生已支持，anydoc 作为更强提取后端优先使用，失败回退原生
		"pdf": true, "docx": true, "pptx": true, "xlsx": true,
		// 老格式：原生不支持，anydoc 成功即补上，失败维持「不支持」
		"doc": true, "xls": true, "ppt": true, "rtf": true,
		"odt": true, "ods": true, "odp": true, "epub": true,
	}
}

// Parse 根据文件扩展名解析文档内容
func (p *DocumentParser) Parse(filePath string) (string, error) {
	ext := strings.ToLower(filePath[strings.LastIndex(filePath, ".")+1:])

	// anydoc 增强：对命中扩展名优先尝试，成功即返回（含更多格式细节的 Markdown）。
	// 失败时：原生支持者回退到下方 switch 的原生解析；原生不支持者落入 default 的
	// 「不支持」分支。anydoc 不可用（未装二进制）时该分支整体跳过，行为与现状一致。
	if anydocSupportedExts()[ext] {
		if converter := p.anydocConverter(); converter.Available() {
			if text, err := converter.Convert(filePath); err == nil {
				return text, nil
			}
		}
	}

	var text string
	var err error
	switch ext {
	case "txt", "md", "mdx", "markdown", "csv", "json", "log":
		text, err = p.parseText(filePath)
	case "pdf":
		text, err = p.parsePDF(filePath)
	case "docx":
		text, err = p.parseDocx(filePath)
	case "pptx":
		text, err = p.parsePptx(filePath)
	case "xlsx":
		text, err = p.parseXlsx(filePath)
	default:
		// 未知扩展名：直接拒绝，不尝试按纯文本读取。
		// 覆盖项目未支持、且 anydoc 也无法转换的格式（如加密/扫描文档等）。
		// anydoc 可转换的老格式（.doc/.xls/.ppt/.rtf/.odt/.ods/.odp/.epub）会先在
		// 上方 anydoc 分支命中；走到这里说明 either anydoc 未装、或 it 报「不可转换」，
		// 故维持明确拒绝并提示可用格式。
		return "", fmt.Errorf("不支持的文件类型 .%s（支持 txt/md/csv/json/pdf/docx/pptx/xlsx，及 anydoc 可转换的 doc/xls/ppt/rtf/odt/ods/odp/epub）", ext)
	}
	if err != nil {
		return "", err
	}

	// 防御：解析结果不是合法 UTF-8（如扫描 PDF 提取出的原始字节、文本文件内嵌的
	// 二进制内容）会被统一拦截，避免乱码向量落库后被图谱/检索读到。
	if !utf8.ValidString(text) {
		return "", fmt.Errorf("解析结果不是合法 UTF-8 文本（字节数 %d，疑似二进制内容，文件可能损坏或为扫描件）", len(text))
	}
	return text, nil
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
	// 预处理：很多真实 PDF（尤其知网/万方下载的论文）在末尾 %%EOF 之后还残留一段
	// 元数据/扫描尾块（如 WebFastLoad<FileProperty>...</FileProperty>）。而
	// ledongthuc/pdf 只读文件最后 100 字节并要求以 %%EOF 结尾（read.go 的 HasSuffix
	// 校验），尾部带数据就会误报 "not a PDF file: missing %%EOF"。
	// 这里先把文件截断到最后一个 %%EOF 处，再交给 pdf.Open 解析。
	if err := p.trimPDFTrailingData(filePath); err != nil {
		return "", err
	}

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

// trimPDFTrailingData 将 PDF 文件原地截断到最后一个 %%EOF 结束处。
// 截断点取 "%%EOF" 之后到行尾结束（含可能的 \r\n），丢弃其后所有附加数据；
// 若文件本身已以 %%EOF 结尾则不做任何改动。找不到 %%EOF 或文件过大时直接放行，
// 交由 pdf.Open 自己报错，保持原有行为。
func (p *DocumentParser) trimPDFTrailingData(filePath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("获取 PDF 文件信息 %s 失败: %w", filePath, err)
	}
	// 不超过大小上限才做扫描，避免超大文件整读
	if info.Size() > maxDocumentSize || info.Size() == 0 {
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取 PDF 文件 %s 失败: %w", filePath, err)
	}

	idx := bytes.LastIndex(data, []byte("%%EOF"))
	if idx < 0 {
		// 没有 %%EOF：不修改，交给 pdf.Open 判定
		return nil
	}
	// 截断点 = %%EOF 之后到行尾（保留 %%EOF 本身与紧随的换行符）
	truncAt := idx + len("%%EOF")
	for truncAt < len(data) && (data[truncAt] == ' ' || data[truncAt] == '\t') {
		truncAt++
	}
	if truncAt < len(data) && (data[truncAt] == '\r' || data[truncAt] == '\n') {
		if data[truncAt] == '\r' && truncAt+1 < len(data) && data[truncAt+1] == '\n' {
			truncAt++
		}
		truncAt++
	}
	if truncAt >= len(data) {
		// 已经以 %%EOF 结尾，无需改动
		return nil
	}

	if err := os.WriteFile(filePath, data[:truncAt], info.Mode().Perm()); err != nil {
		return fmt.Errorf("截断 PDF 尾部数据失败 %s: %w", filePath, err)
	}
	return nil
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
