# AGENTS.md

## Project

**sfm** — CLI for managing SAP for Me (SAP launchpad) users. Go 1.26, cobra.

Module: `github.com/sapcli/sfm`
Binary: `sfm` (entrypoint at `cmd/sapme/main.go`)

## Directory structure

```
.               sfm/       — library package (HTTP client, SSO auth, UserAdmin, PartnerUser APIs)
cmd/sapme/      main       — CLI entrypoint (legacy dir name, keep it as-is)
cmd/sapme/internal/ — shared CLI helpers (MustClient, Print)
cmd/sapme/user/ — `user` subcommand
cmd/sapme/partneruser/ — `partneruser` subcommand
```

## Commands

```sh
go build -o sfm ./cmd/sapme          # build binary
go build ./...                        # verify compilation
go vet ./...                          # vet
go test -race ./...                   # full test suite
go test -run TestName ./...           # single test
```

CI runs: `build → vet → test -race`

## Environment

- `SAP_ADMIN_USERNAME` — S-User ID (starts with `S` + digits, e.g. `S1234567890`)
- `SAP_ADMIN_PASSWORD` — corresponding password
- `HTTP_LOG_LEVEL` — `debug|info|warn|error`

These are set via `PersistentPreRunE` in the root cobra command, checked as fallback after flags.

## Architecture

- **Root package `sfm`** is the library: `Client` (functional options), `SSO` (SAML/Gigya auth flow), `UserAdmin` (OData CRUD via launchpad.support.sap.com), `PartnerUser` (OData CRUD via partnermanagemyusers.cfapps...).
- **`cmd/sapme/`** is the CLI crust. `MustClient()` constructs an authenticated client; `Print()` formats output (json/text/table). Both read global pointer vars set by cobra's `init()`.
- **Auth path**: SID + password → SAML redirect chain → optional Gigya/CDC flow → cookie-based session. The SSO code parses HTML forms and follows redirects.
- **Error handling**: Custom `*sfm.Error` type with `Kind` (ErrClient, ErrHTTP, ErrParse, ErrInvalidSID, ErrPartnerLocked), `Status`, `URL`. Use `errors.As` to unwrap.

## Testing

- Package-level tests (`package sfm`, not `sfm_test`) — in-package access to unexported internals.
- No external test frameworks; stdlib testing only.
- Tests use `t.Parallel()`, table-driven patterns, `httptest.Server`, custom `slog.Handler` capture.
- No mocking library — tests inject `getEndpointMetaFn` on `SSO` struct or override `core` internals.

## Gotchas

- **The `cmd/sapme/` directory name is a legacy artifact** from the project rename (was `me` → `sfm`). Do NOT rename it — it would break import paths across the project.
- **SID validation**: Username is validated against regex `^[sS]\d+$` — must look like `S123...`.
- **Partner flow** requires a separate auth step (`client.Partner().Auth(ctx)`) after login.
- **CSRF tokens**: Both `UserAdmin` and `PartnerUser` cache CSRF tokens in memory and re-login on 401 / login-required headers.
- **No linter config** in the repo. Run `go vet` and `gofmt` manually. CI only runs `go vet`.
- **No README** — don't look for one.
- **Compiled binaries** (`sfm`, `sapme`) are gitignored.

## History

Renamed from `github.com/sapcli/me` (package `launchpad`) to `github.com/sapcli/sfm` (package `sfm`). CLI was called `sapme`, now `sfm`.
