# mcp2cli

`mcp2cli` converts MCP tools from a remote MCP server into dynamic CLI subcommands.

## Features

1. Reads MCP server config from JSON.
2. Connects to remote MCP Streamable HTTP server.
3. Discovers tools at startup (`tools/list`).
4. Exposes each tool as a second-level command.
5. Supports detailed dynamic help:
   1. `weather --help`
   2. `weather cityInfo --help`
6. Performs strict local parameter validation from tool schema.
7. Calls MCP tools and prints JSON output.

## Build

```bash
go build -o mcp2cli ./cmd/mcp2cli
```

Or use `Makefile`:

```bash
make build
make test
```

## Install binary from GitHub Release

Install latest release:

```bash
curl -fsSL https://raw.githubusercontent.com/hengyunabc/mcp2cli/master/scripts/install-release.sh | bash
```

Install a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/hengyunabc/mcp2cli/master/scripts/install-release.sh | VERSION=v0.1.0 bash
```

The installer can be customized via env vars: `OWNER`, `REPO`, `BINARY`, `VERSION`, `INSTALL_DIR`.

## Config

See `examples/weather.json`.

## Usage

```bash
./mcp2cli --config examples/weather.json --help
./mcp2cli --config examples/weather.json tools
./mcp2cli --config examples/weather.json cityInfo --help
./mcp2cli --config examples/weather.json cityInfo --name hk
```

## Install wrapper commands (recommended)

```bash
mcp2cli install weather --url https://example.com/mcp --token <token>
```

This installs:

1. `~/.mcp2cli/bin/weather` (symlink to `mcp2cli`)
2. `~/.mcp2cli/configs/weather.json`

Then add `~/.mcp2cli/bin` to PATH:

```bash
export PATH="$HOME/.mcp2cli/bin:$PATH"
```

After that:

```bash
weather --help
weather cityInfo --help
weather cityInfo --name hk
```

When invoked as `weather`, `mcp2cli` uses `os.Args[0]` to automatically load `~/.mcp2cli/configs/weather.json`.

Import from an existing config file:

```bash
mcp2cli install weather --from-config /path/to/weather.json
```

Manage installed wrappers:

```bash
mcp2cli list
mcp2cli remove weather
```

## Exit codes

1. `0`: success
2. `2`: parameter validation error
3. `3`: MCP connection/protocol error
4. `4`: MCP tool execution error
5. `5`: config error
