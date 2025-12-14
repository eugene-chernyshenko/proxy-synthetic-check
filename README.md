# HTTP requests through SOCKS5 proxy on Go

A Go program for sending HTTP requests through SOCKS5 proxy at specified intervals. Requests are sent in parallel using goroutines.

## Installation

```bash
go mod download
```

## Configuration

All configuration is done through environment variables or `.env` file (for dev environment).

### Required environment variables:

- `SOCKS5_PROXY` - SOCKS5 proxy URL (required)
- `TARGET_URL` - Target URL to send requests to (required)
- `REQUEST_INTERVAL_MS` - Interval between requests in milliseconds (required)

### Optional environment variables:

- `REQUEST_TIMEOUT` - Request timeout in seconds (default: 30)

## Usage

### Option 1: Environment variables

```bash
export SOCKS5_PROXY=socks5://127.0.0.1:1080
export TARGET_URL=https://httpbin.org/ip
export REQUEST_INTERVAL_MS=1000
export REQUEST_TIMEOUT=30
go run main.go
```

### Option 2: .env file (for dev)

Create `.env` file in project root:

```env
SOCKS5_PROXY=socks5://127.0.0.1:1080
TARGET_URL=https://httpbin.org/ip
REQUEST_INTERVAL_MS=1000
REQUEST_TIMEOUT=30
```

Then run:

```bash
go run main.go
```

## Examples

### Basic usage

```env
SOCKS5_PROXY=socks5://127.0.0.1:1080
TARGET_URL=https://httpbin.org/ip
REQUEST_INTERVAL_MS=500
```

### With proxy authentication

```env
SOCKS5_PROXY=socks5://username:password@proxy.example.com:1080
TARGET_URL=https://example.com/api/endpoint
REQUEST_INTERVAL_MS=1000
REQUEST_TIMEOUT=60
```

### High frequency requests (100ms interval)

```env
SOCKS5_PROXY=socks5://127.0.0.1:1080
TARGET_URL=https://api.example.com/check
REQUEST_INTERVAL_MS=100
REQUEST_TIMEOUT=10
```

## Proxy URL format

- Without authentication: `socks5://host:port` or `host:port`
- With authentication: `socks5://username:password@host:port`

## How it works

- The program sends requests at specified intervals using a ticker
- Each request runs in a separate goroutine, so requests are executed in parallel
- If a request takes longer than the interval, multiple requests will run simultaneously
- The program runs indefinitely until interrupted (Ctrl+C)

## Build

To create an executable:

```bash
go build -o http-proxy main.go
```

Then run:

```bash
./http-proxy
```

## Requirements

- SOCKS5 proxy is required
- All required environment variables must be set
