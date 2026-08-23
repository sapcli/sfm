# sfm

CLI for managing SAP For Me (SAP launchpad) users and partner users.

## Installation

### Homebrew (macOS / Linux)

```bash
brew tap sapcli/tap https://github.com/sapcli/homebrew-tap.git
brew install sapcli/tap/sfm --formula
```

### Winget (Windows)

```bash
winget install SAPCLI.sfm
```

### Chocolatey (Windows)

```bash
choco install sfm
```

### Go

```bash
go install github.com/sapcli/sfm/cmd/sfm@latest
```

### Build from source

```bash
git clone https://github.com/sapcli/sfm.git
cd sfm
go build -o sfm ./cmd/sfm
```

## Quickstart

```bash
# Authenticate (S-User credentials)
export SAP_ADMIN_USERNAME=S1234567890
export SAP_ADMIN_PASSWORD=your-password

# List users
sfm user list

# Get user details
sfm user get --user-id U123456

# Search users
sfm user search --keyword "john@example.com"

# Create a user
sfm user create --email "user@example.com" --first-name "John" --last-name "Doe" \
  --customer-id C123 --department-id D456

# Extend user expiry
sfm user extend --user-ids U123456 --days 90
```

## Commands

| Command | Description |
|---|---|
| `sfm user` | Manage SAP For Me users (list, get, create, delete, extend, search, requests) |
| `sfm partneruser` | Manage SAP partner users (list, create, delete, search) |
| `sfm config` | Persist credentials to config file (set, get, unset) |

## Authentication

Credentials are resolved in this precedence order (highest first):

1. `--username` / `--password` CLI flags
2. `SAP_ADMIN_USERNAME` / `SAP_ADMIN_PASSWORD` environment variables
3. Config file saved via `sfm config set`

Session cookies are cached to disk to avoid re-authentication on repeated invocations. Cookies expire with the session.

### Config file location

| Platform | Path |
|---|---|
| Linux | `~/.config/sfm/config.json` |
| macOS | `~/Library/Application Support/sfm/config.json` |
| Windows | `%APPDATA%\sfm\config.json` |

```bash
# Save credentials
sfm config set --username S1234567890 --password your-password

# View saved config
sfm config get

# Remove saved credentials
sfm config unset
```

## Output Formats

All commands support `--output` / `-o` with:

- `json` — structured JSON (default)
- `text` — key:value or CSV lines
- `table` — aligned columns with tabwriter

```bash
sfm user list --output table
sfm user get --user-id U123456 --output text
```

## Environment Variables

| Variable | Description |
|---|---|
| `SAP_ADMIN_USERNAME` / `--username` | S-User ID (starts with `S` + digits) |
| `SAP_ADMIN_PASSWORD` / `--password` | Corresponding password |
| `HTTP_LOG_LEVEL` / `--http-log-level` | HTTP log level: `debug` \| `info` \| `warn` \| `error` |
| `--timeout` | Request timeout (default `1m30s`) |
| `--debug-body-max` | Max body bytes to log (default `2048`) |
| `-o` / `--output` | Output format: `json` \| `text` \| `table` (default `json`) |

## Library

The root `sfm` package can also be used as a Go library:

```go
import "github.com/sapcli/sfm"

client, err := sfm.NewClient("S1234567890", "password")
if err != nil {
    log.Fatal(err)
}
defer client.Logout()

if err := client.Login(ctx); err != nil {
    log.Fatal(err)
}

users, err := client.UserAdmin().Users(ctx)
```

## License

MIT
