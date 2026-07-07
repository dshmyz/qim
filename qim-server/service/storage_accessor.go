package service

import (
	"context"
	"io"
)

// StorageAccessor 提供存储后端的读写能力，由 DI 层注入实现。
// service 包不直接依赖 storage 包（storage 已依赖 service，反向引用会成循环依赖），
// 故用此接口解耦。
type StorageAccessor interface {
	// GetByPath 按 StoragePath 读取文件内容
	GetByPath(ctx context.Context, storagePath string) (io.ReadCloser, error)
	// Put 上传到默认存储后端，返回 BuildPath 后的 StoragePath
	Put(ctx context.Context, key string, data io.Reader, size int64, mime string) (storagePath string, err error)
	// DeleteByPath 按 StoragePath 删除文件
	DeleteByPath(ctx context.Context, storagePath string) error
	// Kind 返回默认存储后端类型 ("local" / "s3")
	Kind() string
}
