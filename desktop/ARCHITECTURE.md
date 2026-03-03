# Botster Desktop — Architecture

## Overview
Botster Desktop is a Tauri desktop app that provides a user-friendly interface for managing AI agent assistants. It wraps the `actuator-g` Go binary as a sidecar and connects to a Botster Broker for secure command execution.

## Components

### 1. Tauri App (`botster-desktop`)
- **Language:** Rust + HTML/CSS/JS (no frameworks, no build step)
- **Responsibilities:**
  - System tray icon with native menu
  - Dashboard window (sidebar nav: Status, File Access, Logs, Settings)
  - File/folder drag-and-drop UI
  - Connection management (broker URL/token)
  - IPC with actuator-g sidecar (HTTP or Unix socket)
  - Native dialogs (file picker, folder picker)
  - Local settings storage (`~/.botster/config.json`)

### 2. Actuator-g Sidecar (`actuator-g`)
- **Language:** Go (single static binary)
- **Responsibilities:**
  - Local file grant database (`~/.botster/grants.db`, SQLite)
  - IPC server for Tauri communication (HTTP `localhost:18790`)
  - Capability token generation (JWT-like, scoped to paths)
  - Broker WebSocket connection
  - Command execution (shell, git, docker, filesystem)
  - Process management (start/stop/pause)

### 3. Botster Broker (Go)
- **Repository:** TheBotsters/botster-broker-go (forthcoming)
- **Responsibilities:**
  - Capability validation (file access tokens)
  - Agent registration with granted paths
  - Secret management (API keys, credentials)
  - Inference proxy (LLM calls)
  - WebSocket command bus
  - Role-based API (Simple vs Advanced users)

## Data Flow

### File Grant Flow
```
User drags file → Tauri UI → Actuator-g IPC → Store in grants.db
                                      ↓
                               Generate capability token
                                      ↓
                          Register token with broker on connect
```

### Command Execution Flow
```
Agent requests read_file(/path) → Broker validates token → Forward to actuator-g
                                      ↓
                               Actuator checks local grants
                                      ↓
                               Read file → Return to agent
```

### Connection Flow
```
Tauri UI (broker URL/token) → Start actuator-g with env vars
                                      ↓
                          Actuator connects to broker WebSocket
                                      ↓
                          Broker validates token, registers agent
```

## IPC Design

### Tauri ↔ Actuator-g
**Protocol:** HTTP REST (localhost:18790)
**Endpoints:**
- `POST /grants/file` — grant file access
- `POST /grants/folder` — grant folder access  
- `DELETE /grants/:id` — revoke access
- `GET /grants` — list all grants
- `POST /connect` — connect to broker (URL, token, agent ID)
- `POST /disconnect` — disconnect from broker
- `GET /status` — connection status, uptime, granted count
- `GET /logs` — recent logs

### Actuator-g ↔ Broker
**Protocol:** WebSocket + JSON-RPC
**Messages:**
- `register` — agent ID, capability tokens
- `command` — shell, git, docker, file operations
- `response` — command output, exit code
- `heartbeat` — keepalive

## Database Schema

### `grants` table
```sql
CREATE TABLE grants (
  id TEXT PRIMARY KEY,
  path TEXT NOT NULL,
  type TEXT NOT NULL, -- 'file' or 'dir'
  operations TEXT NOT NULL, -- 'read', 'write', 'exec'
  created_at INTEGER NOT NULL,
  expires_at INTEGER, -- NULL for no expiry
  token TEXT NOT NULL -- capability token
);
```

### `settings` table
```sql
CREATE TABLE settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
```

## Capability Tokens

**Format:** `botster_cap_<base64>`
**Payload:**
```json
{
  "agent_id": "peter-agent-01",
  "path": "/Users/peter/Documents/project.pdf",
  "path_prefix": true, // for directories
  "operations": ["read"],
  "exp": 1741011720
}
```

**Validation:** Broker checks token signature, expiry, path match, operations.

## Security Model

### Simple vs Advanced Users
- **Simple mode:** UI hides broker URL/token, shows high-level actions ("process document", "analyze folder")
- **Advanced mode:** Shows raw broker config, capability tokens, shell access

### File Access Sandboxing
- All file operations go through broker validation
- Tokens are path-scoped (no wildcard `*` access)
- Directory grants use path prefix matching
- Expiry enforced (default 30 days)

### Local Storage
- Grants DB encrypted at rest (SQLCipher optional)
- Broker tokens stored in system keychain (macOS Keychain, Windows Credential Manager, Linux secret-service)

## Deployment

### Build Targets
- **macOS** (arm64) — `.app` bundle + `.dmg`
- **Linux** (amd64) — `.AppImage` + `.deb`
- **Windows** (amd64) — `.msi`

### Auto-update
- Tauri built-in updater
- GitHub Releases as package repository
- Signature verification

## Development Phases

### Phase 1: Local Foundation
- [ ] Actuator-g IPC server
- [ ] Local grants DB
- [ ] Tauri ↔ Actuator IPC client
- [ ] Real tray menu actions
- [ ] Connect/disconnect to existing broker

### Phase 2: Broker Integration  
- [ ] Capability token system in broker
- [ ] File operation validation
- [ ] Agent registration with grants
- [ ] Role-based API endpoints

### Phase 3: Production Polish
- [ ] Auto-update
- [ ] Installer packages
- [ ] Keychain integration
- [ ] Analytics (opt-in)
- [ ] Crash reporting

## Dependencies

### Tauri
- `tauri` v2
- `tauri-plugin-shell` (sidecar management)
- `tauri-plugin-dialog` (file pickers)
- `tauri-plugin-store` (settings)

### Go
- `nhooyr.io/websocket` (broker connection)
- `modernc.org/sqlite` (grants DB)
- `github.com/gorilla/mux` (IPC HTTP server)

## Open Questions
1. **Token revocation** — How to revoke tokens already registered with broker?
2. **Offline operation** — Can agent use cached grants when broker offline?
3. **Multi-user** — Support multiple agents/identities per desktop?
4. **Audit logging** — Log all file accesses locally?

## References
- [SEKS Architecture](/Users/Shared/seksbot-shared/projects/botster-architecture/)
- [Actuator-g README](../README.md)
- [Botster Broker Go](https://github.com/TheBotsters/botster-broker-go) (forthcoming)
