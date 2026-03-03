# Botster Desktop — Mockup Spec

## Context
This is a working Tauri v2 app mockup for "Botster Desktop" — the user-facing helper app that wraps the actuator-g Go backend. Goal: demonstrate velocity to LAUNCH group. Must look polished, be visible/palpable to non-technical people.

## Architecture (already scaffolded)
- Tauri v2 + Rust backend (thin shim)
- Go sidecar (`actuator-g`) — bundled binary
- Plain JS frontend (NO TypeScript, NO React, NO build step)
- HTML/CSS/JS in `src/`

## What to Build

### 1. Menu Bar Tray Dropdown
When user clicks the tray icon, show a **compact dropdown popover** (not a full window). Contents:
- **Agent status** — green/red dot + name + "Connected" / "Disconnected"
- **Quick stats** — uptime, messages handled, last activity
- **Common actions:**
  - "Open Dashboard" (opens the main window)
  - "Grant File Access..." (opens file picker)
  - "Grant Folder Access..." (opens folder picker)  
  - "View Granted Access" (opens dashboard to access tab)
  - Separator
  - "Pause Agent" / "Resume Agent"
  - "Settings..."
  - "Quit Botster"

For the mockup: this is the tray menu — implement as a Tauri native menu from the tray icon (not a webview popup). Use `tray.set_menu()` in Rust.

### 2. Main Dashboard Window (separate from tray)
A proper resizable window (700×500 default) with a **sidebar + content** layout. Tabs/sections:

#### a) Status (default view)
- Agent connection status (large, clear indicator)
- Broker connection info (URL, connection health)
- Agent ID and capabilities
- Uptime timer
- Recent activity feed (last 10 events, timestamps)

#### b) File Access
- **Drop zone** — large drag-and-drop area: "Drop files or folders here to grant access"
- **Granted access list** — table/list showing:
  - Path (file or directory)
  - Type (File / Directory)  
  - Granted timestamp
  - "Revoke" button per item
- **Add button** — alternative to drag (opens native file/folder picker)
- Visual: clean, obvious, non-technical people should immediately understand "I drag my files here and the agent can see them"

#### c) Logs
- Scrollable log viewer (already exists, move here)
- Filter buttons: All / Info / Errors
- Clear button
- Auto-scroll toggle

#### d) Settings
- **Connection section:**
  - Broker URL
  - Broker Token (masked)
  - Actuator ID
  - Connect / Disconnect buttons
- **Advanced section** (collapsible, for admin/power users):
  - Capabilities configuration
  - Webhook port
  - Raw JSON config editor
- **Role toggle** (mockup): "Simple" vs "Advanced" mode — hides broker details in Simple mode

### 3. Visual Design
- **Dark theme** (keep existing color scheme, it's good)
- **macOS native feel** — rounded corners, subtle shadows, system font
- Accent color: the existing cyan/blue (`#4fc3f7`)
- Status indicators: green (connected), amber (connecting), red (disconnected/error)
- The app should look like it belongs on macOS — not like an Electron nightmare
- Window title bar: use Tauri's decorations (native macOS title bar)
- Sidebar: dark, ~180px wide, icons + labels

### 4. Demo Data
Since this is a mockup, populate with realistic demo data:
- Agent: "peter-agent-01" connected to "broker.botsters.dev"
- Uptime: "2h 34m"
- Some granted files: ~/Documents/project-brief.pdf, ~/Projects/my-app/ (directory)
- Activity feed with realistic entries:
  - "Agent connected to broker" 
  - "File access granted: project-brief.pdf"
  - "Agent executed: git status"
  - "Directory access granted: ~/Projects/my-app/"
  - etc.
- Logs: mix of info and connection messages

## Files to Modify/Create

### Frontend (src/)
- `src/index.html` — complete rewrite for dashboard layout
- `src/style.css` — complete rewrite for new design
- `src/app.js` — complete rewrite for new UI logic + demo data

### Rust (src-tauri/)
- `src-tauri/src/main.rs` — update tray to use native menu items, add dashboard window management
- `src-tauri/src/tray.rs` — new file, tray menu setup and event handling
- `src-tauri/tauri.conf.json` — update window config (700×500, visible by default for demo, title "Botster Desktop")

### DO NOT modify:
- `src-tauri/src/sidecar.rs` — keep existing sidecar management as-is
- `src-tauri/Cargo.toml` — only add deps if absolutely necessary
- Any Go code

## Constraints
- Plain JS only. No frameworks, no TypeScript, no npm, no build step.
- Keep it working — `cargo tauri dev` should launch and show the mockup.
- The mockup should be self-contained (demo data hardcoded in JS).
- Prioritize visual polish over functionality — this is for a demo.
- File drag-and-drop should visually work (highlight on drag, show in list) even if backend isn't wired up.
