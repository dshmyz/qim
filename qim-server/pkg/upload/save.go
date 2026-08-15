package upload

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"
)

// StorageBackend 抽象存储后端，避免 upload 包依赖 storage 包造成循环依赖。
type StorageBackend interface {
	Put(ctx context.Context, key string, data io.Reader, size int64, mime string) error
	Delete(ctx context.Context, key string) error
	Kind() string
}

// SavedFile 保存结果。
type SavedFile struct {
	StoragePath string // 完整存储路径（含 /static/ 前缀）
	Key         string // 存储 key（不含前缀）
	MimeType    string // 服务端检测出的真实 MIME
	Size        int64  // 实际读取的字节数
	SafeName    string // 清洗后的原始文件名
}

// SaveResult 保存配置。
type SaveConfig struct {
	Policy       *Policy                                      // 上传策略（大小/类型校验）
	Storage      StorageBackend                               // 存储后端
	KeyPrefix    string                                       // 存储 key 前缀，如 "uploads/feedbacks/2026/01/"
	FilenameFn   func() string                                // 生成最终存储文件名（不含目录），若为 nil 则用 SafeName
	MaxBytesRead int64                                        // 最多读取的字节数（用于限制内存），0 表示用 Policy.MaxSize
	ContextFn    func() (context.Context, context.CancelFunc) // 自定义 context，nil 用默认 30s
	// SkipTypeCheck 跳过类型/扩展名校验（黑名单 + 白名单），仅用于受信任的管理员分发来源
	// （source = version / client_update，发布 CLI、MCP、客户端安装包等二进制产物）。
	// 大小限制与文件名清洗仍然生效。默认 false，不影响普通上传的安全校验。
	SkipTypeCheck bool
}

// SaveMultipartFile 公共的"读取+校验+存储"函数。
// 流程：校验大小 → 读取文件到内存（受 MaxBytesRead 限制）→ 检测 MIME → 校验类型 → 存储上传。
// 返回 SavedFile，调用方负责创建数据库记录；若建记录失败应调用 Cleanup 删除已上传文件。
//
// 设计权衡：
//   - 仍读全量到内存：因为 MIME 检测需要文件头，类型校验需要完整内容，且需要 size 用于 Content-Length。
//     若改纯流式，需放弃服务端 MIME 检测（信任客户端）或先写临时文件再读头（增加 IO）。
//   - 对内部 IM 场景，单文件 500MB 内存可控，且 MaxBytesReader 在网络层已限制请求体大小。
//   - 真正的大文件场景走分片上传（ChunkService），不走本函数。
func SaveMultipartFile(header *multipart.FileHeader, cfg SaveConfig) (*SavedFile, error) {
	if cfg.Policy == nil {
		return nil, errors.New("upload: Policy 不能为空")
	}
	if cfg.Storage == nil {
		return nil, errors.New("upload: Storage 不能为空")
	}

	// 1. 校验大小（客户端声明）
	if err := cfg.Policy.ValidateSize(header.Size); err != nil {
		return nil, err
	}

	// 2. 读取文件到内存
	fileData, err := header.Open()
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer fileData.Close()

	maxRead := cfg.MaxBytesRead
	if maxRead <= 0 {
		maxRead = cfg.Policy.MaxSize
	}
	fileBytes, err := io.ReadAll(io.LimitReader(fileData, maxRead+1))
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}
	if int64(len(fileBytes)) > maxRead {
		return nil, ErrFileTooLarge
	}

	// 3. 检测真实 MIME（不信任客户端 Content-Type）
	detectedMime := DetectMimeType(fileBytes)

	// 4. 校验类型（黑名单 + 可选白名单）；受信任的管理员分发来源跳过（如 .exe 安装包/CLI 二进制）
	if !cfg.SkipTypeCheck {
		if err := cfg.Policy.ValidateType(header.Filename, detectedMime); err != nil {
			return nil, err
		}
	}

	// 5. 清洗文件名
	safeName := SanitizeFilename(header.Filename)

	// 6. 生成存储 key
	filename := safeName
	if cfg.FilenameFn != nil {
		filename = cfg.FilenameFn()
	}
	key := cfg.KeyPrefix + filename
	if cfg.KeyPrefix != "" && !strings.HasSuffix(cfg.KeyPrefix, "/") {
		key = cfg.KeyPrefix + "/" + filename
	}

	// 7. 存储上传
	ctx, cancel := buildContext(cfg.ContextFn)
	defer cancel()

	size := int64(len(fileBytes))
	if err := cfg.Storage.Put(ctx, key, bytes.NewReader(fileBytes), size, detectedMime); err != nil {
		return nil, fmt.Errorf("存储文件失败: %w", err)
	}

	return &SavedFile{
		StoragePath: buildStoragePath(cfg.Storage.Kind(), key),
		Key:         key,
		MimeType:    detectedMime,
		Size:        size,
		SafeName:    safeName,
	}, nil
}

// buildStoragePath 生成存储路径，与 storage.BuildPath 保持一致。
// 这里不直接 import storage 包（避免循环依赖），而是用同样的规则。
func buildStoragePath(kind, key string) string {
	return "/static/" + key
}

// buildContext 构造存储操作的 context。
func buildContext(fn func() (context.Context, context.CancelFunc)) (context.Context, context.CancelFunc) {
	if fn != nil {
		return fn()
	}
	return context.WithTimeout(context.Background(), 30*time.Second)
}

// Cleanup 删除已上传的存储文件（建记录失败时调用，避免孤儿文件）。
func (s *SavedFile) Cleanup(storage StorageBackend) {
	if s == nil || storage == nil || s.Key == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = storage.Delete(ctx, s.Key)
}

// Ext 返回清洗后文件名的扩展名（小写）。
func (s *SavedFile) Ext() string {
	return strings.ToLower(filepath.Ext(s.SafeName))
}
