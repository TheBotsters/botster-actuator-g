# PTY Sidecar (Future)

When PTY support is needed, a small C sidecar will handle PTY allocation and proxying.

## Why a sidecar?

PTY allocation requires OS-level syscalls that vary by platform. Keeping this in a separate
binary allows the main actuator-g to remain pure Go (CGO_ENABLED=0, fully static).

## Protocol (Unix socket, length-prefixed binary frames)

```
actuator-g                          pty-sidecar
     │                                    │
     │──── spawn(cmd, rows, cols) ───────▶│  (forks, execs, attaches PTY)
     │◀─── ready(pid) ──────────────────── │
     │                                    │
     │──── data(bytes) ─────────────────▶ │  (stdin → PTY master)
     │◀─── data(bytes) ────────────────── │  (PTY master → stdout)
     │                                    │
     │──── resize(rows, cols) ──────────▶ │  (TIOCSWINSZ ioctl)
     │                                    │
     │◀─── exit(code, signal) ─────────── │
```

## Language: C (~200 lines)

Direct syscall access, tiny binary, zero dependencies beyond libc.

## Status: Not yet implemented

The main actuator logs a warning and falls back to regular exec when PTY is requested.
