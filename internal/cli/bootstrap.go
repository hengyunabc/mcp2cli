package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/hengyunabc/mcp2cli/internal/apperr"
	"github.com/hengyunabc/mcp2cli/internal/home"
	"github.com/hengyunabc/mcp2cli/internal/install"
	"github.com/hengyunabc/mcp2cli/internal/shellpath"
	"github.com/spf13/cobra"
)

func NewBootstrapCommand(stdout io.Writer, stderr io.Writer, stdin io.Reader, version string, manager *install.Manager) *cobra.Command {
	var showVersion bool

	cmd := &cobra.Command{
		Use:   "mcp2cli",
		Short: "MCP to CLI bridge",
		Long: strings.TrimSpace(`
MCP to CLI bridge.

Install wrapper commands into ~/.mcp2cli/bin and keep config files in ~/.mcp2cli/configs.

Examples:
  mcp2cli install weather --url https://example.com/mcp --token <token>
  mcp2cli list
  mcp2cli remove weather
  mcp2cli --config weather.json cityInfo --name hk
`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			if showVersion {
				_, err := fmt.Fprintln(stdout, version)
				return err
			}
			return cmd.Help()
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.Flags().StringP("config", "c", "", "Path to config file")
	cmd.Flags().BoolVar(&showVersion, "version", false, "Show version")

	cmd.AddCommand(newBootstrapVersionCommand(stdout, version))
	cmd.AddCommand(newInstallCommand(manager, stdin, stdout))
	cmd.AddCommand(newListCommand(manager, stdout))
	cmd.AddCommand(newRemoveCommand(manager, stdout))
	return cmd
}

func newBootstrapVersionCommand(stdout io.Writer, version string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintln(stdout, version)
			return err
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}
}

func newInstallCommand(manager *install.Manager, stdin io.Reader, stdout io.Writer) *cobra.Command {
	var (
		url            string
		token          string
		timeoutSeconds int
		description    string
		fromConfig     string
		force          bool
		yes            bool
	)

	cmd := &cobra.Command{
		Use:   "install <name>",
		Short: "Install a wrapper command into ~/.mcp2cli/bin",
		Long: strings.TrimSpace(`
Install wrapper command and config:
  wrapper -> ~/.mcp2cli/bin/<name>
  config  -> ~/.mcp2cli/configs/<name>.json

By default this command refuses to overwrite existing files.
Use --force to overwrite.
`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])
			result, err := manager.Install(install.InstallOptions{
				Name:           name,
				URL:            url,
				Token:          token,
				TimeoutSeconds: timeoutSeconds,
				Description:    description,
				FromConfigPath: fromConfig,
				Force:          force,
			})
			if err != nil {
				return apperr.Wrap(apperr.CodeConfig, err, "install %q", name)
			}

			_, _ = fmt.Fprintf(stdout, "Installed command: %s\n", result.Name)
			_, _ = fmt.Fprintf(stdout, "Wrapper: %s\n", result.WrapperPath)
			_, _ = fmt.Fprintf(stdout, "Config:  %s\n", result.ConfigPath)

			printPathGuidance(manager.Paths, stdin, stdout, yes)
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().StringVar(&url, "url", "", "MCP server URL")
	cmd.Flags().StringVar(&token, "token", "", "Bearer token")
	cmd.Flags().IntVar(&timeoutSeconds, "timeout", 30, "Server timeout in seconds")
	cmd.Flags().StringVar(&description, "description", "", "CLI description")
	cmd.Flags().StringVar(&fromConfig, "from-config", "", "Import from an existing JSON config file")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing config/wrapper if present")
	cmd.Flags().BoolVar(&yes, "yes", false, "Automatically accept PATH update prompt")
	return cmd
}

func newListCommand(manager *install.Manager, stdout io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed wrapper commands",
		RunE: func(cmd *cobra.Command, _ []string) error {
			entries, err := manager.List()
			if err != nil {
				return apperr.Wrap(apperr.CodeInternal, err, "list installed commands")
			}
			if len(entries) == 0 {
				_, err := fmt.Fprintf(stdout, "No installed commands under %s\n", manager.Paths.BaseDir)
				return err
			}

			w := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "NAME\tWRAPPER\tCONFIG\tURL")
			for _, entry := range entries {
				url := entry.ServerURL
				if strings.TrimSpace(url) == "" {
					url = "-"
				}
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", entry.Name, boolMarker(entry.WrapperExists), entry.ConfigPath, url)
			}
			return w.Flush()
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	return cmd
}

func newRemoveCommand(manager *install.Manager, stdout io.Writer) *cobra.Command {
	var keepConfig bool

	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove an installed wrapper command",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := strings.TrimSpace(args[0])
			result, err := manager.Remove(name, keepConfig)
			if err != nil {
				return apperr.Wrap(apperr.CodeConfig, err, "remove %q", name)
			}

			if result.WrapperRemoved {
				_, _ = fmt.Fprintf(stdout, "Removed wrapper: %s\n", result.WrapperPath)
			}
			if result.ConfigRemoved {
				_, _ = fmt.Fprintf(stdout, "Removed config:  %s\n", result.ConfigPath)
			}
			if keepConfig {
				_, _ = fmt.Fprintf(stdout, "Kept config:     %s\n", result.ConfigPath)
			}
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.Flags().BoolVar(&keepConfig, "keep-config", false, "Keep config file and remove wrapper only")
	return cmd
}

func printPathGuidance(paths home.Paths, stdin io.Reader, stdout io.Writer, yes bool) {
	manual := shellpath.PathExportLine
	if shellpath.PathContains(os.Getenv("PATH"), paths.BinDir) {
		_, _ = fmt.Fprintf(stdout, "PATH already contains %s\n", paths.BinDir)
		return
	}

	rcPath, shellName, supported := shellpath.DetectRCFile(os.Getenv("SHELL"), paths.HomeDir)
	if !supported {
		_, _ = fmt.Fprintf(stdout, "Cannot auto-detect shell rc file (SHELL=%q).\n", shellName)
		_, _ = fmt.Fprintf(stdout, "Add PATH manually:\n  %s\n", manual)
		return
	}

	allowUpdate := yes
	if !allowUpdate {
		if !isInteractiveInput(stdin) {
			_, _ = fmt.Fprintf(stdout, "PATH is missing %s\n", paths.BinDir)
			_, _ = fmt.Fprintf(stdout, "Add it manually:\n  %s\n", manual)
			return
		}

		_, _ = fmt.Fprintf(stdout, "Add %s to PATH in %s now? [y/N]: ", paths.BinDir, rcPath)
		allowUpdate = readYes(stdin)
	}

	if !allowUpdate {
		_, _ = fmt.Fprintf(stdout, "Skipped PATH update. Add manually:\n  %s\n", manual)
		return
	}

	added, err := shellpath.EnsurePathExportLine(rcPath, shellpath.PathExportLine)
	if err != nil {
		_, _ = fmt.Fprintf(stdout, "Failed to update %s: %v\n", rcPath, err)
		_, _ = fmt.Fprintf(stdout, "Add PATH manually:\n  %s\n", manual)
		return
	}

	if added {
		_, _ = fmt.Fprintf(stdout, "Updated %s\n", rcPath)
	} else {
		_, _ = fmt.Fprintf(stdout, "PATH export already exists in %s\n", rcPath)
	}
	_, _ = fmt.Fprintf(stdout, "Run: source %s\n", rcPath)
}

func readYes(stdin io.Reader) bool {
	reader := bufio.NewReader(stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		line = strings.TrimSpace(line)
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}

func isInteractiveInput(stdin io.Reader) bool {
	f, ok := stdin.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

func boolMarker(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}
