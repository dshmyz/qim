package service

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

// fakeAnydocBackend 测试用的 anydoc 增强后端：可控制可用性、输出与错误。
type fakeAnydocBackend struct {
	avail   bool
	output  string
	convertErr error
	// 记录最后一次调用的文件路径，便于断言走了 anydoc 分支
	lastCalls []string
}

func (f *fakeAnydocBackend) Available() bool { return f.avail }

func (f *fakeAnydocBackend) Convert(filePath string) (string, error) {
	f.lastCalls = append(f.lastCalls, filePath)
	return f.output, f.convertErr
}

// disabledAnydoc 返回一个始终不可用的后端，用于明确「关闭 anydoc」的确定性测试。
func disabledAnydoc() *fakeAnydocBackend {
	return &fakeAnydocBackend{avail: false}
}

// TestTrimPDFTrailingData 验证对尾部带附加数据（知网/万方论文常见，%%EOF 之后仍残留
// 元数据尾块）的 PDF 进行截断，使其能以 %%EOF 结尾，从而通过 ledongthuc/pdf 的
// HasSuffix 校验；同时确认对正常 PDF 与不含 %%EOF 的文件不做破坏性改动。
func TestTrimPDFTrailingData(t *testing.T) {
	p := NewDocumentParser()

	t.Run("strip trailing bytes after last %%EOF", func(t *testing.T) {
		content := []byte("PDFCONTENT\r\nstartxref\r\n123\r\n%%EOF\r\nWebFastLoad<FileProperty><Doi>x</Doi></FileProperty>")
		path := writeTemp(t, content)
		defer os.Remove(path)

		if err := p.trimPDFTrailingData(path); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, _ := os.ReadFile(path)
		want := "PDFCONTENT\r\nstartxref\r\n123\r\n%%EOF\r\n"
		if string(got) != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	})

	t.Run("leave file already ending with %%EOF unchanged", func(t *testing.T) {
		content := []byte("PDFCONTENT\r\nstartxref\r\n123\r\n%%EOF")
		path := writeTemp(t, content)
		defer os.Remove(path)

		if err := p.trimPDFTrailingData(path); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, _ := os.ReadFile(path)
		if string(got) != string(content) {
			t.Fatalf("file was modified: got %q, want %q", got, content)
		}
	})

	t.Run("no %%EOF passes through without modification", func(t *testing.T) {
		content := []byte("this is not really a pdf but let's be tolerant")
		path := writeTemp(t, content)
		defer os.Remove(path)

		if err := p.trimPDFTrailingData(path); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, _ := os.ReadFile(path)
		if string(got) != string(content) {
			t.Fatalf("file was modified: got %q, want %q", got, content)
		}
	})
}

func writeTemp(t *testing.T, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "sample.pdf")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}

// TestParse_RejectsBinaryAsText 验证 Parse 对二进制/未知格式的防御：
//  1. 未支持的扩展名（如 .doc 二进制老格式）直接被拒绝，不再按纯文本读取。
//  2. 文本类文件若内含非法 UTF-8 字节（如真二进制混入），也会被拦截。
//  3. 合法 UTF-8 的文本文件正常解析。
func TestParse_RejectsBinaryAsText(t *testing.T) {
	p := NewDocumentParser()
	// 注入「不可用」的 anydoc，确保 .doc 老格式走测试不依赖本机是否装了 anydoc：
	// 关闭增强后 .doc 稳定落入 default 拒绝分支。
	p.SetAnydoc(disabledAnydoc())

	t.Run("unsupported extension (.doc) rejected", func(t *testing.T) {
		binary := []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}
		path := filepath.Join(t.TempDir(), "legacy.doc")
		if err := os.WriteFile(path, binary, 0o644); err != nil {
			t.Fatalf("write temp .doc: %v", err)
		}
		defer os.Remove(path)

		_, err := p.Parse(path)
		if err == nil {
			t.Fatal("Parse should reject unsupported .doc extension, got nil error")
		}
	})

	t.Run("text ext with binary bytes rejected", func(t *testing.T) {
		// .txt 是允许的扩展名，但内容是非 UTF-8 二进制（如误把二进制文件改名成 .txt）。
		// 用 0xFF 等无合法 UTF-8 解释的字节序列，确保会被 utf8.ValidString 拦截。
		binary := []byte("prefix\xff\xfe\x00\x80suffix")
		path := filepath.Join(t.TempDir(), "bad.txt")
		if err := os.WriteFile(path, binary, 0o644); err != nil {
			t.Fatalf("write temp .txt: %v", err)
		}
		defer os.Remove(path)

		_, err := p.Parse(path)
		if err == nil {
			t.Fatal("Parse should reject non-UTF-8 content even with .txt extension, got nil error")
		}
	})

	t.Run("valid utf-8 text parses fine", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "note.md")
		content := "### 需求\n\n这是一个正常的 UTF-8 中文文档，图谱应正确展示。"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write temp .md: %v", err)
		}
		defer os.Remove(path)

		got, err := p.Parse(path)
		if err != nil {
			t.Fatalf("Parse should accept valid UTF-8 text, got error: %v", err)
		}
		if got != content {
			t.Fatalf("Parse output mismatch:\n got %q\nwant %q", got, content)
		}
	})
}

