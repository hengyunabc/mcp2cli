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

## Config

See `examples/weather.json`.

## Usage

```bash
./mcp2cli --config examples/weather.json --help
./mcp2cli --config examples/weather.json tools
./mcp2cli --config examples/weather.json cityInfo --help
./mcp2cli --config examples/weather.json cityInfo --name hk
```

## Wrap as a custom command

```bash
alias weather='mcp2cli --config /path/to/weather.json'
weather --help
weather cityInfo --help
weather cityInfo --name hk
```

Or install a standalone wrapper command:

```bash
chmod +x scripts/install-wrapper.sh
scripts/install-wrapper.sh weather /path/to/weather.json ~/.local/bin
weather --help
```

## Exit codes

1. `0`: success
2. `2`: parameter validation error
3. `3`: MCP connection/protocol error
4. `4`: MCP tool execution error
5. `5`: config error
