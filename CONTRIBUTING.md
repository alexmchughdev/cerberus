# Contributing to Cerberus

Cerberus is a passive network observer — no traffic modification, no active probing. Keep that in mind when proposing changes.

## Before you start

- Check open issues and PRs to avoid duplicate work.
- For anything non-trivial, open an issue first. Easier to align on approach before code is written.
- The eBPF/TC pipeline is the critical path. Changes to `cerberus_tc.c` affect the struct layout and the Go parser in lockstep — read both before touching either.

## Setup

```bash
git clone https://github.com/zrougamed/cerberus.git
cd cerberus

# Dependencies (Ubuntu/Debian)
sudo apt-get install -y clang llvm libbpf-dev linux-headers-$(uname -r) make

# Build
make all

# Run
sudo ./build/cerberus
```

## Development loop

```bash
make bpf          # recompile cerberus_tc.c → cerberus_tc.o
make build        # build Go binary
go vet ./...      # must be clean
gofmt -s -w .     # must be clean
```


## Struct alignment

The `network_event` struct in `cerberus_tc.c` and `models.EventSize` in Go must stay in sync. If you change the C struct, update `internal/models/types.go` and `internal/utils/converter.go` accordingly, then verify with a short capture run.

## Adding a detection or alert rule

1. Add parsing logic in `internal/utils/converter.go` if new fields are needed.
2. Add the alert rule in `internal/monitor/monitor.go`.
3. Document the behavior in `docs/threat-and-anomaly-patterns.md` — what the rule measures, what it doesn't claim to mean.
4. Update `docs/api-reference.md` if the alert type or any API field changes.

## PR guidelines

- Conventional commit titles: `feat:`, `fix:`, `docs:`, `ci:`, `refactor:`.
- One logical change per PR.
- If it touches `cerberus_tc.c`, say which kernel versions you tested on.
- Fill in the PR template — especially the testing section.

## What's out of scope

- Active probing or traffic injection.
- ML models that require external inference services.
- Windows or macOS support (eBPF TC is Linux-only).
- Features that require breaking the existing BuntDB schema without a migration path.

## License

By contributing you agree your work is licensed under the project's [MIT License](LICENSE).