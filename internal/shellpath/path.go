package shellpath

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const PathExportLine = `export PATH="$HOME/.mcp2cli/bin:$PATH"`

func PathContains(pathEnv string, targetDir string) bool {
	targetAbs := filepath.Clean(targetDir)
	for _, entry := range filepath.SplitList(pathEnv) {
		if strings.TrimSpace(entry) == "" {
			continue
		}
		if filepath.Clean(entry) == targetAbs {
			return true
		}
	}
	return false
}

func DetectRCFile(shellEnv string, homeDir string) (string, string, bool) {
	shellName := strings.TrimSpace(filepath.Base(shellEnv))
	switch shellName {
	case "bash":
		return filepath.Join(homeDir, ".bashrc"), shellName, true
	case "zsh":
		return filepath.Join(homeDir, ".zshrc"), shellName, true
	default:
		return "", shellName, false
	}
}

func EnsurePathExportLine(rcPath string, exportLine string) (bool, error) {
	if strings.TrimSpace(rcPath) == "" {
		return false, fmt.Errorf("rc file path is empty")
	}
	if strings.TrimSpace(exportLine) == "" {
		return false, fmt.Errorf("export line is empty")
	}

	content, err := os.ReadFile(rcPath)
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("read rc file %q: %w", rcPath, err)
	}

	if strings.Contains(string(content), exportLine) {
		return false, nil
	}

	f, err := os.OpenFile(rcPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return false, fmt.Errorf("open rc file %q: %w", rcPath, err)
	}
	defer func() {
		_ = f.Close()
	}()

	if len(content) > 0 && !strings.HasSuffix(string(content), "\n") {
		if _, err := f.WriteString("\n"); err != nil {
			return false, fmt.Errorf("write newline to rc file %q: %w", rcPath, err)
		}
	}
	if len(content) > 0 {
		if _, err := f.WriteString("\n"); err != nil {
			return false, fmt.Errorf("write separator newline to rc file %q: %w", rcPath, err)
		}
	}

	if _, err := f.WriteString(exportLine + "\n"); err != nil {
		return false, fmt.Errorf("append path export to rc file %q: %w", rcPath, err)
	}

	return true, nil
}
