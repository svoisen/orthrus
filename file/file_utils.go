package file

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Copy copies a file from src to dest
func Copy(src string, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	return out.Close()
}

// RemoveIfExists removes a file or directory if it exists
func RemoveIfExists(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return os.RemoveAll(path)
	}

	if os.IsNotExist(err) {
		return nil
	}

	return fmt.Errorf("error checking path: %w", err)
}

// Basename returns the filename without the extension
func Basename(path string) string {
	filename := filepath.Base(path)
	return strings.TrimSuffix(filename, filepath.Ext(path))
}