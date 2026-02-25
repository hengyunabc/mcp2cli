#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "Usage: $0 <command-name> <config-path>" >&2
  echo "Example: $0 weather /path/to/weather.json" >&2
  exit 1
fi

command_name="$1"
config_path="$2"

if [[ ! -f "$config_path" ]]; then
  echo "Config file does not exist: $config_path" >&2
  exit 1
fi

echo "[deprecated] use: mcp2cli install <name> --from-config <file>"
exec mcp2cli install "$command_name" --from-config "$config_path"
