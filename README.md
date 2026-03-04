# mcp2cli

`mcp2cli` 用于把远端 MCP Server 暴露的工具，动态转换成本地命令行子命令。

它在启动时通过 `tools/list` 自动发现工具，并根据工具的 JSON Schema 生成参数与帮助信息。

## 功能特性

1. 通过 JSON 配置连接 MCP Streamable HTTP 服务端。
2. 启动时自动发现工具并生成二级命令（例如 `weather cityInfo`）。
3. 动态生成详细帮助，包括参数、输入 Schema、返回 Schema。
4. 根据工具输入 Schema 做本地参数校验（含必填、类型、枚举）。
5. 调用 MCP 工具并输出 JSON 结果。
6. 支持安装 wrapper 命令（软链接）并自动关联配置文件。

## 安装

### 方式 1：源码构建

```bash
go build -o mcp2cli ./cmd/mcp2cli
```

或使用 Makefile：

```bash
make build
make test
```

### 方式 2：从 GitHub Release 安装

安装最新版：

```bash
curl -fsSL https://raw.githubusercontent.com/hengyunabc/mcp2cli/refs/heads/master/scripts/install-release.sh | bash
```

安装指定版本：

```bash
curl -fsSL https://raw.githubusercontent.com/hengyunabc/mcp2cli/refs/heads/master/scripts/install-release.sh | VERSION=v0.1.0 bash
```

安装脚本支持的环境变量：`OWNER`、`REPO`、`BINARY`、`VERSION`、`INSTALL_DIR`。

## 快速开始

### 方式 1：直接通过 `--config` 调用

示例配置见 `examples/weather.json`。

```bash
./mcp2cli --config examples/weather.json --help
./mcp2cli --config examples/weather.json tools
./mcp2cli --config examples/weather.json cityInfo --help
./mcp2cli --config examples/weather.json cityInfo --name hk
```

### 方式 2：安装 wrapper 命令（推荐）

```bash
mcp2cli install weather --url https://example.com/mcp --token <token>
```

安装后会生成：

1. `~/.mcp2cli/bin/weather`（指向 `mcp2cli` 的软链接）
2. `~/.mcp2cli/configs/weather.json`

然后把 `~/.mcp2cli/bin` 加入 PATH：

```bash
export PATH="$HOME/.mcp2cli/bin:$PATH"
```

之后可直接调用：

```bash
weather --help
weather tools
weather cityInfo --help
weather cityInfo --name hk
```

当以 `weather` 形式运行时，会自动读取 `~/.mcp2cli/configs/weather.json`。

## 管理已安装命令

导入已有配置文件：

```bash
mcp2cli install weather --from-config /path/to/weather.json
```

查看已安装 wrapper：

```bash
mcp2cli list
```

删除 wrapper（默认同时删除配置）：

```bash
mcp2cli remove weather
```

仅删除 wrapper，保留配置：

```bash
mcp2cli remove weather --keep-config
```

## 配置文件说明

完整示例：

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

字段说明：

1. `server.url`：MCP 服务地址（必填）。
2. `server.transport`：目前仅支持 `streamable_http`。
3. `server.timeout_seconds`：超时秒数，默认 `30`。
4. `auth.type`：可选，支持 `bearer`。
5. `auth.token`：当 `auth.type=bearer` 时必填。
6. `cli.description`：CLI 描述文本。
7. `cli.include_return_schema_in_help`：是否在帮助中显示返回 Schema，默认 `true`。
8. `cli.pretty_json`：是否美化 JSON 输出，默认 `false`。

配置文件支持环境变量展开（例如 `"${WEATHER_MCP_TOKEN}"`）。

## 常用命令

```bash
mcp2cli --help
mcp2cli version
mcp2cli install --help
mcp2cli list
mcp2cli remove --help
```

如果已安装 wrapper（如 `weather`）：

```bash
weather --help
weather tools
weather <toolName> --help
weather <toolName> --arg value
```

## 环境变量

1. `MCP2CLI_CONFIG`：可指定默认配置文件路径（等价于 `--config`）。
2. `MCP2CLI_HOME`：自定义安装根目录（默认 `~/.mcp2cli`）。

## 退出码

1. `0`：成功
2. `1`：通用错误
3. `2`：参数校验错误
4. `3`：连接或协议错误
5. `4`：工具执行错误
6. `5`：配置错误
7. `6`：内部错误
