package di

import (
	"context"
	"fmt"
	"io"

	"github.com/dshmyz/qim/qim-server/service"
	"github.com/dshmyz/qim/qim-server/service/storage"
)

// storageAccessor 包装 storage.Manager，实现 service.StorageAccessor 接口。
// 放在 di 包以同时引用 storage 和 service 两个包，打破 service↔storage 循环依赖。
type storageAccessor struct {
	mgr *storage.Manager
}

// NewStorageAccessor 用 storage.Manager 构造一个 service.StorageAccessor 实现。
func NewStorageAccessor(mgr *storage.Manager) service.StorageAccessor {
	return &storageAccessor{mgr: mgr}
}

func (a *storageAccessor) GetByPath(ctx context.Context, storagePath string) (io.ReadCloser, error) {
	st, key, ok := a.mgr.ByPath(storagePath)
	if !ok || st == nil {
		return nil, fmt.Errorf("存储类型不支持: %s", storagePath)
	}
	return st.Get(ctx, key)
}

func (a *storageAccessor) Put(ctx context.Context, key string, data io.Reader, size int64, mime string) (string, error) {
	st := a.mgr.Default()
	if st == nil {
		return "", fmt.Errorf("存储服务未初始化")
	}
	if err := st.Put(ctx, key, data, size, mime); err != nil {
		return "", err
	}
	return storage.BuildPath(st.Kind(), key), nil
}

func (a *storageAccessor) DeleteByPath(ctx context.Context, storagePath string) error {
	st, key, ok := a.mgr.ByPath(storagePath)
	if !ok || st == nil {
		return fmt.Errorf("存储类型不支持: %s", storagePath)
	}
	return st.Delete(ctx, key)
}

func (a *storageAccessor) Kind() string {
	st := a.mgr.Default()
	if st == nil {
		return ""
	}
	return st.Kind()
}
