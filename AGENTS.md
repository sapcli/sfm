# AGENTS.md

## Project

**sfm** — CLI for managing SAP for Me (SAP launchpad) users. Go 1.26, cobra.

Module: `github.com/sapcli/sfm`
Binary: `sfm` (entrypoint at `cmd/sfm/main.go`)

## Directory structure

```
.                   — library package (client.go, sso.go, useradmin.go, partner.go, cookies.go, etc.)
cmd/sfm/            main.go        — CLI entrypoint
cmd/sfm/internal/   config.go      — shared CLI helpers (Config, MustClient, Print)
cmd/sfm/config/     config.go      — `config` subcommand (set/get/unset)
cmd/sfm/user/       — `user` subcommand
cmd/sfm/partneruser/ — `partneruser` subcommand
```

## Commands

```sh
go build -o sfm ./cmd/sfm              # build binary
go build ./...                        # verify compilation
go vet ./...                          # vet
go test -race ./...                   # full test suite
go test -run TestName ./...           # single test
```

CI runs: `build → vet → test -race`

## Environment

| Variable / Flag | Description |
|---|---|
| `SAP_ADMIN_USERNAME` / `--username` | S-User ID (starts with `S` + digits, e.g. `S1234567890`) |
| `SAP_ADMIN_PASSWORD` / `--password` | Corresponding password |
| `HTTP_LOG_LEVEL` / `--http-log-level` | `debug\|info\|warn\|error` |
| `--timeout` | Request timeout (default `1m30s`) |
| `--debug-body-max` | Max body bytes to log (default `2048`) |
| `-o` / `--output` | Output format: `json\|text\|table` (default `json`) |

Credentials are resolved in order (highest precedence first):
1. `--username` / `--password` CLI flags
2. `SAP_ADMIN_USERNAME` / `SAP_ADMIN_PASSWORD` env vars
3. Config file (saved via `sfm config set`)

The three sources are combined in `PersistentPreRunE` in the root cobra command. If flags are empty, env vars are checked; if both are empty, the config file is read.

## Architecture

- **Root package `sfm`** is the library: `Client` (functional options), `SSO` (SAML/Gigya auth flow), `UserAdmin` (OData CRUD via launchpad.support.sap.com), `PartnerUser` (OData CRUD via partnermanagemyusers.cfapps...).
- **`cmd/sfm/`** is the CLI crust. `MustClient()` constructs an authenticated client; `Print()` formats output (json/text/table). Both read global pointer vars set by cobra's `init()`.
- **Auth path**: SID + password → SAML redirect chain → optional Gigya/CDC flow → cookie-based session. The SSO code parses HTML forms and follows redirects.
- **Error handling**: Custom `*sfm.Error` type with `Kind` (ErrClient, ErrHTTP, ErrParse, ErrInvalidSID, ErrPartnerLocked), `Status`, `URL`. Use `errors.As` to unwrap.

## Testing

- Package-level tests (`package sfm`, not `sfm_test`) — in-package access to unexported internals.
- No external test frameworks; stdlib testing only.
- Tests use `t.Parallel()`, table-driven patterns, `httptest.Server`, custom `slog.Handler` capture.
- No mocking library — tests inject `getEndpointMetaFn` on `SSO` struct or override `core` internals.

## Gotchas

- **SID validation**: Username is validated against regex `^[sS]\d+$` — must look like `S123...`.
- **Partner flow** requires a separate auth step (`client.Partner().Auth(ctx)`) after login.
- **CSRF tokens**: Both `UserAdmin` and `PartnerUser` cache CSRF tokens in memory and re-login on 401 / login-required headers.
- **No linter config** in the repo. Run `go vet` and `gofmt` manually. CI only runs `go vet`.
- **Compiled binaries** (`sfm`, `sapme`) are gitignored.

## History

Renamed from `github.com/sapcli/me` (package `launchpad`) to `github.com/sapcli/sfm` (package `sfm`). CLI was called `sapme`, now `sfm`.
