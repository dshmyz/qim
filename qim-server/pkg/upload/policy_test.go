package upload

import (
	"testing"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"正常文件名", "document.pdf", "document.pdf"},
		{"空字符串", "", "unnamed"},
		{"路径遍历", "../../../etc/passwd", "passwd"},
		{"反斜杠路径遍历", "..\\..\\..\\windows\\system32", "system32"},
		// filepath.Clean 会把 / 当路径分隔符取最后一段，再替换特殊字符
		{"特殊字符", "file<>:\"/\\|?*.txt", "___.txt"},
		{"控制字符", "file\x00\x01\x02.txt", "file___.txt"},
		{"以点开头", ".hidden", "hidden"},
		{"只有点", ".", "unnamed"},
		{"两个点", "..", "unnamed"},
		{"超长文件名", string(make([]byte, 300)) + ".txt", ""},
		{"中文文件名", "测试文件.pdf", "测试文件.pdf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 超长文件名只检查长度不超过 200
			if tt.name == "超长文件名" {
				result := SanitizeFilename(tt.input)
				if len(result) > 200 {
					t.Errorf("超长文件名未截断: got len=%d", len(result))
				}
				return
			}
			result := SanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDetectMimeType(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{"空数据", []byte{}, "application/octet-stream"},
		{"PNG 文件头", []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, "image/png"},
		{"JPEG 文件头", []byte{0xFF, 0xD8, 0xFF}, "image/jpeg"},
		{"PDF 文件头", []byte("%PDF-1.4"), "application/pdf"},
		{"GIF 文件头", []byte("GIF89a"), "image/gif"},
		{"HTML 内容", []byte("<html><body>test</body></html>"), "text/html"},
		{"纯文本", []byte("hello world"), "text/plain"},
		{"超过512字节取前512", append(make([]byte, 600), []byte{0x89, 0x50, 0x4E, 0x47}...), ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectMimeType(tt.data)
			if tt.expected == "" {
				// 超长数据只验证不 panic
				return
			}
			// http.DetectContentType 可能带 charset 参数，只检查前缀
			if len(result) < len(tt.expected) || result[:len(tt.expected)] != tt.expected {
				t.Errorf("DetectMimeType() = %q, want prefix %q", result, tt.expected)
			}
		})
	}
}

func TestPolicy_ValidateSize(t *testing.T) {
	policy := NewPolicy(1024, nil, false)

	tests := []struct {
		name    string
		size    int64
		wantErr bool
	}{
		{"小于限制", 512, false},
		{"等于限制", 1024, false},
		{"超过限制", 2048, true},
		{"零大小", 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := policy.ValidateSize(tt.size)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateSize(%d) error = %v, wantErr %v", tt.size, err, tt.wantErr)
			}
		})
	}
}

func TestPolicy_ValidateType(t *testing.T) {
	// 开启白名单校验的策略
	policy := NewPolicy(0, map[string]bool{
		".pdf": true, ".png": true, ".txt": true,
	}, true)

	tests := []struct {
		name     string
		filename string
		mime     string
		wantErr  bool
	}{
		{"白名单内的PDF", "doc.pdf", "application/pdf", false},
		{"白名单内的PNG", "img.png", "image/png", false},
		{"白名单外的DOCX", "doc.docx", "application/octet-stream", true},
		{"白名单外的HTML", "evil.html", "text/html", true},
		{"白名单外的EXE", "malware.exe", "application/x-msdownload", true},
		{"白名单外的SVG", "icon.svg", "image/svg+xml", true},
		{"危险MIME但安全扩展名", "doc.pdf", "text/html", true},
		{"无扩展名", "noext", "application/octet-stream", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := policy.ValidateType(tt.filename, tt.mime)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateType(%q, %q) error = %v, wantErr %v",
					tt.filename, tt.mime, err, tt.wantErr)
			}
		})
	}
}

func TestPolicy_ValidateType_AllExtensionsAllowed(t *testing.T) {
	// 上传阶段已放开所有扩展名（含可执行/可渲染）：关闭白名单校验时，任何扩展名都应放行
	policy := NewPolicy(0, nil, false)

	allowedFiles := []string{"evil.html", "malware.exe", "script.js", "shell.sh", "icon.svg", "doc.pdf"}
	for _, f := range allowedFiles {
		if err := policy.ValidateType(f, "application/octet-stream"); err != nil {
			t.Errorf("上传应放行: ValidateType(%q) = %v, want nil", f, err)
		}
	}

	// 但"会被内联渲染"的扩展名（非 blockedExtensions）遇到危险 MIME 仍应拦截，防止伪装扩展名绕过
	// 例如 .pdf 内容实为 HTML：扩展名不在渲染黑名单，会内联渲染，故 MIME 兜底必须生效
	if err := policy.ValidateType("doc.pdf", "text/html"); err != ErrFileTypeBlocked {
		t.Errorf("危险MIME应拦截: ValidateType(\"doc.pdf\", \"text/html\") = %v, want ErrFileTypeBlocked", err)
	}
}

func TestShouldForceDownload(t *testing.T) {
	// 即使所有扩展名均可上传，服务端出文件时对这些可渲染/可执行类型仍强制下载，防存储型 XSS
	for _, f := range []string{"evil.html", "icon.svg", "script.js", "malware.exe", "shell.sh"} {
		if !ShouldForceDownload(f) {
			t.Errorf("ShouldForceDownload(%q) = false, want true", f)
		}
	}
	// 普通文档不应强制下载
	if ShouldForceDownload("doc.pdf") {
		t.Error("ShouldForceDownload(\"doc.pdf\") = true, want false")
	}
}

func TestIsUploadEnabled(t *testing.T) {
	tests := []struct {
		name      string
		enableFunc func() (string, error)
		want      bool
	}{
		{"nil函数默认允许", nil, true},
		{"返回true", func() (string, error) { return "true", nil }, true},
		{"返回false", func() (string, error) { return "false", nil }, false},
		{"返回空字符串", func() (string, error) { return "", nil }, true},
		{"返回错误默认允许", func() (string, error) { return "", errTest }, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsUploadEnabled(tt.enableFunc)
			if got != tt.want {
				t.Errorf("IsUploadEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

var errTest = errSentinel{}

type errSentinel struct{}

func (e errSentinel) Error() string { return "test error" }
