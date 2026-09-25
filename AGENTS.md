# EMIR Agent — Agent Instructions

Agent-focused guide for working on the `emir-agent` Go desktop service.

## Project

- **Language:** Go 1.22+
- **Module:** `github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent`
- **Repo:** https://github.com/Subsecretaria-TIC-Santa-Rosa-de-Cabal/emir-agent
- **Communication:** Outbound polling only. The agent calls `emir-core`; the backend never calls the agent.

## Layout

```
emir-agent/
├── main.go                     # entrypoint
├── go.mod
├── internal/
│   ├── agent/                  # lifecycle: pair, poll, heartbeat, inventory, update
│   ├── api/                    # authenticated HTTP client
│   ├── auth/                   # Ed25519 key generation and request signing
│   ├── collectors/             # per-OS hardware/software inventory collectors
│   ├── config/                 # env/config loading
│   ├── models/                 # JSON schemas and shared constants
│   ├── pairing/                # interactive CLI pairing flow
│   ├── storage/                # state.json + credentials.json
│   ├── tray/                   # system tray icon (Windows interactive)
│   └── updater/                # self-update logic
└── scripts/                    # build and install scripts
```

## Dev Commands

```bash
# Run locally (pairing mode)
go run . --pair

# Run the agent loop (shows system tray icon on Windows)
go run .

# Print compiled version
go run . --version

# Build current platform
go build -o emir-agent .

# Cross-compile
GOOS=windows GOARCH=amd64 go build -o emir-agent.exe .
GOOS=linux   GOARCH=amd64 go build -o emir-agent .
GOOS=darwin  GOARCH=amd64 go build -o emir-agent .

# Build all release assets
bash scripts/build-all.sh 0.2.0 dist

# Lint/vet
GOTOOLCHAIN=local go vet ./...
```

## Environment Variables

| Variable | Default | Purpose |
|---|---|---|
| `EMIR_CORE_URL` | `http://localhost:8000` | Base URL of `emir-core` |
| `EMIR_STATE_PATH` | OS config dir (`%APPDATA%`/`.config`/Library) | Local `state.json` path |

The install scripts override `EMIR_STATE_PATH` to a machine-wide directory so the service account can read the state and credentials created during interactive pairing:

- Windows: `C:\ProgramData\emir-agent\state.json`
- Linux: `/var/lib/emir-agent/state.json`
- macOS: `/Library/Application Support/emir-agent/state.json`

## Authentication

Each agent generates an Ed25519 key pair during pairing. The public key is sent to `/api/agent/pair`. Subsequent requests are signed with the private key:

- `Authorization: Agent <token>`
- `X-Agent-Timestamp: <RFC3339>`
- `X-Agent-Signature: <base64 Ed25519 signature of timestamp>`

The token and private key seed are stored in `<state-dir>/credentials.json` alongside `state.json`. The install scripts use a machine-wide state directory so the service account (e.g., Windows SYSTEM) can read credentials created during interactive pairing.

## Release Workflow

1. Bump `Version` in `internal/models/models.go`.
2. Commit and push a tag: `git tag v0.2.0 && git push origin v0.2.0`.
3. GitHub Actions builds per-platform binaries, verifies each binary reports the correct version via `--version`, creates a Release, and attaches install scripts.
4. Register the release in `emir-core` (`agent_releases` table) including Windows, Linux and macOS assets.
5. Agents detect the new version on the next 10-minute poll and auto-update if `is_mandatory` is true.

## Windows service support

The agent integrates natively with the Windows Service Control Manager (`internal/service/service_windows.go`). When running non-interactively it registers with SCM and responds to `Start`, `Stop`, and `Shutdown`. The install script registers the service with `sc.exe`.

## System tray icon

When the agent runs interactively on Windows it displays an icon in the system tray (notification area) using the EMIR favicon (`internal/tray/favicon.ico`). The tray menu provides an option to close/exit the agent. Services cannot display tray icons because they run in session 0, so the icon is only available in interactive mode.

## Testing Auto-Update Locally

Use the helper scripts to simulate a new release without publishing to GitHub:

```powershell
# Windows
.\scripts\test-update.ps1 -CoreURL http://localhost:8000 -OldVersion 0.3.0 -NewVersion 0.3.5
```

```bash
# Linux / macOS
bash scripts/test-update.sh http://localhost:8000 0.3.0 0.3.5
```

The script builds two binaries, starts a local HTTP server, and prints the SQL to register the fake release in `emir-core`. Run the old binary, watch it detect the update, replace itself, and restart.

## Version persistence

`internal/models/models.go` declares `Version` as a `var` (not a `const`) so release builds can inject the real version via `-ldflags`. The agent also persists the last applied target version in `state.json`. If a release build is compiled without the correct `-ldflags` and reports the default `0.1.0`, the persisted version prevents an infinite auto-update loop.

## Conventions

- One package per conceptual responsibility under `internal/`.
- Platform-specific code uses Go build tags (`//go:build windows/linux/darwin`).
- Collectors return the request models directly; no business logic outside `internal/agent`.
- Keep dependencies minimal. Current external dependencies:
  - `golang.org/x/sys` — Windows service integration.
  - `fyne.io/systray` — System tray icon.
- Use `GOTOOLCHAIN=local` when the installed Go toolchain is newer than the `go 1.22` directive in `go.mod`.

## Notes

- Do not commit `dist/`, `*.exe`, or `.env`.
- The local copy in the EMIR monorepo is a mirror; the authoritative repo is the separate `emir-agent` project.
- When editing here, remember to sync `go.mod` module path with the real repo.