// TestParse_AnydocEnhancement 验证 anydoc 增强分支的行为：
//  1. 对老格式（.doc/.xls 等）anydoc 可用时优先转，输出其 Markdown；
//  2. anydoc 失败（不可转换）时，原生支持者回退原生解析；
//  3. anydoc 失败时，原生不支持者（老格式）维持「不支持」拒绝；
//  4. anydoc 可用时，原生支持格式（docx/pdf）也走 anydoc；
//  5. anydoc 不可用时，老格式走 default 拒绝（与关闭增强一致）。
func TestParse_AnydocEnhancement(t *testing.T) {
	// 5: anydoc 不可用时老格式拒绝
	t.Run("legacy ext rejected when anydoc unavailable", func(t *testing.T) {
		p := NewDocumentParser()
		p.SetAnydoc(disabledAnydoc())
		path := filepath.Join(t.TempDir(), "legacy.xls")
		if err := os.WriteFile(path, []byte("binary"), 0o644); err != nil {
			t.Fatalf("write temp: %v", err)
		}
		defer os.Remove(path)

		if _, err := p.Parse(path); err == nil {
			t.Fatal("legacy .xls should be rejected when anydoc unavailable")
		}
	})

	// 1: anydoc 可用时老格式优先转
	t.Run("legacy ext converted by anydoc when available", func(t *testing.T) {
		p := NewDocumentParser()
		fake := &fakeAnydocBackend{avail: true, output: "# 老文档\n\n内容"}
		p.SetAnydoc(fake)

		path := filepath.Join(t.TempDir(), "legacy.xls")
		if err := os.WriteFile(path, []byte("anything"), 0o644); err != nil {
			t.Fatalf("write temp: %v", err)
		}
		defer os.Remove(path)

		got, err := p.Parse(path)
		if err != nil {
			t.Fatalf("Parse should succeed via anydoc: %v", err)
		}
		if got != "# 老文档\n\n内容" {
			t.Fatalf("got %q, want anydoc markdown", got)
		}
		if len(fake.lastCalls) != 1 {
			t.Fatalf("anydoc should be called once, got %d", len(fake.lastCalls))
		}
	})

	// 3: anydoc 失败时老格式维持「不支持」
	t.Run("legacy ext stays rejected when anydoc fails", func(t *testing.T) {
		p := NewDocumentParser()
		p.SetAnydoc(&fakeAnydocBackend{avail: true, convertErr: errAnydocUnsupported})

		path := filepath.Join(t.TempDir(), "legacy.xls")
		if err := os.WriteFile(path, []byte("anything"), 0o644); err != nil {
			t.Fatalf("write temp: %v", err)
		}
		defer os.Remove(path)

		if _, err := p.Parse(path); err == nil {
			t.Fatal("legacy .xls should be rejected when anydoc cannot convert it")
		}
	})

	// 2: anydoc 失败时原生格式 docx 回退原生解析
	t.Run("native ext falls back to native parser when anydoc fails", func(t *testing.T) {
		p := NewDocumentParser()
		p.SetAnydoc(&fakeAnydocBackend{avail: true, convertErr: errAnydocUnsupported})

		// 构造一个最简 docx（仅含 word/document.xml + 一个文本段），原生解析器可读。
		path := filepath.Join(t.TempDir(), "sample.docx")
		writeMinimalDocx(t, path)
		defer os.Remove(path)

		got, err := p.Parse(path)
		if err != nil {
			t.Fatalf("native fallback should parse docx: %v", err)
		}
		if got != "来自docx的正文" {
			t.Fatalf("native fallback got %q, want document text", got)
		}
	})

	// 4: anydoc 可用时原生格式 docx 也走 anydoc
	t.Run("native ext converted by anydoc when available", func(t *testing.T) {
		p := NewDocumentParser()
		fake := &fakeAnydocBackend{avail: true, output: "anydoc-提取的docx内容"}
		p.SetAnydoc(fake)

		path := filepath.Join(t.TempDir(), "sample.docx")
		writeMinimalDocx(t, path)
		defer os.Remove(path)

		got, err := p.Parse(path)
		if err != nil {
			t.Fatalf("Parse should succeed via anydoc: %v", err)
		}
		if got != "anydoc-提取的docx内容" {
			t.Fatalf("got %q, want anydoc output for docx", got)
		}
		if len(fake.lastCalls) != 1 {
			t.Fatalf("anydoc should be called for docx, got %d calls", len(fake.lastCalls))
		}
	})

	// anydoc 不可用时 docx 原生解析正常（不因关闭增强而退化）
	t.Run("native ext parses natively when anydoc unavailable", func(t *testing.T) {
		p := NewDocumentParser()
		p.SetAnydoc(disabledAnydoc())

		path := filepath.Join(t.TempDir(), "sample.docx")
		writeMinimalDocx(t, path)
		defer os.Remove(path)

		got, err := p.Parse(path)
		if err != nil {
			t.Fatalf("native docx parse should succeed: %v", err)
		}
		if got != "来自docx的正文" {
			t.Fatalf("got %q, want document text", got)
		}
	})
}

// writeMinimalDocx 生成一个极简但合法的 docx（zip 内含 word/document.xml，
// 一个段落一个文本），供原生 parseDocx 与 anydoc 分支测试使用。
func writeMinimalDocx(t *testing.T, path string) {
	t.Helper()
	const text = "来自docx的正文"
	body := `<?xml version="1.0" encoding="UTF-8"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body><w:p><w:r><w:t>` + text + `</w:t></w:r></w:p></w:body>
</w:document>`

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create docx: %v", err)
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	w, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatalf("zip create entry: %v", err)
	}
	if _, err := w.Write([]byte(body)); err != nil {
		t.Fatalf("zip write: %v", err)
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
}

