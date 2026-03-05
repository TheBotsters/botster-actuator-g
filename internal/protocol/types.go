// Package protocol defines the wire protocol types for actuator ↔ broker communication.
// Must match seks-broker-2 (BotstersDev/botsters-broker).
package protocol

import "encoding/json"

// ─── Broker → Actuator ─────────────────────────────────────────────────────────

// InboundMessage is the raw envelope from the broker. Type is inspected first,
// then the message is re-parsed into the appropriate struct.
type InboundMessage struct {
	Type string `json:"type"`
}

// CommandDelivery is a command dispatched from the broker to this actuator.
type CommandDelivery struct {
	Type       string          `json:"type"` // "command_delivery"
	ID         string          `json:"id"`
	Capability string          `json:"capability"`
	Payload    json.RawMessage `json:"payload"`
}

// ExecPayload is the payload for shell execution commands.
type ExecPayload struct {
	Command    string            `json:"command"`
	Cwd        string            `json:"cwd,omitempty"`
	Timeout    int               `json:"timeout,omitempty"`
	Env        map[string]string `json:"env,omitempty"`
	Pty        bool              `json:"pty,omitempty"`
	Background bool              `json:"background,omitempty"`
	YieldMs    int               `json:"yieldMs,omitempty"`
}

// ProcessPayload is the payload for process management commands.
type ProcessPayload struct {
	Action    string   `json:"action"`
	SessionID string   `json:"sessionId,omitempty"`
	Data      string   `json:"data,omitempty"`
	Keys      []string `json:"keys,omitempty"`
	Rows      *uint16  `json:"rows,omitempty"`
	Cols      *uint16  `json:"cols,omitempty"`
	Offset    *int     `json:"offset,omitempty"`
	Limit     *int     `json:"limit,omitempty"`
}

// ReadPayload is the payload for file read commands.
type ReadPayload struct {
	Path   string `json:"path"`
	Offset *int   `json:"offset,omitempty"`
	Limit  *int   `json:"limit,omitempty"`
}

// WritePayload is the payload for file write commands.
type WritePayload struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// EditPayload is the payload for file edit commands.
type EditPayload struct {
	Path    string `json:"path"`
	OldText string `json:"oldText"`
	NewText string `json:"newText"`
}

// PingMessage is a keepalive from the broker.
type PingMessage struct {
	Type string  `json:"type"` // "ping"
	Ts   float64 `json:"ts"`
}

// WakeDelivery is a wake event for the co-located brain.
type WakeDelivery struct {
	Type   string `json:"type"` // "wake"
	Text   string `json:"text"`
	Source string `json:"source"`
	Ts     string `json:"ts"`
}

// TokenRotated is sent by the broker when the agent token has been rotated.
type TokenRotated struct {
	Type  string `json:"type"`  // "token_rotated"
	Token string `json:"token"` // new agent token
}

// ErrorMessage is an error from the broker.
type ErrorMessage struct {
	Type    string `json:"type"` // "error"
	Code    string `json:"code"`
	Message string `json:"message"`
	RefID   string `json:"ref_id,omitempty"`
}

// ─── Actuator → Broker ─────────────────────────────────────────────────────────

// CommandResult reports the outcome of a command back to the broker.
type CommandResult struct {
	Type   string      `json:"type"`   // "command_result"
	ID     string      `json:"id"`
	Status string      `json:"status"` // "completed" | "failed" | "running"
	Result interface{} `json:"result"`
}

// PongMessage is the keepalive response.
type PongMessage struct {
	Type string  `json:"type"` // "pong"
	Ts   float64 `json:"ts"`
}

// CommandStream sends incremental output (not currently used, but in protocol).
type CommandStream struct {
	Type   string `json:"type"`   // "command_stream"
	ID     string `json:"id"`
	Stream string `json:"stream"` // "stdout" | "stderr"
	Data   string `json:"data"`
}

// ─── Result Payloads ────────────────────────────────────────────────────────────

// ExecResult is the result payload for shell execution commands.
type ExecResult struct {
	Stdout     string `json:"stdout,omitempty"`
	Stderr     string `json:"stderr,omitempty"`
	ExitCode   int    `json:"exitCode,omitempty"`
	DurationMs int64  `json:"durationMs,omitempty"`
	SessionID  string `json:"sessionId,omitempty"`
	Pid        int    `json:"pid,omitempty"`
	Error      string `json:"error,omitempty"`
}

// FileResult is the result payload for file operations.
type FileResult struct {
	Content   string `json:"content,omitempty"`
	LinesRead int    `json:"linesRead,omitempty"`
	Truncated bool   `json:"truncated,omitempty"`
	Error     string `json:"error,omitempty"`
}

// ProcessResult is the result payload for process management commands.
type ProcessResult struct {
	Sessions []ProcessInfo `json:"sessions,omitempty"`
	Session  *ProcessInfo  `json:"session,omitempty"`
	Tail     string        `json:"tail,omitempty"`
	Output   string        `json:"output,omitempty"`
	Error    string        `json:"error,omitempty"`
}

// ProcessInfo describes a tracked process session.
type ProcessInfo struct {
	ID           string  `json:"id"`
	Command      string  `json:"command"`
	Pid          int     `json:"pid,omitempty"`
	StartedAt    int64   `json:"startedAt"`
	Cwd          string  `json:"cwd"`
	Exited       bool    `json:"exited"`
	ExitCode     *int    `json:"exitCode,omitempty"`
	ExitSignal   string  `json:"exitSignal,omitempty"`
	Backgrounded bool    `json:"backgrounded"`
}
