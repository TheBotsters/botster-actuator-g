package executor

import (
	"fmt"
	"strings"

	"github.com/TheBotsters/botster-actuator-g/internal/process"
	"github.com/TheBotsters/botster-actuator-g/internal/protocol"
)

// ProcessHandler handles process management actions (list, poll, log, write, send-keys, kill).
type ProcessHandler struct {
	registry *process.Registry
}

// NewProcessHandler creates a ProcessHandler backed by the given registry.
func NewProcessHandler(registry *process.Registry) *ProcessHandler {
	return &ProcessHandler{registry: registry}
}

// Handle dispatches a process action and returns the result.
func (h *ProcessHandler) Handle(payload protocol.ProcessPayload) protocol.ProcessResult {
	switch payload.Action {
	case "list":
		return h.listSessions()
	case "poll":
		return h.pollSession(payload.SessionID)
	case "log":
		return h.getSessionLog(payload.SessionID, payload.Offset, payload.Limit)
	case "write":
		return h.writeToSession(payload.SessionID, payload.Data)
	case "send-keys":
		return h.sendKeysToSession(payload.SessionID, payload.Keys)
	case "resize":
		return h.resizeSession(payload.SessionID, payload.Rows, payload.Cols)
	case "kill":
		return h.killSession(payload.SessionID)
	default:
		return protocol.ProcessResult{Error: fmt.Sprintf("Unknown process action: %s", payload.Action)}
	}
}

func (h *ProcessHandler) listSessions() protocol.ProcessResult {
	sessions := h.registry.ListSessions()
	infos := make([]protocol.ProcessInfo, len(sessions))
	for i, s := range sessions {
		infos[i] = h.registry.ToProcessInfo(s)
	}
	return protocol.ProcessResult{Sessions: infos}
}

func (h *ProcessHandler) pollSession(sessionID string) protocol.ProcessResult {
	if sessionID == "" {
		return protocol.ProcessResult{Error: "Session ID required for poll action"}
	}
	s := h.registry.GetSession(sessionID)
	if s == nil {
		return protocol.ProcessResult{Error: fmt.Sprintf("Session not found: %s", sessionID)}
	}
	info := h.registry.ToProcessInfo(s)
	tail := h.registry.Tail(sessionID)
	return protocol.ProcessResult{Session: &info, Tail: tail}
}

func (h *ProcessHandler) getSessionLog(sessionID string, offset, limit *int) protocol.ProcessResult {
	if sessionID == "" {
		return protocol.ProcessResult{Error: "Session ID required for log action"}
	}
	s := h.registry.GetSession(sessionID)
	if s == nil {
		return protocol.ProcessResult{Error: fmt.Sprintf("Session not found: %s", sessionID)}
	}

	output := h.registry.GetOutput(sessionID)

	if offset != nil || limit != nil {
		lines := strings.Split(output, "\n")
		startLine := 0
		if offset != nil {
			startLine = *offset - 1
			if startLine < 0 {
				startLine = 0
			}
		}
		endLine := len(lines)
		if limit != nil {
			endLine = startLine + *limit
			if endLine > len(lines) {
				endLine = len(lines)
			}
		}
		if startLine >= len(lines) {
			output = ""
		} else {
			output = strings.Join(lines[startLine:endLine], "\n")
		}
	}

	info := h.registry.ToProcessInfo(s)
	return protocol.ProcessResult{Session: &info, Output: output}
}

func (h *ProcessHandler) writeToSession(sessionID, data string) protocol.ProcessResult {
	if sessionID == "" {
		return protocol.ProcessResult{Error: "Session ID required for write action"}
	}
	if data == "" {
		return protocol.ProcessResult{Error: "Data required for write action"}
	}
	s := h.registry.GetSession(sessionID)
	if s == nil {
		return protocol.ProcessResult{Error: fmt.Sprintf("Session not found: %s", sessionID)}
	}
	if s.Exited {
		return protocol.ProcessResult{Error: "Cannot write to exited process"}
	}

	if err := WriteToSession(s, data); err != nil {
		return protocol.ProcessResult{Error: fmt.Sprintf("Failed to write to session: %s", err.Error())}
	}

	info := h.registry.ToProcessInfo(s)
	return protocol.ProcessResult{Session: &info}
}

