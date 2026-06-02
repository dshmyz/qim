package storage

import (
	"context"
	"fmt"
	"io"
	"time"

	"qim-server/config"
	"qim-server/service"
)

type Storage interface {
	Put(ctx context.Context, key string, data io.Reader, size int64, mime string) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	URL(ctx context.Context, key string, expires time.Duration) (string, error)
	Kind() string
}

type Manager struct {
	defaultStorage  Storage
	implementations map[string]Storage
}

func NewManager(defaultStorage Storage, others ...Storage) *Manager {
	m := &Manager{
		defaultStorage:  defaultStorage,
		implementations: make(map[string]Storage),
	}
	for _, s := range others {
		if s != nil {
			m.implementations[s.Kind()] = s
		}
	}
	if defaultStorage != nil {
		m.implementations[defaultStorage.Kind()] = defaultStorage
	}
	return m
}

func (m *Manager) Default() Storage {
	return m.defaultStorage
}

func (m *Manager) ByKind(kind string) (Storage, bool) {
	s, ok := m.implementations[kind]
	return s, ok
}

func (m *Manager) ByPath(storagePath string) (Storage, string, bool) {
	kind, key := ParsePath(storagePath)
	s, ok := m.implementations[kind]
	return s, key, ok
}

func NewFromConfig(cfg config.StorageConfig) (Storage, error) {
	switch cfg.Type {
	case "local":
		return NewLocalStorage(cfg.Local)
	case "s3":
		s3Svc, err := service.NewS3Service(cfg.S3)
		if err != nil {
			return nil, err
		}
		return NewS3Storage(s3Svc, cfg.S3), nil
	default:
		return nil, fmt.Errorf("unknown storage type: %s", cfg.Type)
	}
}
