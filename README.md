# check_grpc

## Overview

check_grpc is a Nagios-compatible monitoring plugin for gRPC services, similar to `check_http` for HTTP.

It connects to a gRPC endpoint, invokes a unary RPC method using a protocol buffer descriptor file, and returns Nagios-style exit codes based on:

- Response time thresholds (warning / critical)
- Expected response content (substring match)
- Expected gRPC status code

Works with any Nagios-compatible monitoring system, including Mackerel, Zabbix, and Sensu.

## Prerequisites

- **protoc** — required to generate `.pb` descriptor files from `.proto` definitions
- **Go 1.22 or later** — only required if building from source

## Installation

**Download pre-built binary:**

Download the latest binary for your platform from the [Releases](https://github.com/heartbeatsjp/check_grpc/releases) page.

```sh
# Example: Linux amd64
curl -L -o check_grpc.zip https://github.com/heartbeatsjp/check_grpc/releases/latest/download/check_grpc_linux_amd64.zip
unzip check_grpc.zip
chmod +x check_grpc
sudo mv check_grpc /usr/local/bin/
```

**Via `go install`:**

```sh
go install github.com/heartbeatsjp/check_grpc@latest
```

**Build from source:**

```sh
git clone https://github.com/heartbeatsjp/check_grpc.git
cd check_grpc
go build -o check_grpc .
```

## Generating Descriptor Set Files

check_grpc uses protocol buffer descriptor set files (`.pb`) to dynamically invoke gRPC methods at runtime without compiled Go stubs. You must generate a descriptor set file from your service's `.proto` definition.

**Basic usage:**

```sh
protoc -I <proto_path> --descriptor_set_out=service.pb service.proto
```

The `-I` flag (alias of `--proto_path`) tells `protoc` which directory to use as the base for resolving `.proto` files and their imports. If omitted, only the current working directory is searched, which often fails for real projects where `.proto` files live in a separate tree.

**Example using files in this repository:**

```sh
# Run from the repo root
protoc -I testdata --descriptor_set_out=test.pb testdata/test.proto
```

**If your proto imports other proto files** (e.g., `google/protobuf/timestamp.proto`), include the `--include_imports` flag and point `-I` at the directory that contains the imported files:

```sh
# testdata/imports/service.proto contains `import "common.proto";`
protoc -I testdata/imports --include_imports \
  --descriptor_set_out=service.pb testdata/imports/service.proto
```

**Example for the standard gRPC health check service:**

The `grpc/health/v1/health.proto` file is published in the [grpc/grpc-proto](https://github.com/grpc/grpc-proto) repository. Clone it (or vendor the file) and pass its root via `-I`:

```sh
git clone https://github.com/grpc/grpc-proto.git
protoc -I grpc-proto --descriptor_set_out=health.pb grpc/health/v1/health.proto
```

See the `testdata/` directory for example `.proto` and `.pb` files.

## Usage

```
check_grpc -H <host:port> -m <package.Service/Method> -D <descriptor.pb> [options]
```

### Required Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--host` | `-H` | Target gRPC endpoint (`host:port`) |
| `--method` | `-m` | Target gRPC method in `package.Service/Method` format |
| `--descriptor_set_file` | `-D` | Path to the descriptor set file (`*.pb`) |

### Optional Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--secure` | `-S` | `false` | Use TLS/SSL for the gRPC connection (see Limitations) |
| `--timeout` | `-t` | `10` | Timeout for the gRPC call (integer seconds) |
| `--warning` | `-w` | `-1` (disabled) | WARNING threshold for response time (integer seconds) |
| `--critical` | `-c` | `-1` (disabled) | CRITICAL threshold for response time (integer seconds) |
| `--string` | `-s` | `""` | String to expect in the response content |
| `--argument` | `-a` | `""` | Request data in JSON format |
| `--expect_status_code` | `-e` | `0` (OK) | Expected gRPC status code |
| `--metadata` | `-M` | `[]` | Metadata `key:value` pairs (use multiple times for additional headers) |
| `--verbose` | `-v` | `false` | Show debugging details |

## Examples

**Basic gRPC health check:**

```sh
check_grpc -H localhost:50051 -m grpc.health.v1.Health/Check -D health.pb
```

**Health check over TLS:**

```sh
check_grpc -H grpc.example.com:443 -m grpc.health.v1.Health/Check -D health.pb -S
```

**With response time thresholds** (warn at 2s, critical at 5s):

```sh
check_grpc -H localhost:50051 -m grpc.health.v1.Health/Check -D health.pb -w 2 -c 5
```

**With request data and expected response string:**

```sh
check_grpc -H localhost:50051 -m com.example.MyService/MyMethod -D service.pb \
  -a '{"name":"test"}' -s "expected_value"
```

The string passed to `-s` is matched as a substring against the response rendered in **protobuf text-format** — not JSON. For example, a `grpc.health.v1.Health/Check` reply with `status = SERVING` renders roughly as:

```
status:SERVING
```

so to assert the service is healthy you would pass `-s SERVING`, not `-s '"SERVING"'` or `-s '{"status":"SERVING"}'`. If you are unsure what your service actually returns, run once with `-v` first to inspect the text-format output and pick a substring from it.

**With metadata headers:**

```sh
check_grpc -H localhost:50051 -m grpc.health.v1.Health/Check -D health.pb \
  -M "authorization:Bearer token123" -M "x-request-id:abc"
```

## Exit Codes

check_grpc follows the Nagios plugin exit code convention:

| Code | Status | Condition |
|------|--------|-----------|
| 0 | OK | gRPC call succeeded and all checks passed |
| 1 | WARNING | Response time exceeded the warning threshold |
| 2 | CRITICAL | Response time exceeded the critical threshold, expected string not found, unexpected gRPC status code, service/method not found, or connection failure |
| 3 | UNKNOWN | Invalid arguments, bad descriptor file, invalid metadata format, or TLS certificate pool failure |

## Output Format

Output is a single line with the status prefix followed by a descriptive message:

```
OK: gRPC 0 OK - invoke grpc.health.v1.Health/Check, 15.234ms response time
WARNING: Response Time: 2.5s > 2s
CRITICAL: Response Time: 5.234s > 3s
CRITICAL: does not contain expected_string
CRITICAL: gRPC call failed or returned unexpected status code. Expected: 0, Actual: 14 (Response Time: 150ms)
UNKNOWN: Invalid method format: missing '/' in method argument
```

## Development

**Build:**

```sh
go build -o check_grpc .
```

**Run tests:**

```sh
go test ./...
```

**Project structure:**

```
cmd/        CLI flag definitions and Cobra command setup
internal/   Core check logic (gRPC invocation, threshold evaluation)
grpc/       Protocol buffer descriptor parsing and dynamic gRPC helpers
nagios/     Nagios-compatible result formatting and exit codes
testdata/   Sample .proto and .pb files used in tests
```

## Limitations

- Only **Unary RPC** is supported. Client streaming, server streaming, and bidirectional streaming are not supported.
- The `--string` flag performs a substring match against the protobuf text-format representation of the response, not JSON.
- The method format must be `package.Service/Method` (e.g., `grpc.health.v1.Health/Check`).
- The `--secure` flag uses the system certificate pool only. Self-signed or private CA certificates are not currently supported (no `--cacert` option).
- Response-time thresholds (`--timeout`, `--warning`, `--critical`) accept **integer seconds only**; sub-second values such as `0.5` are not supported.
