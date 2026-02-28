# Actuator G Desktop

Tauri v2 desktop app wrapping the actuator-g Go binary as a sidecar.

## Architecture

```
Tauri App (.app bundle)
├── Rust backend (thin shim)
│   └── spawns + manages actuator-g sidecar
├── JS frontend (webview, plain HTML/CSS/JS)
│   └── status UI, config, logs
└── actuator-g binary (sidecar)
    └── the actual actuator
```

## Prerequisites

- Rust toolchain (`rustup`)
- Tauri CLI: `cargo install tauri-cli`
- Go (to build actuator-g binary)

## Setup

1. Build the actuator-g sidecar for your platform:

```bash
# From repo root
make build  # or make build-darwin for macOS

# Copy with target triple name
cp actuator-g desktop/src-tauri/binaries/actuator-g-aarch64-apple-darwin    # macOS arm64
cp actuator-g desktop/src-tauri/binaries/actuator-g-x86_64-unknown-linux-gnu  # Linux x64
```

2. Generate icons (or use placeholders):

```bash
cd desktop/src-tauri
cargo tauri icon path/to/icon.png
```

3. Run in dev mode:

```bash
cd desktop
cargo tauri dev
```

4. Build release:

```bash
cd desktop
cargo tauri build
```

## Frontend

Plain HTML/CSS/JS in `src/`. No TypeScript, no React, no build step.
Edit `src/index.html`, `src/style.css`, `src/app.js` directly.

## Platform Support

- **macOS** (arm64) — primary target
- **Linux** (amd64) — planned
- **Windows** (amd64) — planned

Cross-platform support is built into Tauri; just need to add
sidecar binaries for each target triple.
