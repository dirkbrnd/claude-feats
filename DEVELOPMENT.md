# Development

## Prerequisites

- Go 1.24+
- [GoReleaser](https://goreleaser.com/install/) (for release builds)
- `gh` CLI (for repo and release management)

## Setup

```bash
git clone https://github.com/dirkbrnd/claude-feats
cd claude-feats
go mod download
```

## Build

```bash
go build .                    # build binary in current directory
go install .                  # install to $GOPATH/bin
```

## Tests

```bash
go test ./...                 # run all tests
go test ./internal/store/...  # run a specific package
go test -v ./internal/detector/...  # verbose output
go test -run TestBehavioral_VimReflex ./internal/detector/...  # single test
```

Tests are fully deterministic and self-contained — no external dependencies, no file system
side-effects (each test that touches the store uses `t.TempDir()` via `t.Setenv("HOME", ...)`).

### What's covered

| Package | Tests |
|---|---|
| `internal/store` | Load/Save, UnlockFeat (new + repeat + PR), IsProcessed/MarkProcessed, UpdateMana, LifetimeTotal, UpdateStreak (consecutive, broken, idempotent), atomic save, LoadLocked, PrevMonthKey, ManaMonth.AvgPerSession |
| `internal/catalog` | ByID, Visible, IsFibonacci, IsPalindrome, rarity ordering, unique IDs, mana feat presence, hidden feat flags |
| `internal/detector` | Behavioral, HiddenBehavioral, Milestone, HiddenMilestone, ManaDetector — all major conditions including edge cases |
| `internal/transcript` | User messages, timestamps, bash command parsing (git reset, push, worktree, rm -rf, --force), file edits + extensions, interrupted, empty/missing files |

## Project structure

```
claude-feats/
├── cmd/                   # Cobra commands (one file per command)
│   ├── root.go            # rootCmd + Execute()
│   ├── check.go           # Stop hook: reads usage, writes pending job, spawns worker
│   ├── worker.go          # Detached worker + shared analyzeSession() helper
│   ├── seed.go            # Historical backfill
│   ├── list.go            # Feat display
│   ├── stats.go           # Full stats with rarity bars
│   ├── mana.go            # Monthly token chart
│   ├── bio.go             # Anthropic API RPG bio
│   ├── export.go          # Markdown dump
│   └── hook.go            # settings.json install/uninstall
├── internal/
│   ├── catalog/           # Feat definitions (ID, name, rarity, hidden, countable)
│   ├── store/             # progress.json schema + file-locked read/write
│   ├── transcript/        # JSONL parser → Session struct
│   └── detector/          # Detector interface + all detection logic
└── skills/claude-feats/   # CC agent skill (SKILL.md)
```

## Adding a new feat

1. Add an entry to `internal/catalog/catalog.go` (`All` slice).
2. Write detection logic in the appropriate detector file:
   - `behavioral.go` — transcript content
   - `temporal.go` — wall-clock time
   - `milestone.go` — session counts, streaks, mana thresholds
   - `git.go` — git operations
3. Add tests in the corresponding `_test.go` file.
4. Run `go test ./...` to verify.

Hidden feats (`Hidden: true`) only need steps 2–4 — they must **not** appear in `Visible()` or be
documented anywhere except `SKILL.md`'s vague hint that hidden feats exist.

## Release process

### Prerequisites

1. Create a GitHub Personal Access Token with `repo` scope and store it as the `HOMEBREW_TAP_TOKEN`
   secret in the repo (`gh secret set HOMEBREW_TAP_TOKEN`).
2. Ensure `github.com/dirkbrnd/homebrew-tap` exists (already created).

### Tagging a release

```bash
git tag v0.1.0
git push origin v0.1.0
```

This triggers `.github/workflows/release.yml`, which runs GoReleaser and:
- Builds macOS arm64/amd64 and Linux amd64 binaries
- Creates a GitHub Release with tarballs and a checksum file
- Updates the Homebrew formula in `homebrew-tap`

### Local release dry-run

```bash
goreleaser release --snapshot --clean
```

Binaries are written to `dist/` without publishing anything.

### Installing from source (dev)

```bash
go install github.com/dirkbrnd/claude-feats@latest
```

### Verifying the hook

After `go install`:

```bash
claude-feats hook install
claude-feats --help
```

Check `~/.claude/settings.json` for the Stop hook entry.
