package embedding

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"tluagent-web/pkg/jsonx"
)

// marshalJSON serializes a value with the project JSON codec.
func marshalJSON(v any) ([]byte, error) {
	return jsonx.Marshal(v)
}

// unmarshalJSON parses JSON data into a target value.
func unmarshalJSON(data []byte, v any) error {
	return jsonx.Unmarshal(data, v)
}

// atomicWriteFile writes data to a temp file then renames it over the destination.
func atomicWriteFile(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Chmod(tmpName, mode); err != nil {
		return fmt.Errorf("chmod temp file: %w", err)
	}
	_ = os.Remove(path)
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("move file into place: %w", err)
	}
	return nil
}

const (
	MaxManifestSizeBytes   = 16 << 20
	MaxVocabSizeBytes      = 64 << 20
	MaxValidationSizeBytes = 1 << 20
	MaxChecksumsSizeBytes  = 1 << 20
)

// checkNotSymlink ensures that the path is not a symbolic link.
func checkNotSymlink(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("symbolic links are forbidden: %s", filepath.Base(path))
	}
	return nil
}

// readBoundedFile reads a regular file with a maximum size limit and symlink guard.
func readBoundedFile(path string, maxBytes int64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("symbolic links are forbidden: %s", filepath.Base(path))
	}
	if info.IsDir() {
		return nil, fmt.Errorf("expected regular file but found directory: %s", filepath.Base(path))
	}
	if info.Size() > maxBytes {
		return nil, fmt.Errorf("file %s exceeds maximum size of %d bytes", filepath.Base(path), maxBytes)
	}
	return os.ReadFile(path)
}

// ComputeFileSHA256 returns the hexadecimal SHA-256 digest of a regular file.
func ComputeFileSHA256(path string) (string, error) {
	if err := checkNotSymlink(path); err != nil {
		return "", err
	}
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// CleanupStaleTempFiles scans and deletes leftover temporary download and extract files.
func CleanupStaleTempFiles(root string, maxAge time.Duration) {
	now := time.Now()
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".download-") ||
			strings.HasPrefix(name, ".extract-") ||
			strings.HasPrefix(name, ".archive-") ||
			strings.HasPrefix(name, ".tmp-") {
			if info, err := entry.Info(); err == nil && now.Sub(info.ModTime()) > maxAge {
				_ = os.Remove(path)
			}
		}
		return nil
	})
}

