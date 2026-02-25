package home

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// EnvHome overrides the default base directory (~/.mcp2cli) when set.
	EnvHome = "MCP2CLI_HOME"
)

type Paths struct {
	HomeDir   string
	BaseDir   string
	BinDir    string
	ConfigDir string
}

func Resolve(baseDirOverride string) (Paths, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, fmt.Errorf("resolve user home directory: %w", err)
	}

	baseDir := strings.TrimSpace(baseDirOverride)
	if baseDir == "" {
		if fromEnv := strings.TrimSpace(os.Getenv(EnvHome)); fromEnv != "" {
			baseDir = fromEnv
		} else {
			baseDir = filepath.Join(homeDir, ".mcp2cli")
		}
	}
	baseDir = filepath.Clean(baseDir)

	return Paths{
		HomeDir:   homeDir,
		BaseDir:   baseDir,
		BinDir:    filepath.Join(baseDir, "bin"),
		ConfigDir: filepath.Join(baseDir, "configs"),
	}, nil
}

func EnsureLayout(paths Paths) error {
	for _, dir := range []string{paths.BaseDir, paths.BinDir, paths.ConfigDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("create directory %q: %w", dir, err)
		}
	}
	return nil
}
