package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/hengyunabc/mcp2cli/internal/config"
	"github.com/hengyunabc/mcp2cli/internal/home"
)

var commandNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

type Manager struct {
	Paths home.Paths
}

type InstallOptions struct {
	Name           string
	URL            string
	Token          string
	TimeoutSeconds int
	Description    string
	FromConfigPath string
	Force          bool
}

type InstallResult struct {
	Name        string
	ConfigPath  string
	WrapperPath string
}

type RemoveResult struct {
	Name           string
	ConfigPath     string
	WrapperPath    string
	ConfigRemoved  bool
	WrapperRemoved bool
}

type Entry struct {
	Name          string
	ConfigPath    string
	WrapperPath   string
	WrapperExists bool
	ServerURL     string
}

func NewManager(baseDirOverride string) (*Manager, error) {
	paths, err := home.Resolve(baseDirOverride)
	if err != nil {
		return nil, err
	}
	return &Manager{Paths: paths}, nil
}

func (m *Manager) Install(opts InstallOptions) (InstallResult, error) {
	name := strings.TrimSpace(opts.Name)
	if err := validateName(name); err != nil {
		return InstallResult{}, err
	}

	cfg, err := buildConfig(name, opts)
	if err != nil {
		return InstallResult{}, err
	}

	if err := home.EnsureLayout(m.Paths); err != nil {
		return InstallResult{}, err
	}

	configPath := filepath.Join(m.Paths.ConfigDir, name+".json")
	wrapperPath := filepath.Join(m.Paths.BinDir, name)

	if !opts.Force {
		if exists(configPath) {
			return InstallResult{}, fmt.Errorf("config already exists: %s (use --force to overwrite)", configPath)
		}
		if exists(wrapperPath) {
			return InstallResult{}, fmt.Errorf("wrapper already exists: %s (use --force to overwrite)", wrapperPath)
		}
	} else {
		if exists(configPath) {
			if err := os.Remove(configPath); err != nil {
				return InstallResult{}, fmt.Errorf("remove existing config %q: %w", configPath, err)
			}
		}
		if exists(wrapperPath) {
			if err := os.Remove(wrapperPath); err != nil {
				return InstallResult{}, fmt.Errorf("remove existing wrapper %q: %w", wrapperPath, err)
			}
		}
	}

	configContent, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return InstallResult{}, fmt.Errorf("marshal config: %w", err)
	}
	configContent = append(configContent, '\n')

	if err := os.WriteFile(configPath, configContent, 0o600); err != nil {
		return InstallResult{}, fmt.Errorf("write config %q: %w", configPath, err)
	}

	exePath, err := os.Executable()
	if err != nil {
		return InstallResult{}, fmt.Errorf("resolve current executable path: %w", err)
	}
	exePath, err = filepath.Abs(exePath)
	if err != nil {
		return InstallResult{}, fmt.Errorf("resolve absolute executable path %q: %w", exePath, err)
	}
	exePath = filepath.Clean(exePath)
	if err := os.Symlink(exePath, wrapperPath); err != nil {
		return InstallResult{}, fmt.Errorf("create wrapper symlink %q -> %q: %w", wrapperPath, exePath, err)
	}

	return InstallResult{
		Name:        name,
		ConfigPath:  configPath,
		WrapperPath: wrapperPath,
	}, nil
}

