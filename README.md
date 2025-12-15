# Proxy Synthetic Check

A Go-based tool for testing proxy performance by sending HTTP requests through multiple proxies (SOCKS5 and HTTP) in parallel. Provides Prometheus metrics for monitoring latency, success rates, and error distribution.

## Features

- **Multiple Proxy Support**: Test multiple proxies simultaneously in parallel
- **Proxy Protocols**: Supports SOCKS5 and HTTP proxies
- **Parallel Execution**: Each proxy runs independently with its own goroutine
- **Prometheus Metrics**: Built-in metrics for request tracking and latency analysis
- **Configurable Latency Buckets**: Customize histogram buckets for your use case
- **Error Categorization**: Detailed error tracking (timeout, connection errors, HTTP errors, etc.)
- **Custom Metric Labels**: Add custom labels to metrics for better filtering and grouping
- **Modular Architecture**: Clean, maintainable codebase with separated concerns

## Project Structure

```
.
├── cmd/
│   └── proxy-synthetic-check/
│       └── main.go          # Application entry point
├── internal/
│   ├── config/              # Configuration loading and parsing
│   ├── metrics/             # Prometheus metrics initialization
│   ├── proxy/               # Proxy transport creation
│   ├── request/             # HTTP request handling and error categorization
│   └── runner/              # Proxy runner orchestration
├── proxies.yaml             # Configuration file (create from example)
├── proxies.yaml.example     # Example configuration
└── go.mod
```

## Installation

### Prerequisites

- Go 1.23 or higher

### Build from Source

```bash
# Clone the repository
git clone <repository-url>
cd proxy-synthetic-check

# Download dependencies
go mod download

# Build the binary
go build -o proxy-synthetic-check ./cmd/proxy-synthetic-check

# Run the application
./proxy-synthetic-check
```

### Quick Start with Go Run

```bash
go run ./cmd/proxy-synthetic-check/main.go
```

## Configuration

Configuration is done through a YAML file `proxies.yaml` in the project root. Copy `proxies.yaml.example` to `proxies.yaml` and modify it according to your needs.

### Configuration Structure

```yaml
target_url: https://www.cloudflare.com/cdn-cgi/trace
request_interval_ms: 1000
request_timeout: 30
metrics_port: 8080
# Optional: custom latency buckets for histogram
latency_buckets: [0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 0.75, 1.0, 1.5, 2.0, 5.0, 10.0]
proxies:
  - protocol: socks5
    proxy: username:password@proxy.example.com:1080
    labels:
      name: wifi
      region: us
  - protocol: http
    proxy: username:password@proxy2.example.com:8080
    labels:
      name: mobile
      region: eu
```

### Configuration Fields

#### Global Settings

- `target_url` (required): Target URL to send requests to
- `request_interval_ms` (required): Interval between requests in milliseconds
- `request_timeout` (required): Request timeout in seconds
- `metrics_port` (optional): Port for Prometheus metrics endpoint (default: 8080)
- `latency_buckets` (optional): Custom latency buckets for histogram. If not specified, defaults with better observability in 0.2-2s range are used

#### Proxy Configuration

Each proxy in the `proxies` array requires:

- `protocol` (required): Proxy protocol - `socks5` or `http`
- `proxy` (required): Proxy address in format `username:password@host:port` or `host:port` (without scheme)
- `labels` (optional): Custom labels as key-value pairs for metrics filtering

### Proxy Address Format

The `proxy` field should contain only the address and credentials, **without** the protocol scheme:

- **SOCKS5**: `username:password@proxy.example.com:1080` or `proxy.example.com:1080`
- **HTTP**: `username:password@proxy.example.com:8080` or `proxy.example.com:8080`

