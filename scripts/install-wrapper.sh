#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 2 || $# -gt 3 ]]; then
  echo "Usage: $0 <command-name> <config-path> [install-dir]" >&2
  echo "Example: $0 weather /path/to/weather.json ~/.local/bin" >&2
  exit 1
fi

command_name="$1"
config_path="$2"
install_dir="${3:-$HOME/.local/bin}"

if [[ ! -f "$config_path" ]]; then
  echo "Config file does not exist: $config_path" >&2
  exit 1
fi

mkdir -p "$install_dir"
target="$install_dir/$command_name"

cat >"$target" <<EOF
#!/usr/bin/env bash
set -euo pipefail
exec mcp2cli --config "$config_path" "\$@"
EOF

chmod +x "$target"

echo "Wrapper installed: $target"
echo "Run: $command_name --help"
echo "If needed, add to PATH:"
echo "  export PATH=\"$install_dir:\$PATH\""