func (h *ProcessHandler) sendKeysToSession(sessionID string, keys []string) protocol.ProcessResult {
	if sessionID == "" {
		return protocol.ProcessResult{Error: "Session ID required for send-keys action"}
	}
	if len(keys) == 0 {
		return protocol.ProcessResult{Error: "Keys required for send-keys action"}
	}
	s := h.registry.GetSession(sessionID)
	if s == nil {
		return protocol.ProcessResult{Error: fmt.Sprintf("Session not found: %s", sessionID)}
	}
	if s.Exited {
		return protocol.ProcessResult{Error: "Cannot send keys to exited process"}
	}

	for _, key := range keys {
		seq := convertKeySequence(key)
		if err := WriteToSession(s, seq); err != nil {
			return protocol.ProcessResult{Error: fmt.Sprintf("Failed to send keys: %s", err.Error())}
		}
	}

	info := h.registry.ToProcessInfo(s)
	return protocol.ProcessResult{Session: &info}
}

func (h *ProcessHandler) killSession(sessionID string) protocol.ProcessResult {
	if sessionID == "" {
		return protocol.ProcessResult{Error: "Session ID required for kill action"}
	}
	s := h.registry.GetSession(sessionID)
	if s == nil {
		return protocol.ProcessResult{Error: fmt.Sprintf("Session not found: %s", sessionID)}
	}
	if s.Exited {
		return protocol.ProcessResult{Error: "Process already exited"}
	}

	if !h.registry.KillSession(sessionID) {
		return protocol.ProcessResult{Error: "Failed to kill process (may have already exited)"}
	}

	info := h.registry.ToProcessInfo(s)
	return protocol.ProcessResult{Session: &info}
}

func (h *ProcessHandler) resizeSession(sessionID string, rows, cols *uint16) protocol.ProcessResult {
	if sessionID == "" {
		return protocol.ProcessResult{Error: "Session ID required for resize action"}
	}
	if rows == nil || cols == nil {
		return protocol.ProcessResult{Error: "Both rows and cols are required for resize action"}
	}
	s := h.registry.GetSession(sessionID)
	if s == nil {
		return protocol.ProcessResult{Error: fmt.Sprintf("Session not found: %s", sessionID)}
	}
	if s.Exited {
		return protocol.ProcessResult{Error: "Cannot resize exited process"}
	}
	if err := ResizePTY(s, *rows, *cols); err != nil {
		return protocol.ProcessResult{Error: fmt.Sprintf("Failed to resize PTY: %s", err.Error())}
	}
	info := h.registry.ToProcessInfo(s)
	return protocol.ProcessResult{Session: &info}
}

var keyMap = map[string]string{
	"Enter":     "\r",
	"Return":    "\r",
	"Tab":       "\t",
	"Space":     " ",
	"Escape":    "\x1b",
	"Backspace": "\x08",
	"Delete":    "\x7f",
	"ArrowUp":   "\x1b[A",
	"ArrowDown": "\x1b[B",
	"ArrowRight":"\x1b[C",
	"ArrowLeft": "\x1b[D",
	"Home":      "\x1b[H",
	"End":       "\x1b[F",
	"PageUp":    "\x1b[5~",
	"PageDown":  "\x1b[6~",
	"F1":        "\x1bOP",
	"F2":        "\x1bOQ",
	"F3":        "\x1bOR",
	"F4":        "\x1bOS",
	"F5":        "\x1b[15~",
	"F6":        "\x1b[17~",
	"F7":        "\x1b[18~",
	"F8":        "\x1b[19~",
	"F9":        "\x1b[20~",
	"F10":       "\x1b[21~",
	"F11":       "\x1b[23~",
	"F12":       "\x1b[24~",
}

func convertKeySequence(key string) string {
	// Handle Ctrl+ combinations
	if strings.HasPrefix(key, "Ctrl+") {
		char := strings.ToLower(key[5:])
		if len(char) == 1 {
			code := char[0] - 96
			if code >= 1 && code <= 26 {
				return string(rune(code))
			}
		}
	}

	if seq, ok := keyMap[key]; ok {
		return seq
	}
	return key
}
