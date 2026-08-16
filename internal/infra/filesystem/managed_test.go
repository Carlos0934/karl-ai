package filesystem_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/carlos0934/karl-ai/internal/infra/filesystem"
)

func TestWriteAtomic(t *testing.T) {
	tempDir := t.TempDir()
	targetFile := filepath.Join(tempDir, "subdir", "test.txt")
	content := []byte("hello atomic world")

	if err := filesystem.WriteAtomic(targetFile, content, 0o644); err != nil {
		t.Fatalf("WriteAtomic failed: %v", err)
	}

	readBack, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	if string(readBack) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", string(readBack), string(content))
	}

	// Overwrite atomically
	updatedContent := []byte("updated content")
	if err := filesystem.WriteAtomic(targetFile, updatedContent, 0o644); err != nil {
		t.Fatalf("WriteAtomic overwrite failed: %v", err)
	}

	readBack2, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("ReadFile failed after overwrite: %v", err)
	}
	if string(readBack2) != string(updatedContent) {
		t.Errorf("content mismatch after overwrite: got %q, want %q", string(readBack2), string(updatedContent))
	}
}
