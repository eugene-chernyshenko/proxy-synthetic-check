# Proxy Performance Testing Tool

A Go-based tool for testing proxy performance by sending HTTP requests through multiple proxies (SOCKS5 and HTTP) in parallel. Provides Prometheus metrics for monitoring latency, success rates, and error distribution.

## Features

- **Multiple Proxy Support**: Test multiple proxies simultaneously in parallel
- **Proxy Types**: Supports SOCKS5 and HTTP proxies
- **Parallel Execution**: Each proxy runs independently with its own goroutine
- **Prometheus Metrics**: Built-in metrics for request tracking and latency analysis
- **Configurable Latency Buckets**: Customize histogram buckets for your use case
- **Error Categorization**: Detailed error tracking (timeout, connection errors, HTTP errors, etc.)

## Installation

```bash
go mod download
go build -o pop-syn-check main.go
```

## Configuration

Configuration is done through a YAML file `proxies.yaml` in the project root.

### Configuration Structure

```yaml
target_url: https://www.cloudflare.com/cdn-cgi/trace
request_interval_ms: 1000
request_timeout: 30
metrics_port: 8080
# Optional: custom latency buckets for histogram
latency_buckets:
  [0.05, 0.1, 0.2, 0.3, 0.4, 0.5, 0.75, 1.0, 1.5, 2.0, 5.0, 10.0, 30.0]
proxies:
  - name: wifi
    type: socks5
    url: socks5://username:password@proxy.example.com:1080
  - name: mobile
    type: http
    url: http://username:password@proxy.example.com:8080
```

### Configuration Fields

- `target_url` (required): Target URL to send requests to
- `request_interval_ms` (required): Interval between requests in milliseconds
- `request_timeout` (required): Request timeout in seconds
- `metrics_port` (required): Port for Prometheus metrics endpoint (default: 8080)
- `latency_buckets` (optional): Custom latency buckets for histogram. If not specified, defaults with better observability in 0.2-2s range are used
- `proxies` (required): Array of proxy configurations
  - `name`: Proxy identifier (used in metrics labels)
  - `type`: Proxy type - `socks5` or `http`
  - `url`: Proxy URL with authentication if needed

### Proxy URL Format

- SOCKS5: `socks5://username:password@host:port` or `socks5://host:port`
- HTTP: `http://username:password@host:port` or `http://host:port`

## Usage

1. Create `proxies.yaml` configuration file (see `proxies.yaml.example` for template)
2. Run the program:

```bash
go run main.go
# or
./pop-syn-check
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

Total number of requests with labels:

- `proxy_type`: Proxy identifier (e.g., "wifi", "mobile")
- `proxy_protocol`: Protocol type ("socks5" or "http")
- `status`: Request status ("success" or "error")
- `error`: Error type (empty for success, or one of: "timeout", "connection_error", "dns_error", "http_404", "http_500", "read_error", etc.)

#### `request_duration_seconds`

Request latency histogram with labels:

- `proxy_type`: Proxy identifier
- `proxy_protocol`: Protocol type

### Example Queries

```promql
# Total requests per proxy
sum(requests_total) by (proxy_type, proxy_protocol)

# Success rate
sum(requests_total{status="success"}) / sum(requests_total)

# Error rate by type
sum(requests_total{status="error"}) by (error)

# 95th percentile latency
histogram_quantile(0.95, sum(rate(request_duration_seconds_bucket[5m])) by (le, proxy_type, proxy_protocol))

# Average latency
rate(request_duration_seconds_sum[5m]) / rate(request_duration_seconds_count[5m])
```

### Default Latency Buckets

If `latency_buckets` is not specified in config, the following buckets are used (optimized for proxy testing):

```
0.05s, 0.1s, 0.2s, 0.3s, 0.4s, 0.5s, 0.75s, 1.0s, 1.5s, 2.0s, 5.0s, 10.0s, 30.0s
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
  - name: proxy1
    type: socks5
    url: socks5://127.0.0.1:1080
```

### Multiple Proxies with Different Types

```yaml
target_url: https://www.cloudflare.com/cdn-cgi/trace
request_interval_ms: 500
request_timeout: 30
metrics_port: 8080
proxies:
  - name: wifi
    type: socks5
    url: socks5://user:pass@proxy1.example.com:1080
  - name: mobile
    type: http
    url: http://user:pass@proxy2.example.com:8080
```

### Custom Latency Buckets

```yaml
target_url: https://example.com
request_interval_ms: 1000
request_timeout: 30
metrics_port: 8080
latency_buckets: [0.1, 0.25, 0.5, 1.0, 2.0, 5.0]
proxies:
  - name: test
    type: socks5
    url: socks5://proxy.example.com:1080
```

## How It Works

1. **Configuration Loading**: Program reads `proxies.yaml` on startup
2. **Metrics Initialization**: Prometheus metrics are initialized with configured latency buckets
3. **Parallel Execution**: Each proxy configuration runs in a separate goroutine
4. **Request Sending**: Each proxy sends requests at the configured interval independently
5. **Metrics Collection**: All requests are tracked with detailed labels for analysis
6. **Metrics Exposure**: Metrics are available via HTTP endpoint for Prometheus scraping

## Error Types

The tool categorizes errors for better observability:

- `timeout`: Request timeout errors
- `connection_error`: Network connection errors (refused, reset, etc.)
- `dns_error`: DNS resolution errors
- `http_<code>`: HTTP errors with status code (e.g., `http_404`, `http_500`)
- `read_error`: Errors reading response body

## Requirements

- Go 1.23 or higher
- Valid `proxies.yaml` configuration file
- At least one proxy configuration

## License

[Specify your license here]
