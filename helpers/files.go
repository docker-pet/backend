package helpers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func WriteFileIfChanged(path string, content []byte, perm os.FileMode) (bool, error) {
	old, err := os.ReadFile(path)
	if err == nil && len(old) == len(content) && string(old) == string(content) {
		return false, nil
	}
	return true, os.WriteFile(path, content, perm)
}

func ReadFileIfExists(path string) ([]byte, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to stat file %q: %w", path, err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", path, err)
	}

	return content, nil
}

func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

func MapFileNamesToPaths(dirPath, extension string) (map[string]string, error) {
	if !strings.HasPrefix(extension, ".") {
		extension = "." + extension
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read dir %q: %w", dirPath, err)
	}

	files := make(map[string]string)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), extension) {
			fullPath := filepath.Join(dirPath, entry.Name())
			files[entry.Name()] = fullPath
		}
	}

	return files, nil
}

func RemoveFileIfExists(path string) error {
	err := os.Remove(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return nil
}
