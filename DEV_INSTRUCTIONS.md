# Developer Quickstart & Debugging

This file explains how to run the project locally for development, enable debug logging, and do quick tests.

## Prereqs
- Go 1.20+ (1.24 recommended)
- `git`
- On Windows testing: you can cross-compile on Linux/macOS with the provided script or build on Windows.

## Running locally (Linux / macOS)
From project root:

1. Fetch modules:
   ```bash
   go mod tidy
2. Run in dev mode with logs:
```bash
DEBUG=1 go run ./cmd/rootsh
```
- This creates/uses data/0xrootshell.db and data/debug.log.

- When DEBUG=1 is set, runtime log messages are written to data/debug.log and stderr.
3. Quick tests inside the UI:
- help
- launch firefox (or launch chrome depending on your installed apps)
- find resume
- sys status

---
## Helper script (dev_run.sh)
A small helper script is included: ./dev_run.sh which sets DEBUG and runs the program.

- Cross-compile for Windows (on Linux/macOS):
  (From project root:)
```bash
./scripts/build-windows.sh
```
(Or manually:)
```bash
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o dist/0xrootshell.exe ./cmd/rootsh
```

### Where to look for debug logs

- data/debug.log : runtime debug messages (only created when DEBUG=1)

- The program also logs fatal errors to stderr which will appear in the terminal.

### Dev notes & checks

- If a command prints "Opened X" but doesn't actually open it, run the command in a terminal outside the UI to see the OS error (or enable DEBUG to see logs).

- If find blocks or takes a long time, do not press Enter repeatedly — the current search is blocking in the commands handler. (We'll fix this in Phase 2 by making search non-blocking).

- Use go vet and go test as you modify code.