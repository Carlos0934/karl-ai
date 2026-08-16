package filesystem

import (
	"os"
	"path/filepath"
)

// WriteAtomic replaces a file only after its complete content has been written safely to a temporary file.
func WriteAtomic(path string, content []byte, mode os.FileMode) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(directory, ".karl-ai-*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)

	if _, err = temporary.Write(content); err != nil {
		_ = temporary.Close()
		return err
	}
	if err = temporary.Chmod(mode); err != nil {
		_ = temporary.Close()
		return err
	}
	if err = temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryName, path)
}
