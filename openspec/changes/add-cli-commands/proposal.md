# Proposal: Add CLI Commands

## Why
The wstun library needs executable command-line tools to function as both a server and client. Users need a simple way to run wstun server and client instances with configurable options loaded from files.

## What Changes
- Add `cmd/wstun-server/` directory with main.go implementing the server CLI
- Add `cmd/wstun-client/` directory with main.go implementing the client CLI
- Add `-c` flag to both commands for specifying configuration file path
- Create `ServerOptions` struct to represent server configuration
- Create `ClientOptions` struct to represent client configuration
- Implement configuration file loading (YAML format)
- Implement configuration validation for both options structs

## Impact
- Affected specs: cli-server, cli-client, configuration (new capabilities)
- Affected code: New `cmd/` directory with two command implementations
- Dependencies: May require YAML parsing library (e.g., `gopkg.in/yaml.v3`)

