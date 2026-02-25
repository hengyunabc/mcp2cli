# mcp2cli Implementation Plan

## 1. Goal

Build `mcp2cli` in Go, based on `github.com/mark3labs/mcp-go`, so a configured remote MCP server can be auto-exposed as CLI subcommands without custom code.

Required UX:

1. `mcp2cli --config weather.json --help` shows server info and dynamic tool list.
2. `mcp2cli --config weather.json cityInfo --help` shows detailed input and return schema.
3. `mcp2cli --config weather.json cityInfo --name hk` calls tool and prints JSON result.
4. Easy command wrapping in shell, for example aliasing to `weather`.

## 2. Locked Decisions

1. Language: Go
2. MCP client library: `mark3labs/mcp-go`
3. Transport in v1: remote MCP over HTTP (`streamable_http`)
4. Auth in v1: Bearer token header
5. Scope in v1: single server per config
6. Tool discovery: fetch at startup
7. Validation policy: strict local parameter validation
8. Output policy: JSON by default

## 3. External Contract

### 3.1 CLI shape

`mcp2cli --config <file.json> [toolName] [flags]`

Built-in commands:

1. `help` / `--help`
2. `version`
3. `tools` (dynamic tool summary)

Dynamic commands:

1. One subcommand per MCP tool, for example `cityInfo`.

### 3.2 Exit codes

1. `0`: success
2. `2`: parameter validation error
3. `3`: MCP connection/protocol error
4. `4`: MCP tool execution error
5. `5`: config error

## 4. Config File (JSON)

```json
{
  "name": "weather",
  "server": {
    "url": "https://example.com/mcp",
    "transport": "streamable_http",
    "timeout_seconds": 30
  },
  "auth": {
    "type": "bearer",
    "token": "${WEATHER_MCP_TOKEN}"
  },
  "cli": {
    "description": "Weather MCP CLI",
    "include_return_schema_in_help": true,
    "pretty_json": false
  }
}
```

## 5. Package Layout

1. `cmd/mcp2cli/main.go`: entrypoint
2. `internal/config`: load/validate config and env expansion
3. `internal/mcpclient`: wrapper around `mcp-go` calls (`initialize`, `tools/list`, `tools/call`)
4. `internal/schema`: schema-to-flags conversion and argument parsing
5. `internal/cli`: dynamic Cobra command generation and help rendering
6. `internal/render`: JSON output and error mapping

## 6. Help Requirements

### 6.1 Root help

Must include:

1. server name/url/transport/auth type
2. dynamic tool list (name + description)
3. quick examples

### 6.2 Tool help (`weather cityInfo --help`)

Must include:

1. tool name and description
2. parameter list with type/required/default/enum/description
3. return schema section (when provided by server)
4. request/response example

## 7. Test Plan

1. Unit: config parsing and `${ENV}` expansion
2. Unit: schema mapping and strict validation
3. Unit: help text includes input and return schema sections
4. Integration with mock MCP HTTP server:
   1. root help shows dynamic tools
   2. `tool --help` shows detailed fields and return schema
   3. `tool` invocation returns JSON
   4. missing required param fails
   5. auth failure path works

## 8. Milestones and Progress

| Milestone | Status | Deliverables |
| --- | --- | --- |
| A. Foundation | Done | module setup, config loader, `mcp-go` connectivity (`initialize`, `tools/list`) |
| B. Dynamic CLI | Done | dynamic subcommands and root help tool listing |
| C. Tool help + validation | Done | tool flag generation, strict validation, detailed `tool --help` |
| D. Invocation + output | Done | `tools/call`, result rendering, exit code mapping |
| E. Tests + docs | Done | unit/integration tests, examples, README updates |

## 9. Progress Update Log

### 2026-02-08

1. Added initial implementation plan to `docs/plan.md`.
2. Set milestone tracker format for incremental updates after each completed milestone.
3. Started milestone A (foundation setup).
4. Completed milestone A:
   1. Added executable entrypoint `cmd/mcp2cli/main.go`.
   2. Implemented config loading/validation/env expansion in `internal/config`.
   3. Implemented MCP Streamable HTTP client wrapper using `mcp-go` in `internal/mcpclient`.
   4. Added application error code layer and base JSON renderer.
   5. Verified compile via `go test ./...`.
5. Completed milestone B:
   1. Added dynamic root command builder in `internal/cli`.
   2. Implemented runtime tool-to-subcommand generation from MCP `tools/list`.
   3. Added root help sections with server metadata and discovered tool summary.
6. Completed milestone C:
   1. Added schema parser/mapping in `internal/schema`.
   2. Implemented schema-driven flag registration and strict local validation.
   3. Implemented detailed `tool --help` with parameter and return schema sections.
7. Completed milestone D:
   1. Implemented dynamic tool invocation path (`tools/call`).
   2. Implemented default JSON output rendering.
   3. Implemented stable error-to-exit-code mapping.
8. Completed milestone E:
   1. Added config unit tests (`internal/config/config_test.go`).
   2. Added schema unit tests (`internal/schema/schema_test.go`).
   3. Added integration tests with `mcp-go` Streamable HTTP test server (`internal/app/run_integration_test.go`).
   4. Added user documentation (`README.md`) and sample config (`examples/weather.json`).
   5. Verified full test pass via `go test ./...`.
9. Additional updates by request:
   1. Added `Makefile` with `build/test/fmt/run/clean` targets.
   2. Added wrapper installer script `scripts/install-wrapper.sh`.
   3. Updated `README.md` with build and wrapper installation usage.
10. Local installation UX enhancements:
   1. Added `mcp2cli install/list/remove` bootstrap commands.
   2. Added managed home layout `~/.mcp2cli/bin` and `~/.mcp2cli/configs`.
   3. Added argv0-based default config discovery for installed wrapper commands.
   4. Added PATH guidance with optional shell rc update flow.
   5. Added install manager + app/config integration tests.
