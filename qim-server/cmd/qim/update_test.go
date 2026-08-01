package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestReplaceBinary_Success(t *testing.T) {
	dir := t.TempDir()
	selfPath := filepath.Join(dir, "qim")
	tmpPath := selfPath + ".tmp"
	backupPath := selfPath + ".bak"

	if err := os.WriteFile(selfPath, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmpPath, []byte("new"), 0o755); err != nil {
		t.Fatal(err)
	}

	rolledBack, err := replaceBinary(selfPath, tmpPath, func(p string) error { return nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rolledBack {
		t.Fatal("should not roll back on success")
	}

	data, err := os.ReadFile(selfPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new" {
		t.Fatalf("expected new content, got %s", data)
	}
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Fatal("backup should be removed on success")
	}
}

func TestReplaceBinary_RollbackOnVerifyFailure(t *testing.T) {
	dir := t.TempDir()
	selfPath := filepath.Join(dir, "qim")
	tmpPath := selfPath + ".tmp"

	if err := os.WriteFile(selfPath, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmpPath, []byte("broken"), 0o755); err != nil {
		t.Fatal(err)
	}

	rolledBack, err := replaceBinary(selfPath, tmpPath, func(p string) error {
		return fmt.Errorf("binary corrupted")
	})
	if err == nil {
		t.Fatal("expected error from verify")
	}
	if !rolledBack {
		t.Fatal("should roll back on verify failure")
	}

	data, err := os.ReadFile(selfPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "old" {
		t.Fatalf("expected old content after rollback, got %s", data)
	}
}

func TestReplaceBinary_TmpMissing(t *testing.T) {
	dir := t.TempDir()
	selfPath := filepath.Join(dir, "qim")
	tmpPath := selfPath + ".tmp"

	if err := os.WriteFile(selfPath, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := replaceBinary(selfPath, tmpPath, nil)
	if err == nil {
		t.Fatal("expected error when tmp missing")
	}

	data, _ := os.ReadFile(selfPath)
	if string(data) != "old" {
		t.Fatal("original should be intact")
	}
}

func TestDownloadBinaryRejectsOversized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("a"), maxBinarySize+1))
	}))
	t.Cleanup(srv.Close)

	if _, err := downloadBinary(srv.URL); err == nil {
		t.Fatal("expected error for oversized binary")
	}
}

func TestDownloadBinarySuccess(t *testing.T) {
	payload := []byte("hello binary")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	t.Cleanup(srv.Close)

	got, err := downloadBinary(srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("expected %q, got %q", payload, got)
	}
}
