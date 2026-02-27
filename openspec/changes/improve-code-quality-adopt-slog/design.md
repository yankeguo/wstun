## Context

wstun is a WebSocket tunnelling library and pair of CLI tools (`wstun-server`, `wstun-client`). The Go module already targets Go 1.25 (`go.mod`), so `log/slog` is available. Today every logging call goes through a `*log.Logger` that is either set on the protocol structs (`ServerProtocol.Logger`, `ClientProtocol.Logger`) or created ad-hoc in the commands (via `log.New`). A thin `logf` helper on each struct adds a hard-coded prefix (`[ServerProtocol]`, `[ClientProtocol]`). There is no structured context and no per-service log attribute on the server side.

Both `cmd/wstun-server/main.go` (398 lines) and `cmd/wstun-client/main.go` (285 lines) mix configuration, lifecycle management, connection handling, and helper functions in a single file. A `shouldLog` function is duplicated in both packages.

## Goals / Non-Goals

**Goals:**

- Replace `*log.Logger` with `*slog.Logger` on `ServerProtocol` and `ClientProtocol` so all library log output is structured.
- Ensure server-side log entries carry a `service` key so operators can filter by service name.
- Map the existing `LogLevel` config string (`debug`, `info`, `warn`, `error`) to `slog.Level` values.
- Eliminate the duplicated `shouldLog` helper and `validLogLevels` map: centralise log-level parsing into the `wstun` package.
- Improve code organisation in `cmd/` by extracting logical units into separate files.
- Improve variable and receiver naming for clarity and Go convention.

**Non-Goals:**

- Changing the on-wire WebSocket protocol (message formats, connection flow).
- Adding new user-facing features beyond structured logging.
- Introducing a third-party logging framework (e.g., zap, zerolog).
- Switching config file format or adding new configuration fields.

## Decisions

### 1. Adopt `*slog.Logger` as the sole logging interface

**Decision**: Replace the `Logger *log.Logger` field on both `ServerProtocol` and `ClientProtocol` with `Logger *slog.Logger`. When `nil`, the structs fall back to `slog.Default()`.

**Alternatives considered**:

- *Keep `*log.Logger` and add a separate slog logger\*: doubles API surface for no benefit.
- _Use an interface (`interface{ Info(…); Error(…) }`)_: adds abstraction that isn't needed when the stdlib already provides `slog`.

**Rationale**: `slog` is the standard structured logger since Go 1.21. The module already requires Go 1.25, so there is zero dependency cost. A pointer-typed field keeps the zero-value usable.

### 2. Remove `logf` helpers, call slog directly

**Decision**: Delete `ServerProtocol.logf` and `ClientProtocol.logf`. Replace each call site with the appropriate `slog` method (`logger.Info`, `logger.Warn`, `logger.Error`, `logger.Debug`) passing structured `slog.Attr` / key-value pairs.

**Rationale**: `logf` exists only to check for nil and prepend a prefix; `slog.Default()` fallback and `slog.With()` eliminate both needs.

### 3. Attach `service` attribute via `slog.With`

**Decision**: On the server-side, whenever a request is authenticated and the `serviceName` is known, derive a child logger with `logger.With("service", serviceName)` and pass it through call chains. On the client-side, `ClientProtocol` stores a child logger created once in `NewClientProtocol` with `logger.With("service", serviceName)`.

**Alternatives considered**:

- _Pass `serviceName` to every log call_: repetitive, easy to forget.
- _Use context-based slog handler_: over-engineered for this use case.

**Rationale**: `slog.With` produces a cheap child that automatically includes the attribute in every entry—zero per-call overhead and impossible to forget.

### 4. Centralise log-level parsing in `wstun` package

**Decision**: Add a helper `func ParseLogLevel(s string) slog.Level` in a new file `log.go` in the `wstun` package. Both CLI commands use this to build `slog.HandlerOptions{Level: level}`. Remove the duplicated `shouldLog` function and `validLogLevels` map from both commands.

**Rationale**: Eliminates duplication; the mapping is trivial but belongs in the shared package so tests cover it once.

### 5. Split `cmd/` files by responsibility

**Decision**: Reorganise each command package:

`cmd/wstun-server/`:
| File | Contents |
|------|----------|
| `main.go` | `main()` entry point only |
| `server.go` | `Server` struct, `NewServer`, `Start`, `Shutdown` |
| `handler.go` | `acceptLoop`, `handleDataConnection` |

`cmd/wstun-client/`:
| File | Contents |
|------|----------|
| `main.go` | `main()` entry point only |
| `client.go` | `Client` struct, `NewClient`, `Run`, `Shutdown` |

**Alternatives considered**:

- _Leave single files_: they are approaching 400 lines and will grow further.
- _Move into `internal/` packages_: unnecessary indirection for simple CLI tools.

**Rationale**: Smaller files are easier to navigate and review; aligns with Go convention of one primary type per file.

### 6. Improve variable naming

**Decision**: Rename key identifiers for clarity:

- `sp` receiver on `ServerProtocol` → keep `sp` (short receiver is idiomatic Go)
- `cc` (control connection wrapper) → `ctrlConn` in local variables where lifespan > 5 lines
- `cp` receiver on `ClientProtocol` → keep `cp`
- `logPrefix` removed entirely (replaced by slog attributes)
- `pendingConns`, `controlConns` field names are already clear — no change

**Rationale**: Go style favours short receivers but longer local names when scope is large. The refactor targets only locals where brevity hurts readability.

## Risks / Trade-offs

- **Breaking API** → Mitigated by clear migration: callers replace `log.New(…)` with `slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))`. Both CLI commands are in-tree and updated together.
- **Log output format changes** → slog `TextHandler` produces `key=value` lines instead of the old prefix format. Operators with log parsers must adjust. Mitigated: document in README.
- **Test churn** → Tests that pass a `*log.Logger` must change. Mitigated: most tests use `nil` logger or `log.Default()`; mechanical replacement.
- **File split merge conflicts** → If other changes are in flight against `cmd/wstun-server/main.go`, the split will conflict. Mitigated: no other active changes touch those files currently.
