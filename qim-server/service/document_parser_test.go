package service

import (
	"os"
	"path/filepath"
	"testing"
)

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
