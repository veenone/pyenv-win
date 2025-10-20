package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// EnsureDir creates a directory and all parent directories if they don't exist
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// DirExists checks if a directory exists
func DirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}

// RemoveAll removes a directory and all its contents
func RemoveAll(path string) error {
	return os.RemoveAll(path)
}

// ListDir lists all files and directories in a directory
func ListDir(path string) ([]os.FileInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return f.Readdir(-1)
}

// WriteFile writes data to a file
func WriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

// ReadFile reads data from a file
func ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// GetAbsolutePath returns the absolute path
func GetAbsolutePath(path string) (string, error) {
	return filepath.Abs(path)
}

// GetParentDir returns the parent directory of a path
func GetParentDir(path string) string {
	return filepath.Dir(path)
}

// GetBaseName returns the base name of a path
func GetBaseName(path string) string {
	return filepath.Base(path)
}

// GetExtension returns the extension of a file
func GetExtension(path string) string {
	return filepath.Ext(path)
}

// JoinPath joins path elements
func JoinPath(elem ...string) string {
	return filepath.Join(elem...)
}

// MoveFile moves a file from src to dst
func MoveFile(src, dst string) error {
	err := CopyFile(src, dst)
	if err != nil {
		return err
	}
	return os.Remove(src)
}

// DeleteFile deletes a file
func DeleteFile(path string) error {
	return os.Remove(path)
}

// CreateTempDir creates a temporary directory
func CreateTempDir(prefix string) (string, error) {
	return os.MkdirTemp("", prefix)
}

// FindFilesWithExt finds all files with a specific extension in a directory
func FindFilesWithExt(dir, ext string) ([]string, error) {
	var files []string

	entries, err := ListDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ext {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}

	return files, nil
}

// GetFileSize returns the size of a file
func GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// IsEmptyDir checks if a directory is empty
func IsEmptyDir(path string) (bool, error) {
	entries, err := ListDir(path)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

// PrintInfo prints an info message
func PrintInfo(format string, args ...interface{}) {
	fmt.Printf(":: [Info] ::  "+format+"\n", args...)
}

// PrintError prints an error message
func PrintError(format string, args ...interface{}) {
	fmt.Printf(":: [Error] ::  "+format+"\n", args...)
}

// PrintDownloading prints a downloading message
func PrintDownloading(format string, args ...interface{}) {
	fmt.Printf(":: [Downloading] ::  "+format+"\n", args...)
}

// PrintInstalling prints an installing message
func PrintInstalling(format string, args ...interface{}) {
	fmt.Printf(":: [Installing] ::  "+format+"\n", args...)
}
