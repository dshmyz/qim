package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"qim-server/config"
)

type LocalStorage struct {
	baseDir string
	absBase string
}

func NewLocalStorage(cfg config.LocalStorageConfig) (*LocalStorage, error) {
	baseDir := cfg.Path
	if baseDir == "" {
		baseDir = "./uploads"
	}
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, err
	}
	return &LocalStorage{baseDir: baseDir, absBase: absBase}, nil
}

func (l *LocalStorage) resolvePath(key string) (string, error) {
	cleaned := filepath.Clean("/" + key)
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "" || cleaned == "." {
		return "", fmt.Errorf("非法路径: %s", key)
	}
	if strings.Contains(cleaned, "..") {
		return "", fmt.Errorf("非法路径: %s", key)
	}
	path := filepath.Join(l.baseDir, cleaned)
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(absPath, l.absBase+string(filepath.Separator)) && absPath != l.absBase {
		return "", fmt.Errorf("路径越界: %s", key)
	}
	return path, nil
}

func (l *LocalStorage) Put(ctx context.Context, key string, data io.Reader, size int64, mime string) error {
	path, err := l.resolvePath(key)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, data)
	return err
}

func (l *LocalStorage) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	path, err := l.resolvePath(key)
	if err != nil {
		return nil, err
	}
	return os.Open(path)
}

func (l *LocalStorage) Delete(ctx context.Context, key string) error {
	path, err := l.resolvePath(key)
	if err != nil {
		return err
	}
	return os.Remove(path)
}

func (l *LocalStorage) Exists(ctx context.Context, key string) (bool, error) {
	path, err := l.resolvePath(key)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (l *LocalStorage) URL(ctx context.Context, key string, expires time.Duration) (string, error) {
	return "/" + key, nil
}

func (l *LocalStorage) Kind() string {
	return "local"
}
