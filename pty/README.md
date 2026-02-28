# PTY Support

Pure Go via `creack/pty` — no CGO, no sidecar needed.

Works on both Linux and macOS using Go's syscall package
(`posix_openpt`/`grantpt`/`unlockpt`).

## Implementation

PTY is handled directly in `internal/executor/shell_pty.go`.
When `pty: true` is requested, we use `creack/pty` to allocate a
pseudo-terminal and attach it to the subprocess.

## Previous Design (obsolete)

Originally planned a C sidecar to avoid CGO. Turns out `creack/pty`
is pure Go and works cross-platform. Sidecar design scrapped 2026-02-28.
