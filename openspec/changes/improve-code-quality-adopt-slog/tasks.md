## 1. Add ParseLogLevel and remove duplicated helpers

- [x] 1.1 Create `log.go` in the `wstun` package with `func ParseLogLevel(s string) slog.Level` mapping `"debug"`, `"info"`, `"warn"`, `"error"` (case-insensitive) to their `slog.Level` equivalents, defaulting to `slog.LevelInfo`
- [x] 1.2 Create `log_test.go` with table-driven tests for `ParseLogLevel` covering valid levels, empty string, unknown string, and mixed-case input
- [x] 1.3 Remove `validLogLevels` map from `config.go` and update `ServerOptions.Validate()` / `ClientOptions.Validate()` log-level checks to use `ParseLogLevel` (or simply drop the validLogLevels map since slog handles unknown levels gracefully)

## 2. Migrate ServerProtocol to slog

- [x] 2.1 Change `Logger *log.Logger` field to `Logger *slog.Logger` in `ServerProtocol` struct and update `import` block
- [x] 2.2 Add a private `logger()` accessor on `ServerProtocol` that returns `sp.Logger` if non-nil, else `slog.Default()`
- [x] 2.3 Delete the `logf` method from `ServerProtocol`
- [x] 2.4 Replace every `sp.logf(…)` call in `server_protocol.go` with the appropriate `sp.logger().Info/Warn/Error/Debug` call, using structured key-value pairs (`"service"`, `"conn_id"`, `"error"`, etc.)
- [x] 2.5 In `handleControlConnection`, `handleDataConnection`, `RequestConnection`, and `sendToControlConnections`, derive a child logger with `logger.With("service", serviceName)` and use it throughout the method

## 3. Migrate ClientProtocol to slog

- [x] 3.1 Change `Logger *log.Logger` field to `Logger *slog.Logger` in `ClientProtocol` struct and update `import` block
- [x] 3.2 Add a private `logger()` accessor on `ClientProtocol` that returns `cp.Logger` if non-nil, else `slog.Default()`
- [x] 3.3 Delete the `logf` method from `ClientProtocol`
- [x] 3.4 In `NewClientProtocol`, store a child logger with `logger.With("service", serviceName)` so all subsequent log entries include the `"service"` attribute
- [x] 3.5 Replace every `cp.logf(…)` call in `client_protocol.go` with the appropriate slog method using structured key-value pairs (`"conn_id"`, `"error"`, `"attempt"`, `"delay"`, etc.)

## 4. Update cmd/wstun-server

- [x] 4.1 Replace `log.New(…)` logger construction with `slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: wstun.ParseLogLevel(config.LogLevel)}))`
- [x] 4.2 Remove `shouldLog` function and all call-sites; rely on slog level filtering
- [x] 4.3 Remove `logPrefix` variable
- [x] 4.4 Update all `log.Logger` references in `Server` struct and methods to use `*slog.Logger`
- [x] 4.5 Replace `logger.Printf` / `log.Fatalf` calls with `slog` equivalents (`logger.Info`, `logger.Error`, `slog.Error` + `os.Exit(1)`)
- [x] 4.6 In `acceptLoop` and `handleDataConnection`, derive a child logger with `logger.With("service", serviceName)` where applicable
- [x] 4.7 Split `cmd/wstun-server/main.go` into `main.go` (entry point), `server.go` (`Server` struct + `NewServer` + `Start` + `Shutdown` + `acceptLoop` + `handleDataConnection` + helpers)

## 5. Update cmd/wstun-client

- [x] 5.1 Replace `log.New(…)` logger construction with `slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: wstun.ParseLogLevel(config.LogLevel)}))`
- [x] 5.2 Remove `shouldLog` function and all call-sites
- [x] 5.3 Remove `logPrefix` variable
- [x] 5.4 Update all `log.Logger` references in `Client` struct and methods to use `*slog.Logger`
- [x] 5.5 Replace `logger.Printf` / `log.Fatalf` calls with `slog` equivalents
- [x] 5.6 Split `cmd/wstun-client/main.go` into `main.go` (entry point) and `client.go` (`Client` struct + `NewClient` + `Run` + `Shutdown` + helpers)

## 6. Improve variable naming

- [x] 6.1 Rename local `cc` variables in `server_protocol.go` to `ctrlConn` where the variable lifespan exceeds ~5 lines (e.g. in `pingControlConnection`, `sendToControlConnections`)
- [x] 6.2 Remove any remaining `logPrefix` references across the codebase

## 7. Update tests

- [x] 7.1 Update `protocol_test.go`, `edge_cases_test.go`, `connection_health_test.go` and `cmd_integration_test.go` to pass `*slog.Logger` (or `nil`) instead of `*log.Logger` when constructing `ServerProtocol` / `ClientProtocol`
- [x] 7.2 Verify all tests pass with `go test ./...`

## 8. Final verification

- [ ] 8.1 Run `go vet ./...` and `go build ./...` to confirm no compile errors
- [ ] 8.2 Verify log output format (structured key=value) by running server and client against each other manually or via integration test
- [ ] 8.3 Update README.md to note the breaking Logger field change and new structured log format
