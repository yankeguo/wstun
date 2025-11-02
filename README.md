# wstun - WebSocket Tunnel

A simple and efficient WebSocket tunneling library and command-line tools written in Go.

## Features

- **Simple CLI tools**: Separate server and client binaries for clear separation of concerns
- **File-based configuration**: YAML configuration files for easy deployment and automation
- **Minimal dependencies**: Uses Go standard library and only essential external packages
- **Graceful shutdown**: Handles SIGINT and SIGTERM signals for clean connection termination

## Installation

### Build from source

```bash
# Build server
go build -o wstun-server ./cmd/wstun-server

# Build client
go build -o wstun-client ./cmd/wstun-client

# Or build both
go build ./cmd/wstun-server ./cmd/wstun-client
```

## Usage

### Server

Start the WebSocket tunnel server:

```bash
wstun-server -c /path/to/server.yaml
```

#### Server Configuration

Create a YAML configuration file with the following options:

```yaml
# listen_addr (required)
# The address and port to bind the server to
listen_addr: ":8080"

# tls_cert (optional)
# Path to the TLS certificate file
# If set, tls_key must also be set
# tls_cert: "/path/to/cert.pem"

# tls_key (optional)
# Path to the TLS private key file
# If set, tls_cert must also be set
# tls_key: "/path/to/key.pem"

# log_level (optional, default: "info")
# Logging verbosity level
# Valid values: "debug", "info", "warn", "error"
log_level: "info"

# max_connections (optional, default: 0 = unlimited)
# Maximum number of concurrent connections
max_connections: 0
```

See `examples/server.yaml` for a complete example.

### Client

Start the WebSocket tunnel client:

```bash
wstun-client -c /path/to/client.yaml
```

#### Client Configuration

Create a YAML configuration file with the following options:

```yaml
# server_url (required)
# The WebSocket server URL to connect to
server_url: "ws://localhost:8080"

# local_addr (required)
# The local address to listen on for incoming connections
local_addr: ":9090"

# tls_insecure (optional, default: false)
# Skip TLS certificate verification
# WARNING: Only use this for testing!
tls_insecure: false

# reconnect_interval (optional, default: 5s)
# Time to wait between reconnection attempts
reconnect_interval: 5s

# max_reconnect_attempts (optional, default: 0 = unlimited)
# Maximum number of reconnection attempts
max_reconnect_attempts: 0

# log_level (optional, default: "info")
# Logging verbosity level
# Valid values: "debug", "info", "warn", "error"
log_level: "info"
```

See `examples/client.yaml` for a complete example.

## Examples

### Basic Usage

1. Create a server configuration file `server.yaml`:

```yaml
listen_addr: ":8080"
log_level: "info"
```

2. Start the server:

```bash
wstun-server -c server.yaml
```

3. Create a client configuration file `client.yaml`:

```yaml
server_url: "ws://localhost:8080"
local_addr: ":9090"
log_level: "info"
```

4. Start the client:

```bash
wstun-client -c client.yaml
```

### TLS/SSL Support

1. Generate or obtain TLS certificates (example using self-signed cert):

```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes
```

2. Configure server with TLS:

```yaml
listen_addr: ":8443"
tls_cert: "cert.pem"
tls_key: "key.pem"
log_level: "info"
```

3. Update client to use `wss://` protocol:

```yaml
server_url: "wss://localhost:8443"
local_addr: ":9090"
# Only for self-signed certs in testing!
tls_insecure: true
log_level: "info"
```

## Library Usage

You can also use `wstun` as a library in your Go projects:

```go
package main

import (
    "log"
    "cnb.cool/yankeguo/wstun"
)

func main() {
    // Load server configuration
    serverConfig, err := wstun.LoadServerConfig("server.yaml")
    if err != nil {
        log.Fatal(err)
    }
    
    // Use serverConfig...
    
    // Load client configuration
    clientConfig, err := wstun.LoadClientConfig("client.yaml")
    if err != nil {
        log.Fatal(err)
    }
    
    // Use clientConfig...
}
```

## Development

### Running Tests

```bash
go test -v ./...
```

### Running Tests with Race Detection

```bash
go test -race ./...
```

## License

See LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

