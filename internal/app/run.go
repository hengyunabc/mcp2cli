package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/hengyunabc/mcp2cli/internal/apperr"
	"github.com/hengyunabc/mcp2cli/internal/cli"
	"github.com/hengyunabc/mcp2cli/internal/config"
	"github.com/hengyunabc/mcp2cli/internal/install"
	"github.com/hengyunabc/mcp2cli/internal/mcpclient"
)

func Run(ctx context.Context, programName string, args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, version string) int {
	if shouldRunBootstrap(programName, args) {
		manager, err := install.NewManager("")
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err.Error())
			return apperr.CodeInternal
		}
		bootstrap := cli.NewBootstrapCommand(stdout, stderr, stdin, version, manager)
		bootstrap.SetArgs(args)
		if err := bootstrap.ExecuteContext(ctx); err != nil {
			return handleCommandError(err, stderr)
		}
		return apperr.CodeOK
	}

	cfgPath, err := config.ExtractConfigPath(args, programName)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return apperr.CodeConfig
	}

	if cfgPath == "" {
		if expectedPath, ok, err := config.ExpectedConfigPathForProgram(programName); err == nil && ok {
			commandName := commandName(programName)
			_, _ = fmt.Fprintf(stderr, "missing config for command %q\n", commandName)
			_, _ = fmt.Fprintf(stderr, "expected config file: %s\n", expectedPath)
			_, _ = fmt.Fprintf(stderr, "create it with: mcp2cli install %s --url <mcp_server_url> [--token <token>]\n", commandName)
			return apperr.CodeConfig
		} else if err != nil {
			_, _ = fmt.Fprintln(stderr, err.Error())
			return apperr.CodeConfig
		}
		_, _ = fmt.Fprintln(stderr, "missing --config. Example: mcp2cli --config weather.json --help")
		_, _ = fmt.Fprintln(stderr, "or install wrapper command: mcp2cli install <name> --url <mcp_server_url> [--token <token>]")
		return apperr.CodeConfig
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return apperr.CodeConfig
	}

	invokedName := commandName(programName)
	if invokedName != "" && invokedName != "mcp2cli" && cfg.Name == config.DefaultName {
		cfg.Name = invokedName
	}

	client, err := mcpclient.Connect(ctx, cfg, version)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return apperr.CodeConnection
	}
	defer func() {
		_ = client.Close()
	}()

	root, err := cli.NewRootCommand(cli.Runtime{
		ConfigPath: cfgPath,
		Config:     cfg,
		Client:     client,
		Stdout:     stdout,
		Stderr:     stderr,
		Version:    version,
	})
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return apperr.CodeInternal
	}

	root.SetArgs(args)
	if err := root.ExecuteContext(ctx); err != nil {
		return handleCommandError(err, stderr)
	}
	return apperr.CodeOK
}

func handleCommandError(err error, stderr io.Writer) int {
	if err == nil {
		return apperr.CodeOK
	}

	_, _ = fmt.Fprintln(stderr, err.Error())

	var appErr *apperr.Error
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return apperr.CodeGeneric
}

func shouldRunBootstrap(programName string, args []string) bool {
	if commandName(programName) != "mcp2cli" {
		return false
	}
	if len(args) == 0 {
		return true
	}

	first := strings.TrimSpace(args[0])
	switch first {
	case "help", "version", "install", "list", "remove", "completion", "-h", "--help", "--version":
		return true
	}

	return false
}

func commandName(programName string) string {
	base := strings.TrimSpace(filepath.Base(programName))
	if base == "" {
		return ""
	}
	base = strings.TrimSuffix(base, filepath.Ext(base))
	return strings.TrimSpace(base)
}
