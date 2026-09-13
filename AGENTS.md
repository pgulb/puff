# AGENTS.md — puff

Puff is a simple binary package manager for downloading/updating binary releases from GitHub repositories. It is a Go CLI tool (Go 1.25.3) with configuration stored in `~/.config/puff/`.

## Commands

All commands use the `go1.25.3` toolchain explicitly (not plain `go`).

```bash
task build      # builds binary: go1.25.3 build -o puff cmd/main.go
task run        # runs from source: go1.25.3 run cmd/main.go
go1.25.3 vet ./...   # vet for suspicious code
```

There are **no tests** in this repo (no `*_test.go` files). There is no linter or formatter configured.

## Architecture

- **Entry point**: `cmd/main.go` → calls `puff.Run()` in the root package.
- **Root package** (`.`) contains all library code split by concern:
  - `cli.go` — argument parsing and command dispatch (`Run` switches on `list`/`search`/`add`/`upd`/`rm`/`version`, calling `listCmd`, `searchCmd`, `addCmd`, `updCmd`, `rmCmd`)
  - `metadata.go` — `AvailableRepos()` (featured repos), metadata read/write, `Version` constant
  - `bins.go` — binary download, extraction from `.tar.gz`, install/update/remove logic
  - `gh_api.go` — GitHub API client (authenticated HTTP, release lookup, streaming download)
  - `setup.go` — config directory creation, GitHub PAT management, PATH setup prompts
- **Config directory**: `~/.config/puff/` (created on first run). Contains:
  - `bin/` — installed binaries
  - `metadata.json` — tracks installed binaries and their versions
  - `gh_pat` — GitHub Personal Access Token (file mode `0600`). Sentinel value `"-"` means no PAT.
- **No external dependencies** — `go.mod` has no third-party requires; standard library only.

## Adding a Featured Repository

Featured repos are defined in `metadata.go` in `AvailableRepos()` as `Repo` structs with `Path`, `Desc`, and `Regexp`.

To add one:
1. Verify the repo's GitHub releases contain a Linux x86_64 static binary.
2. Add a `Repo` entry with a regex that matches the correct asset name.
3. Build and test: `task build && puff add <repo>`, then run the installed binary with `--help`.
4. Check `rejected_repos.txt` first to avoid re-evaluating rejected repos. Do not add reasons/descriptions to that file — repo paths only, one per line.

## Custom Repos

If a repo is not in `AvailableRepos()`, `puff add <repo>` falls back to custom installation: it lists all release assets, prompts the user for name-part strings to match the binary, downloads it, and saves it to `bin/`.

## Update Concurrency & Safety

`puff upd` (and multi-repo `puff add`) check all installed repos in parallel, then download and install the ones needing updates, also in parallel. Custom (interactive) repos stay sequential. Failures are collected, not fatal: one repo's failure prints to stderr and continues; the command exits non-zero if any repo failed. `puff` self-updates last, sequentially, only if every repo update succeeded.

State is written safely:
- **metadata.json** is loaded once, mutated under a mutex, and written once atomically (write to `.tmp` then `rename`) at the end.
- **binaries** are always written to `<name>.tmp` then renamed over the final path, so a failed/interrupted download never leaves a half-written executable.
- Metadata is only mutated *after* a download succeeds, so a failed download cannot leave metadata claiming a binary that is not on disk.

## Versioning

`Version` is a string constant in `metadata.go` (currently `"v0.10.0"`). The `upd` command checks `pgulb/puff` for a newer release and self-updates.

## CI

`.github/workflows/go.yml` builds and uploads the binary on tag pushes only. Build command in CI: `CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ./puff cmd/main.go`.

## Security / Conventions

- PAT is stored in `~/.config/puff/gh_pat` with `0600` permissions. The PAT needs no GitHub scopes (used only to avoid API rate limits).
- The `.env` file at repo root contains a real PAT but is gitignored (`*.env` in `.gitignore`); never commit secrets.
- Error handling: check errors immediately, wrap with `fmt.Errorf("...: %w", err)`, use `log.Fatal` for unrecoverable setup errors.
- Exit code 1 for errors, 0 for success.