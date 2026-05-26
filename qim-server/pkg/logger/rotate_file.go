package logger

import (
	"os"
	"path/filepath"
	"sync"
	"time"
)

type RotateFile struct {
	dir    string
	name   string
	mu     sync.Mutex
	file   *os.File
	date   string
}

func NewRotateFile(path string) (*RotateFile, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	return &RotateFile{
		dir:  dir,
		name: filepath.Base(path),
	}, nil
}

func (f *RotateFile) Write(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	today := time.Now().Format("2006-01-02")
	if today != f.date {
		if err := f.rotate(today); err != nil {
			return 0, err
		}
	}

	return f.file.Write(p)
}

func (f *RotateFile) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.file != nil {
		return f.file.Close()
	}
	return nil
}

func (f *RotateFile) rotate(today string) error {
	if f.file != nil {
		f.file.Close()
		f.file = nil
	}

	currentPath := filepath.Join(f.dir, f.name)

	if f.date != "" {
		archivedPath := filepath.Join(f.dir, f.name+"."+f.date)
		os.Rename(currentPath, archivedPath)
	}

	f.date = today

	file, err := os.OpenFile(currentPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	f.file = file
	return nil
}
