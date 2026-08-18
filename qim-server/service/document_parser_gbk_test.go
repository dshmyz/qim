package service

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildGBKType0PDF 程序化构造一份最小 PDF，复刻知网/万方中文论文的字体结构：
//
//	Type0 字体 + /Encoding /GBK-EUC-H（GBK 系具名编码）+ 无 ToUnicode CMap，
//	内容流以原始 GBK 字节写正文（"AB心理"：心理 的 GBK 编码为 D0C4 C0ED）。
//
// 这类 PDF 是 ledongthuc/pdf 原生解析的乱码重灾区：解码器不识别 GBK-EUC-H
// 具名编码时会把 2 字节 GBK 码按裸字节处理，正文变成 U+FFFD 乱码。本 fixture
// 用于钉住仓库内 fork 的解码修复（见 third_party/ledongthuc/pdf/gbk_patch.go）。
// xref 偏移由函数在组装时计算，避免手写偏移随内容改动漂移。
func buildGBKType0PDF(t *testing.T) []byte {
	t.Helper()

	content := "BT /F1 12 Tf (AB\xD0\xC4\xC0\xED) Tj ET" // "AB心理"
	objs := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 6 0 R >>",
		"<< /Type /Font /Subtype /Type0 /BaseFont /TestGBK /Encoding /GBK-EUC-H /DescendantFonts [5 0 R] >>",
		"<< /Type /Font /Subtype /CIDFontType2 /BaseFont /TestGBK /CIDSystemInfo << /Registry (Adobe) /Ordering (GB1) /Supplement 0 >> /FontDescriptor 7 0 R /DW 1000 /CIDToGIDMap /Identity >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content),
		"<< /Type /FontDescriptor /FontName /TestGBK /Flags 4 /FontBBox [0 0 1000 1000] /ItalicAngle 0 /Ascent 800 /Descent -200 /CapHeight 700 /StemV 80 >>",
	}

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objs))
	for i, body := range objs {
		offsets[i] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, body)
	}
	xrefAt := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n", len(objs)+1)
	buf.WriteString("0000000000 65535 f \n")
	for _, off := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", off)
	}
	buf.WriteString("trailer\n")
	fmt.Fprintf(&buf, "<< /Size %d /Root 1 0 R >>\n", len(objs)+1)
	fmt.Fprintf(&buf, "startxref\n%d\n%%%%EOF\n", xrefAt)
	return buf.Bytes()
}

// TestParsePDF_GBKType0Font_DecodesChinese 验证对「Type0 字体 + GBK-EUC-H 具名编码
// + 无 ToUnicode」的知网/万方类 PDF，原生解析能按 GBK 正确还原中文，而非输出
// U+FFFD 乱码。该场景是 ledongthuc/pdf 原生解析的历史乱码根因（getEncoder 对
// GBK-EUC-H 等具名编码不识别），修复见 third_party/ledongthuc/pdf/gbk_patch.go。
func TestParsePDF_GBKType0Font_DecodesChinese(t *testing.T) {
	p := NewDocumentParser()
	// 关闭 anydoc，确保走原生解析路径，钉住 fork 内的解码修复
	p.SetAnydoc(disabledAnydoc())

	path := writeTemp(t, buildGBKType0PDF(t))
	defer os.Remove(path)

	got, err := p.Parse(path)
	if err != nil {
		t.Fatalf("Parse should succeed on GBK Type0 PDF, got error: %v", err)
	}
	t.Logf("extracted: %q", got)

	if !strings.Contains(got, "AB心理") {
		t.Fatalf("expected GBK bytes decoded to Chinese 心理, got: %q", got)
	}
	if strings.ContainsRune(got, '�') {
		t.Fatalf("extraction contains U+FFFD mojibake: %q", got)
	}
}

// TestParse_RejectsMojibakeText 验证乱码守卫：解析结果即使合法 UTF-8，只要 U+FFFD
// 替换符占比超过阈值（解码失败的特征），Parse 也拦截而非放行。乱码不只表现为
// 「非法 UTF-8」——解码失败的解析器产出的替换符本身是合法 UTF-8，能通过字节
// 校验，必须按占比再拦一道。
func TestParse_RejectsMojibakeText(t *testing.T) {
	p := NewDocumentParser()
	p.SetAnydoc(disabledAnydoc())

	t.Run("U+FFFD dominated text rejected", func(t *testing.T) {
		// 合法 UTF-8（替换符编码即 EF BF BD），但占比远超 1% 阈值
		mojibake := strings.Repeat("�", 50) + "正常内容"
		path := filepath.Join(t.TempDir(), "garbled.txt")
		if err := os.WriteFile(path, []byte(mojibake), 0o644); err != nil {
			t.Fatalf("write temp txt: %v", err)
		}
		defer os.Remove(path)

		_, err := p.Parse(path)
		if err == nil {
			t.Fatal("Parse should reject U+FFFD dominated text, got nil error")
		}
		if !strings.Contains(err.Error(), "乱码") {
			t.Fatalf("error should mention mojibake, got: %q", err)
		}
	})

	t.Run("normal text with zero U+FFFD passes", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "ok.txt")
		content := "这是一个完全正常的文档，没有任何解码问题。"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write temp txt: %v", err)
		}
		defer os.Remove(path)

		got, err := p.Parse(path)
		if err != nil {
			t.Fatalf("Parse should accept clean text, got error: %v", err)
		}
		if got != content {
			t.Fatalf("Parse returned %q, want %q", got, content)
		}
	})

	t.Run("sparse U+FFFD below threshold tolerated", func(t *testing.T) {
		// 阈值 1%：极少量替换符（如个别符号转换失败）不应误杀整个文档
		content := strings.Repeat("正常文本", 200) + "�"
		path := filepath.Join(t.TempDir(), "sparse.txt")
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write temp txt: %v", err)
		}
		defer os.Remove(path)

		if _, err := p.Parse(path); err != nil {
			t.Fatalf("Parse should tolerate sparse U+FFFD below threshold, got error: %v", err)
		}
	})
}

// TestUtf8ReplacementRatio 直接钉住占比计算函数。
func TestUtf8ReplacementRatio(t *testing.T) {
	tests := []struct {
		name string
		s    string
		want float64
	}{
		{"empty", "", 0},
		{"no replacement", "正常文本 abc", 0},
		{"all replacement", "���", 1},
		{"half replacement", "��ab", 0.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := utf8ReplacementRatio(tt.s); got != tt.want {
				t.Fatalf("utf8ReplacementRatio(%q) = %v, want %v", tt.s, got, tt.want)
			}
		})
	}
}
