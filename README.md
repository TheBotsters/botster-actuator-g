# actuator-g

Botster Actuator in Go. Static binary, no `dist/`, no runtime dependencies.

## Build

```bash
make build            # native
make build-linux      # Linux amd64 (VPS)
make build-darwin     # macOS arm64
```

## Run

```bash
export SEKS_BROKER_URL=https://broker-internal.seksbot.com
export SEKS_BROKER_TOKEN=seks_actuator_xxx

./actuator-g --id my-actuator --cwd /home/agent/workspace
```

### Brain-actuator mode (wake delivery sidecar)

```bash
./actuator-g --brain-actuator --webhook-port 18789 --id agent-ego
```

## Flags

```
--id <name>              Actuator ID (default: hostname)
--cwd <path>             Working directory (default: cwd)
--capabilities <c>       Comma-separated capabilities
--brain-actuator         Wake delivery only, no command execution
--webhook-port <port>    Webhook port for wake delivery
--webhook-token <token>  Webhook auth token
--version                Show version
```

## Why Go?

- Single static binary — `scp` and run
- No `node_modules`, no `dist/`, no build artifacts to accidentally edit
- Cross-compile from anywhere
- One external dependency (`nhooyr.io/websocket`)