func (m *Manager) List() ([]Entry, error) {
	if !exists(m.Paths.ConfigDir) && !exists(m.Paths.BinDir) {
		return nil, nil
	}

	entryMap := map[string]*Entry{}

	configFiles, err := filepath.Glob(filepath.Join(m.Paths.ConfigDir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("list config files: %w", err)
	}

	for _, configPath := range configFiles {
		name := strings.TrimSuffix(filepath.Base(configPath), ".json")
		if !commandNameRegex.MatchString(name) {
			continue
		}

		item := &Entry{
			Name:        name,
			ConfigPath:  configPath,
			WrapperPath: filepath.Join(m.Paths.BinDir, name),
		}

		cfg, err := config.Load(configPath)
		if err == nil {
			item.ServerURL = cfg.Server.URL
		}
		item.WrapperExists = exists(item.WrapperPath)
		entryMap[name] = item
	}

	wrapperFiles, err := os.ReadDir(m.Paths.BinDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("list wrapper files: %w", err)
	}

	for _, file := range wrapperFiles {
		if file.IsDir() {
			continue
		}
		name := file.Name()
		if !commandNameRegex.MatchString(name) {
			continue
		}

		if _, ok := entryMap[name]; ok {
			continue
		}

		entryMap[name] = &Entry{
			Name:          name,
			ConfigPath:    filepath.Join(m.Paths.ConfigDir, name+".json"),
			WrapperPath:   filepath.Join(m.Paths.BinDir, name),
			WrapperExists: true,
		}
	}

	names := make([]string, 0, len(entryMap))
	for name := range entryMap {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]Entry, 0, len(names))
	for _, name := range names {
		result = append(result, *entryMap[name])
	}
	return result, nil
}

func (m *Manager) Remove(name string, keepConfig bool) (RemoveResult, error) {
	name = strings.TrimSpace(name)
	if err := validateName(name); err != nil {
		return RemoveResult{}, err
	}

	result := RemoveResult{
		Name:        name,
		ConfigPath:  filepath.Join(m.Paths.ConfigDir, name+".json"),
		WrapperPath: filepath.Join(m.Paths.BinDir, name),
	}

	if exists(result.WrapperPath) {
		if err := os.Remove(result.WrapperPath); err != nil {
			return RemoveResult{}, fmt.Errorf("remove wrapper %q: %w", result.WrapperPath, err)
		}
		result.WrapperRemoved = true
	}

	if !keepConfig && exists(result.ConfigPath) {
		if err := os.Remove(result.ConfigPath); err != nil {
			return RemoveResult{}, fmt.Errorf("remove config %q: %w", result.ConfigPath, err)
		}
		result.ConfigRemoved = true
	}

	if !result.WrapperRemoved && !result.ConfigRemoved {
		if keepConfig {
			return RemoveResult{}, fmt.Errorf("wrapper not found: %s", result.WrapperPath)
		}
		return RemoveResult{}, fmt.Errorf("no installed command found for %q", name)
	}

	return result, nil
}

func validateName(name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if !commandNameRegex.MatchString(name) {
		return fmt.Errorf("invalid command name %q: only [a-zA-Z0-9._-] are supported", name)
	}
	return nil
}

func buildConfig(name string, opts InstallOptions) (config.Config, error) {
	fromConfigPath := strings.TrimSpace(opts.FromConfigPath)
	if fromConfigPath != "" {
		if strings.TrimSpace(opts.URL) != "" || strings.TrimSpace(opts.Token) != "" || strings.TrimSpace(opts.Description) != "" {
			return config.Config{}, fmt.Errorf("--from-config cannot be used with --url/--token/--description")
		}
		cfg, err := config.Load(fromConfigPath)
		if err != nil {
			return config.Config{}, fmt.Errorf("load --from-config file: %w", err)
		}
		cfg.Name = name
		return cfg, nil
	}

	url := strings.TrimSpace(opts.URL)
	if url == "" {
		return config.Config{}, fmt.Errorf("--url is required unless --from-config is provided")
	}

	timeoutSeconds := opts.TimeoutSeconds
	if timeoutSeconds <= 0 {
		timeoutSeconds = config.DefaultTimeoutSeconds
	}

	cfg := config.Config{
		Name: name,
		Server: config.ServerConfig{
			URL:            url,
			Transport:      config.DefaultTransport,
			TimeoutSeconds: timeoutSeconds,
		},
		CLI: config.CLIConfig{
			Description:               strings.TrimSpace(opts.Description),
			IncludeReturnSchemaInHelp: boolPtr(true),
			PrettyJSON:                false,
		},
	}

	token := strings.TrimSpace(opts.Token)
	if token != "" {
		cfg.Auth.Type = "bearer"
		cfg.Auth.Token = token
	}
	return cfg, nil
}

func boolPtr(v bool) *bool {
	return &v
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}
