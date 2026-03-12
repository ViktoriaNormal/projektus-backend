package utils

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

// SaveFile saves data to the given relative path, creating directories if needed.
func SaveFile(relPath string, data []byte) error {
	dir := filepath.Dir(relPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return ioutil.WriteFile(relPath, data, 0o644)
}

