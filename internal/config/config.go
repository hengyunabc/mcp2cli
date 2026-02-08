package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
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

func ExtractConfigPath(args []string) (string, error) {
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
	return "", nil
}
