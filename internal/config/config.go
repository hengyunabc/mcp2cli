package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hengyunabc/mcp2cli/internal/home"
)

const (
	DefaultTimeoutSeconds = 30
	DefaultTransport      = "streamable_http"
	DefaultName           = "mcp2cli"
)

type Config struct {
	Name   string       `json:"name"`
	Server ServerConfig `json:"server"`
	Auth   AuthConfig   `json:"auth"`
	CLI    CLIConfig    `json:"cli"`
}

type ServerConfig struct {
	URL            string `json:"url"`
	Transport      string `json:"transport"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

type AuthConfig struct {
	Type  string `json:"type"`
	Token string `json:"token"`
}

type CLIConfig struct {
	Description               string `json:"description"`
	IncludeReturnSchemaInHelp *bool  `json:"include_return_schema_in_help"`
	PrettyJSON                bool   `json:"pretty_json"`
}

func Load(path string) (Config, error) {
	if strings.TrimSpace(path) == "" {
		return Config{}, fmt.Errorf("config path is empty")
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config file %q: %w", path, err)
	}

	expanded := os.ExpandEnv(string(raw))

	cfg := Config{
		Name: DefaultName,
		Server: ServerConfig{
			Transport:      DefaultTransport,
			TimeoutSeconds: DefaultTimeoutSeconds,
		},
	}
	if err := json.Unmarshal([]byte(expanded), &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config file %q: %w", path, err)
	}

	applyDefaults(&cfg)
	if err := validate(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func applyDefaults(cfg *Config) {
	if strings.TrimSpace(cfg.Name) == "" {
		cfg.Name = DefaultName
	}
	if strings.TrimSpace(cfg.Server.Transport) == "" {
		cfg.Server.Transport = DefaultTransport
	}
	if cfg.Server.TimeoutSeconds <= 0 {
		cfg.Server.TimeoutSeconds = DefaultTimeoutSeconds
	}
}

func validate(cfg Config) error {
	if strings.TrimSpace(cfg.Server.URL) == "" {
		return fmt.Errorf("config server.url is required")
	}
	if cfg.Server.Transport != DefaultTransport {
		return fmt.Errorf("unsupported server.transport %q, only %q is supported", cfg.Server.Transport, DefaultTransport)
	}
	authType := strings.TrimSpace(cfg.Auth.Type)
	if authType == "" {
		return nil
	}
	if authType != "bearer" {
		return fmt.Errorf("unsupported auth.type %q, supported: bearer", cfg.Auth.Type)
	}
	if strings.TrimSpace(cfg.Auth.Token) == "" {
		return fmt.Errorf("auth.token is required when auth.type is bearer")
	}
	return nil
}

func IncludeReturnSchemaInHelp(cfg Config) bool {
	if cfg.CLI.IncludeReturnSchemaInHelp == nil {
		return true
	}
	return *cfg.CLI.IncludeReturnSchemaInHelp
}

func ExtractConfigPath(args []string, programName string) (string, error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--config" || arg == "-c":
			if i+1 >= len(args) {
				return "", fmt.Errorf("%s requires a value", arg)
			}
			value := strings.TrimSpace(args[i+1])
			if value == "" {
				return "", fmt.Errorf("%s value is empty", arg)
			}
			return value, nil
		case strings.HasPrefix(arg, "--config="):
			value := strings.TrimSpace(strings.TrimPrefix(arg, "--config="))
			if value == "" {
				return "", fmt.Errorf("--config value is empty")
			}
			return value, nil
		case strings.HasPrefix(arg, "-c="):
			value := strings.TrimSpace(strings.TrimPrefix(arg, "-c="))
			if value == "" {
				return "", fmt.Errorf("-c value is empty")
			}
			return value, nil
		}
	}

	if envPath := strings.TrimSpace(os.Getenv("MCP2CLI_CONFIG")); envPath != "" {
		return envPath, nil
	}

	if fallbackPath, ok, err := ExpectedConfigPathForProgram(programName); err != nil {
		return "", err
	} else if ok {
		if _, err := os.Stat(fallbackPath); err == nil {
			return fallbackPath, nil
		}
	}

	return "", nil
}

func ExpectedConfigPathForProgram(programName string) (string, bool, error) {
	commandName := commandNameFromProgram(programName)
	if commandName == "" || commandName == "mcp2cli" {
		return "", false, nil
	}

	paths, err := home.Resolve("")
	if err != nil {
		return "", false, fmt.Errorf("resolve default config directory: %w", err)
	}

	return filepath.Join(paths.ConfigDir, commandName+".json"), true, nil
}

func commandNameFromProgram(programName string) string {
	base := strings.TrimSpace(filepath.Base(programName))
	if base == "" || base == "." || base == string(filepath.Separator) {
		return ""
	}
	// Handle Windows executable names while keeping other platforms unchanged.
	if strings.EqualFold(filepath.Ext(base), ".exe") {
		base = strings.TrimSuffix(base, filepath.Ext(base))
	}
	return strings.TrimSpace(base)
}
