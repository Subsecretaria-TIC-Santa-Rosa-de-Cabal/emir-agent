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
│   ├── storage/                # state.json + OS credential store
│   └── updater/                # self-update logic
└── scripts/                    # build and install scripts
```

## Dev Commands

```bash
# Run locally (pairing mode)
go run . --pair

# Run the agent loop
go run .

# Build current platform
go build -o emir-agent .

# Cross-compile
GOOS=windows GOARCH=amd64 go build -o emir-agent.exe .
GOOS=linux   GOARCH=amd64 go build -o emir-agent .
GOOS=darwin  GOARCH=amd64 go build -o emir-agent .

# Build all release assets
bash scripts/build-all.sh 0.2.0 dist

# Lint/vet
go vet ./...
```

## Environment Variables

| Variable | Default | Purpose |
|---|---|---|
| `EMIR_CORE_URL` | `http://localhost:8000` | Base URL of `emir-core` |
| `EMIR_STATE_PATH` | OS config dir (`%APPDATA%`/`.config`/Library) | Local `state.json` path |

## Authentication

Each agent generates an Ed25519 key pair during pairing. The public key is sent to `/api/agent/pair`. Subsequent requests are signed with the private key:

- `Authorization: Agent <token>`
- `X-Agent-Timestamp: <RFC3339>`
- `X-Agent-Signature: <base64 Ed25519 signature of timestamp>`

The token and private key seed are stored in the OS credential store (Windows Credential Manager / Linux Secret Service / macOS Keychain).

## Release Workflow

1. Bump `Version` in `internal/models/models.go`.
2. Commit and push a tag: `git tag v0.2.0 && git push origin v0.2.0`.
3. GitHub Actions builds per-platform binaries and creates a Release.
4. Register the release in `emir-core` (`agent_releases` table) including Windows, Linux and macOS assets.
5. Agents detect the new version on the next 10-minute poll and auto-update if `is_mandatory` is true.

## Conventions

- One package per conceptual responsibility under `internal/`.
- Platform-specific code uses Go build tags (`//go:build windows/linux/darwin`).
- Collectors return the request models directly; no business logic outside `internal/agent`.
- Keep dependencies minimal. Current external dependency: `github.com/zalando/go-keyring`.

## Notes

- Do not commit `dist/`, `*.exe`, or `.env`.
- The local copy in the EMIR monorepo is a mirror; the authoritative repo is the separate `emir-agent` project.
- When editing here, remember to sync `go.mod` module path with the real repo.
