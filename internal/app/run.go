package app

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/hengyunabc/mcp2cli/internal/apperr"
	"github.com/hengyunabc/mcp2cli/internal/cli"
	"github.com/hengyunabc/mcp2cli/internal/config"
	"github.com/hengyunabc/mcp2cli/internal/mcpclient"
)

func Run(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer, version string) int {
	cfgPath, err := config.ExtractConfigPath(args)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return apperr.CodeConfig
	}

	if cfgPath == "" {
		if isHelpOrVersion(args) || len(args) == 0 {
			bootstrap := cli.NewBootstrapCommand(stdout, stderr, version)
			bootstrap.SetArgs(args)
			if err := bootstrap.ExecuteContext(ctx); err != nil {
				_, _ = fmt.Fprintln(stderr, err.Error())
				return apperr.CodeGeneric
			}
			return apperr.CodeOK
		}
		_, _ = fmt.Fprintln(stderr, "missing --config. Example: mcp2cli --config weather.json --help")
		return apperr.CodeConfig
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err.Error())
		return apperr.CodeConfig
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

func isHelpOrVersion(args []string) bool {
	for _, arg := range args {
		switch arg {
		case "-h", "--help", "help", "version", "--version":
			return true
		}
	}
	return false
}
