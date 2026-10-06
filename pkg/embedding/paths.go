package embedding

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

const RuntimeDirName = "runtime"

// RuntimeDir returns the directory holding platform-specific onnxruntime libraries.
func RuntimeDir(rootDir, goos, goarch string) string {
	return filepath.Join(rootDir, RuntimeDirName, goos+"-"+goarch)
}

// LibraryFileName maps a GOOS to the canonical onnxruntime shared library name.
func LibraryFileName(goos string) (string, error) {
	switch goos {
	case "windows":
		return "onnxruntime.dll", nil
	case "linux":
		return "libonnxruntime.so", nil
	case "darwin":
		return "libonnxruntime.dylib", nil
	default:
		return "", fmt.Errorf("unsupported GOOS %q", goos)
	}
}

// RuntimeLibraryPath returns the expected shared library path for a platform.
func RuntimeLibraryPath(rootDir, goos, goarch string) (string, error) {
	name, err := LibraryFileName(goos)
	if err != nil {
		return "", err
	}
	return filepath.Join(RuntimeDir(rootDir, goos, goarch), name), nil
}

// CurrentRuntimeLibraryPath returns the shared library path for the host platform.
func CurrentRuntimeLibraryPath(rootDir string) (string, error) {
	return RuntimeLibraryPath(rootDir, runtime.GOOS, runtime.GOARCH)
}

// SanitizeModelID rejects model identifiers that could escape the data root or cause OS path conflicts.
func SanitizeModelID(modelID string) error {
	if modelID == "" || modelID == "." || modelID == ".." {
		return fmt.Errorf("invalid model id %q", modelID)
	}
	if strings.ContainsAny(modelID, "/\\") {
		return fmt.Errorf("invalid model id %q", modelID)
	}
	if strings.IndexByte(modelID, 0) >= 0 {
		return fmt.Errorf("invalid model id %q: contains null byte", modelID)
	}
	if len(modelID) > 128 {
		return fmt.Errorf("invalid model id: exceeds 128 characters")
	}
	if modelID != strings.Trim(modelID, " .") {
		return fmt.Errorf("invalid model id %q: contains leading or trailing dots or spaces", modelID)
	}
	switch strings.ToUpper(modelID) {
	case "CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
		"RUNTIME":
		return fmt.Errorf("reserved model id %q", modelID)
	}
	return nil
}
