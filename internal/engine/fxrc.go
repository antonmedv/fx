package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func readFxrc() (string, error) {
	var builder strings.Builder

	// Determine search paths
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get cwd: %w", err)
	}
	paths := []string{filepath.Join(cwd, ".fxrc.js")}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home: %w", err)
	}
	paths = append(paths, filepath.Join(home, ".fxrc.js"))

	xdgHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgHome == "" {
		xdgHome = filepath.Join(home, ".config")
	}
	paths = append(paths, filepath.Join(xdgHome, "fx", ".fxrc.js"))

	xdgDirs := os.Getenv("XDG_CONFIG_DIRS")
	if xdgDirs == "" {
		xdgDirs = "/etc/xdg"
	}
	for _, dir := range strings.Split(xdgDirs, ":") {
		paths = append(paths, filepath.Join(dir, "fx", ".fxrc.js"))
	}

	// Read and combine
	for _, path := range uniq(paths) {
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			continue // skip missing or directories
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", path, err)
		}
		builder.Write(data)
		builder.WriteString("\n")
	}

	return builder.String(), nil
}

func uniq(paths []string) []string {
	seen := make(map[string]bool)
	result := []string{}
	for _, path := range paths {
		if !seen[path] {
			seen[path] = true
			result = append(result, path)
		}
	}
	return result
}