The protocol scheme (socks5:// or http://) is automatically added based on the `protocol` field.

### Custom Labels

Custom labels allow you to add metadata to your metrics for better filtering and grouping:

```yaml
proxies:
  - protocol: socks5
    proxy: user:pass@proxy1.example.com:1080
    labels:
      name: wifi
      region: us
      provider: provider-a
  - protocol: socks5
    proxy: user:pass@proxy2.example.com:1080
    labels:
      name: mobile
      region: eu
      provider: provider-b
```

All label keys from all proxies are automatically collected and added to metrics. If a proxy doesn't have a specific label, an empty string is used for that label value.

## Usage

1. Create `proxies.yaml` configuration file (see `proxies.yaml.example` for template)
2. Run the program:

```bash
./proxy-synthetic-check
```

The program will:

- Load configuration from `proxies.yaml`
- Start Prometheus metrics server on configured port
- Begin sending requests through all configured proxies in parallel
- Run indefinitely until interrupted (Ctrl+C)

## Prometheus Metrics

Metrics are exposed at `http://localhost:<metrics_port>/metrics`

### Available Metrics

#### `requests_total`

Total number of requests (counter) with labels:

- `proxy_id`: Sequential proxy identifier (proxy_1, proxy_2, ...)
- `proxy_protocol`: Protocol type ("socks5" or "http")
- `status`: Request status ("success" or "error")
- `error`: Error type (empty for success, or one of: "timeout", "connection_error", "dns_error", "http_404", "http_500", "read_error", "unknown_error")
- `...custom_labels...`: All custom labels defined in proxy configuration

#### `request_duration_seconds`

Request latency histogram with labels:

- `proxy_id`: Sequential proxy identifier
- `proxy_protocol`: Protocol type
- `...custom_labels...`: All custom labels defined in proxy configuration

### Example Queries

```promql
# Total requests per proxy
sum(requests_total) by (proxy_id, proxy_protocol)

# Total requests by custom label (e.g., region)
sum(requests_total) by (region, proxy_protocol)

# Success rate
sum(requests_total{status="success"}) / sum(requests_total)

# Success rate by region
sum(requests_total{status="success"} by (region)) / sum(requests_total by (region))

# Error rate by type
sum(requests_total{status="error"}) by (error)

# 95th percentile latency
histogram_quantile(0.95, sum(rate(request_duration_seconds_bucket[5m])) by (le, proxy_id, proxy_protocol))

# Average latency by region
rate(request_duration_seconds_sum[5m]) / rate(request_duration_seconds_count[5m]) by (region)

# Compare latency between different providers
histogram_quantile(0.95, sum(rate(request_duration_seconds_bucket[5m])) by (le, provider))
```

### Default Latency Buckets

If `latency_buckets` is not specified in config, the following buckets are used (optimized for proxy testing):

```
0.05s, 0.1s, 0.2s, 0.3s, 0.4s, 0.5s, 0.75s, 1.0s, 1.5s, 2.0s, 5.0s, 10.0s
```

These buckets provide better observability in the 0.2-2s range where most proxy responses fall.

## Examples

### Basic Configuration

```yaml
target_url: https://httpbin.org/ip
request_interval_ms: 1000
request_timeout: 30
metrics_port: 8080
proxies:
  - protocol: socks5
    proxy: 127.0.0.1:1080
```

### Multiple Proxies with Custom Labels

```yaml
target_url: https://www.cloudflare.com/cdn-cgi/trace
request_interval_ms: 500
request_timeout: 30
metrics_port: 8080
proxies:
  - protocol: socks5
    proxy: user:pass@proxy1.example.com:1080
    labels:
      name: wifi
      region: us
      provider: provider-a
  - protocol: http
    proxy: user:pass@proxy2.example.com:8080
    labels:
      name: mobile
      region: eu
      provider: provider-b
```

### Proxies with Authentication

```yaml
target_url: https://example.com
request_interval_ms: 1000
request_timeout: 30
metrics_port: 8080
proxies:
  - protocol: socks5
    proxy: username:password@proxy.example.com:1080
    labels:
      name: authenticated-proxy
```

### Custom Latency Buckets

```yaml
target_url: https://example.com
request_interval_ms: 1000
request_timeout: 30
metrics_port: 8080
latency_buckets: [0.1, 0.25, 0.5, 1.0, 2.0, 5.0]
proxies:
  - protocol: socks5
    proxy: proxy.example.com:1080
    labels:
      name: test-proxy
```

## How It Works

1. **Configuration Loading**: Program reads `proxies.yaml` on startup using the `config` package
2. **Metrics Initialization**: Prometheus metrics are initialized with configured latency buckets and collected label keys
3. **Parallel Execution**: Each proxy configuration runs in a separate goroutine via the `runner` package
4. **Request Sending**: Each proxy sends requests at the configured interval independently
5. **Metrics Collection**: All requests are tracked with detailed labels (proxy_id, protocol, custom labels, status, error)
6. **Metrics Exposure**: Metrics are available via HTTP endpoint for Prometheus scraping

## Error Types

The tool categorizes errors for better observability:

- `timeout`: Request timeout errors
- `connection_error`: Network connection errors (refused, reset, EOF, etc.)
- `dns_error`: DNS resolution errors
- `http_<code>`: HTTP errors with status code (e.g., `http_404`, `http_500`)
- `read_error`: Errors reading response body
- `unknown_error`: Unclassified errors

## Architecture

The application follows a modular architecture:

- **`cmd/proxy-synthetic-check`**: Entry point that orchestrates all components
- **`internal/config`**: Configuration structures and YAML parsing
- **`internal/metrics`**: Prometheus metrics initialization and management
- **`internal/proxy`**: Proxy transport creation for SOCKS5 and HTTP
- **`internal/request`**: HTTP request execution and error categorization
- **`internal/runner`**: Proxy runner that manages request intervals and lifecycle

This architecture provides:

- Clear separation of concerns
- Easy testing of individual components
- Maintainable and extensible codebase
- No global state (metrics are encapsulated in a struct)

## Requirements

- Go 1.23 or higher
- Valid `proxies.yaml` configuration file
- At least one proxy configuration

## Development

### Running Tests

```bash
go test ./...
```

### Code Structure

- All application code is in the `internal/` directory
- The `cmd/` directory contains only the main entry point
- Each internal package has a single, focused responsibility

## License

MIT
