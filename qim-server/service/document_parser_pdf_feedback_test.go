package service

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// minimalScannedLikePDF 构造一个合法但「无文字层」的极简单页 PDF：
// 页面 Content 是空流（无任何文本操作符），ledongthuc/pdf 能打开页面、
// 返回 NumPage()=1，但 GetPlainText 提取不到任何文字——精确模拟扫描件/图片型 PDF。
// 该字节序列经 python 生成、xref 偏移与 startxref 已校准，可用 pdf.Open 打开。
func minimalScannedLikePDF() []byte {
	return []byte(`%PDF-1.4
1 0 obj
<< /Type /Catalog /Pages 2 0 R >>
endobj
2 0 obj
<< /Type /Pages /Kids [3 0 R] /Count 1 >>
endobj
3 0 obj
<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R >>
endobj
4 0 obj
<< /Length 0 >>
stream

endstream
endobj
xref
0 5
0000000000 65535 f
0000000009 00000 n
0000000058 00000 n
0000000115 00000 n
0000000202 00000 n
trailer
<< /Size 5 /Root 1 0 R >>
startxref
251
%%EOF
`)
}

// TestParsePDF_NoTextLayer_YieldsScanHint 验证 Parse 对无文字层（扫描/图片型）PDF
// 返回的错误包含明确的「扫描/无文字层」提示，而非笼统的「格式不支持」。
// 背景：用户上传图片型 PDF 时，parsePDF 提不出文字，若上层把它当成「格式不支持」
// 会误导用户以为是格式问题。此测试钉住错误文案，防止回归成无用提示。
func TestParsePDF_NoTextLayer_YieldsScanHint(t *testing.T) {
	p := NewDocumentParser()
	// 关闭 anydoc，确保走原生解析路径
	p.SetAnydoc(disabledAnydoc())

	path := writeTemp(t, minimalScannedLikePDF())
	defer os.Remove(path)

	_, err := p.Parse(path)
	if err == nil {
		t.Fatal("Parse should error on scanned-like PDF (no text layer), got nil")
	}

	msg := err.Error()
	t.Logf("parse error: %s", msg)

	// 语义正确：错误应指明 PDF 能打开但无文字，且暗示可能是扫描件；绝不能误导为「不支持 pdf 格式」
	if strings.Contains(msg, "不支持") {
		t.Fatalf("error misleads user as unsupported format: %q", msg)
	}
	if !strings.Contains(msg, "扫描") && !strings.Contains(msg, "文字层") {
		t.Fatalf("error should hint scanned / no-text-layer PDF, got: %q", msg)
	}
	if !strings.Contains(msg, "pdf") && !strings.Contains(msg, "PDF") {
		t.Fatalf("error should reference PDF, got: %q", msg)
	}
}

// TestParsePDF_AnydocFails_FallsBackToNative 验证扫描/图片型 PDF 在 anydoc 增强后端
// 明确失败（errAnydocUnsupported，即 anydoc 也识别出它是不可转换文档）时，会回退到
// 原生 parsePDF，而不是把 anydoc 的错误或「格式不支持」抛给用户。即：anydoc 失败
// 不阻塞 PDF 的原生解析路径（降级链路）。由于是扫描件，最终仍以原生「无文字层」报错。
func TestParsePDF_AnydocFails_FallsBackToNative(t *testing.T) {
	p := NewDocumentParser()
	// anydoc 可用但明确报「无法转换该文档」
	p.SetAnydoc(&fakeAnydocBackend{avail: true, convertErr: errAnydocUnsupported})

	path := writeTemp(t, minimalScannedLikePDF())
	defer os.Remove(path)

	_, err := p.Parse(path)
	if err == nil {
		t.Fatal("Parse should still fail for scanned PDF (no text layer), got nil")
	}

	msg := err.Error()
	t.Logf("parse error (anydoc failed): %s", msg)

	// 降级确已发生：错误来自原生 parsePDF 的「无文字层」提示，
	// 而不是 anydoc 的 errAnydocUnsupported 文案，更不是「不支持 pdf 格式」。
	if strings.Contains(msg, "anydoc") {
		t.Fatalf("error came from anydoc, native fallback did NOT happen: %q", msg)
	}
	if strings.Contains(msg, "不支持的文件类型") {
		t.Fatalf("error misleading as unsupported format: %q", msg)
	}
	if !strings.Contains(msg, "扫描") && !strings.Contains(msg, "无文字层") {
		t.Fatalf("expected native parsePDF no-text-layer error after fallback, got: %q", msg)
	}
}

// TestDescribeParseError 验证 describeParseError 会把 Parse 返回的错误整理成对用户
// 友好的一条提示：剥掉内嵌的服务端临时文件绝对路径（避免暴露 /var/folders/...），
// 同时保留「扫描件/无文字层」「格式不支持」等真实语义。
func TestDescribeParseError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		wantContains []string
		wantNot []string
	}{
		{
			name: "scanned pdf strips temp abs path, keeps scan hint",
			err:  fmt.Errorf("PDF /var/folders/zl/abc/T/qim-doc-123.pdf 无法提取文本内容：该文件可能是扫描件/图片型 PDF（无文字层）"),
			wantContains: []string{"扫描", "无法提取文本内容"},
			wantNot:      []string{"/var/folders", "/T/"}, // 不暴露服务端临时目录结构
		},
		{
			name: "unsupported ext passes through",
			err:  fmt.Errorf("不支持的文件类型 .doc（支持 txt/md/csv/json/pdf/docx/pptx/xlsx 等）"),
			wantContains: []string{"不支持的文件类型", ".doc"},
			wantNot:      []string{"/var/", "/tmp/"}, // 无路径可剥，原样透传
		},
		{
			name: "open failure strips path",
			err:  fmt.Errorf("打开 PDF /tmp/qim-456.pdf 失败: some reason"),
			wantContains: []string{"打开 PDF", "失败"},
			wantNot:      []string{"/tmp/"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := describeParseError(tt.err)
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Fatalf("describeParseError(%q) = %q, want contain %q", tt.err.Error(), got, want)
				}
			}
			for _, not := range tt.wantNot {
				if strings.Contains(got, not) {
					t.Fatalf("describeParseError(%q) = %q, should NOT contain %q", tt.err.Error(), got, not)
				}
			}
		})
	}
}
