package orthrus

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func IsTemplateFile(file os.DirEntry) bool {
	return !file.IsDir() && strings.HasSuffix(file.Name(), ".tmpl")
}

func IsMarkdownFile(file os.DirEntry) bool {
	ext := strings.ToLower(filepath.Ext(file.Name()))
	return !file.IsDir() && (ext == ".md" || ext == ".markdown")
}

func PurgeDir(dir string) error {
	if err := RemoveIfExists(dir); err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return nil
}

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

// CopyDir copies a directory from src to dest
func CopyDir(src string, dst string) error {
	files, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, file := range files {
		srcPath := filepath.Join(src, file.Name())
		dstPath := filepath.Join(dst, file.Name())

		if file.IsDir() {
			err := os.MkdirAll(dstPath, os.ModePerm)
			if err != nil {
				return err
			}
			err = CopyDir(srcPath, dstPath)
			if err != nil {
				return err
			}
		} else {
			err := Copy(srcPath, dstPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func IsDatePrefixed(filename string) bool {
	base := filepath.Base(filename)
	re := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)
	return re.MatchString(base)
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
