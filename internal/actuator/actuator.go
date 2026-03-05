// Package actuator provides the core actuator that connects to the broker via WebSocket.
package actuator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"nhooyr.io/websocket"

	"github.com/TheBotsters/botster-actuator-g/internal/executor"
	"github.com/TheBotsters/botster-actuator-g/internal/process"
	"github.com/TheBotsters/botster-actuator-g/internal/protocol"
)

// Config holds the actuator configuration.
type Config struct {
	BrokerURL      string
	AgentToken     string
	ActuatorID     string
	Capabilities   []string
	Cwd            string
	BrainActuator  bool
	WebhookPort    int
	WebhookToken   string
	TokenFile      string
}

// Actuator connects to the broker and executes commands.
type Actuator struct {
	config         Config
	conn           *websocket.Conn
	reconnect      *ReconnectManager
	registry       *process.Registry
	fileExecutor   *executor.FileExecutor
	processHandler *executor.ProcessHandler
	activeCommands map[string]func()
	destroyed      bool
	mu             sync.Mutex
	sendMu         sync.Mutex
}

// New creates a new Actuator with the given configuration.
func New(config Config) *Actuator {
	registry := process.NewRegistry()
	return &Actuator{
		config:         config,
		reconnect:      NewReconnectManager(ReconnectOptions{}),
		registry:       registry,
		fileExecutor:   executor.NewFileExecutor(config.Cwd),
		processHandler: executor.NewProcessHandler(registry),
		activeCommands: make(map[string]func()),
	}
}

// Start begins the WebSocket connection to the broker.
func (a *Actuator) Start() {
	a.connect()
}

// Stop gracefully shuts down the actuator.
func (a *Actuator) Stop() {
	a.mu.Lock()
	a.destroyed = true
	a.reconnect.Destroy()
	for _, kill := range a.activeCommands {
		kill()
	}
	a.activeCommands = make(map[string]func())
	conn := a.conn
	a.conn = nil
	a.mu.Unlock()

	if conn != nil {
		_ = conn.Close(websocket.StatusNormalClosure, "actuator shutting down")
	}
}

func (a *Actuator) connect() {
	a.mu.Lock()
	if a.destroyed {
		a.mu.Unlock()
		return
	}
	a.mu.Unlock()

	base := strings.Replace(a.config.BrokerURL, "https://", "wss://", 1)
	base = strings.Replace(base, "http://", "ws://", 1)

	wsURL := fmt.Sprintf("%s/ws?token=%s&role=actuator&actuator_id=%s",
		base,
		url.QueryEscape(a.config.AgentToken),
		url.QueryEscape(a.config.ActuatorID),
	)

	log.Printf("[botster-actuator] Connecting to %s/ws as %s", a.config.BrokerURL, a.config.ActuatorID)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		log.Printf("[botster-actuator] Failed to connect: %s", err)
		a.scheduleReconnect()
		return
	}

	// Set read limit high enough for large payloads
	conn.SetReadLimit(10 * 1024 * 1024) // 10MB

	a.mu.Lock()
	a.conn = conn
	a.mu.Unlock()

	log.Printf("[botster-actuator] Connected and authenticated as %s", a.config.ActuatorID)
	a.reconnect.Reset()

	// Read loop
	a.readLoop(conn)
}

func (a *Actuator) readLoop(conn *websocket.Conn) {
	for {
		_, data, err := conn.Read(context.Background())
		if err != nil {
			a.mu.Lock()
			destroyed := a.destroyed
			a.conn = nil
			a.mu.Unlock()

			if !destroyed {
				log.Printf("[botster-actuator] Disconnected: %s", err)
				a.scheduleReconnect()
			}
			return
		}

		var env protocol.InboundMessage
		if err := json.Unmarshal(data, &env); err != nil {
			log.Printf("[botster-actuator] Invalid message: %s", err)
			continue
		}

		switch env.Type {
		case "command_delivery":
			var msg protocol.CommandDelivery
			if err := json.Unmarshal(data, &msg); err != nil {
				log.Printf("[botster-actuator] Invalid command_delivery: %s", err)
				continue
			}
			go a.handleCommand(msg)

		case "ping":
			var msg protocol.PingMessage
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}
			a.send(protocol.PongMessage{Type: "pong", Ts: msg.Ts})

		case "wake":
			var msg protocol.WakeDelivery
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}
			go a.handleWake(msg)

		case "token_rotated":
			var msg protocol.TokenRotated
			if err := json.Unmarshal(data, &msg); err != nil {
				log.Printf("[actuator] Invalid token_rotated message: %s", err)
				continue
			}
			a.handleTokenRotated(msg)

		case "error":
			var msg protocol.ErrorMessage
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}
			log.Printf("[botster-actuator] Broker error [%s]: %s", msg.Code, msg.Message)

		default:
			log.Printf("[botster-actuator] Unknown message type: %s", env.Type)
		}
	}
}

func (a *Actuator) handleCommand(msg protocol.CommandDelivery) {
	if a.config.BrainActuator {
		a.sendResult(msg.ID, "failed", map[string]string{"error": "Brain-actuator mode does not execute commands"})
		return
	}

	log.Printf("[botster-actuator] Command %s: %s", msg.ID, msg.Capability)

	switch msg.Capability {
	case "exec":
		var payload protocol.ExecPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			a.sendResult(msg.ID, "failed", map[string]string{"error": "Invalid exec payload: " + err.Error()})
			return
		}
		a.handleExec(msg.ID, payload)

	case "process":
		var payload protocol.ProcessPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			a.sendResult(msg.ID, "failed", map[string]string{"error": "Invalid process payload: " + err.Error()})
			return
		}
		result := a.processHandler.Handle(payload)
		if result.Error != "" {
			a.sendResult(msg.ID, "failed", result)
		} else {
			a.sendResult(msg.ID, "completed", result)
		}

	case "read":
		var payload protocol.ReadPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			a.sendResult(msg.ID, "failed", map[string]string{"error": "Invalid read payload: " + err.Error()})
			return
		}
		result := a.fileExecutor.Read(payload)
		if result.Error != "" {
			a.sendResult(msg.ID, "failed", result)
		} else {
			a.sendResult(msg.ID, "completed", result)
		}

	case "write":
		var payload protocol.WritePayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			a.sendResult(msg.ID, "failed", map[string]string{"error": "Invalid write payload: " + err.Error()})
			return
		}
		result := a.fileExecutor.Write(payload)
		if result.Error != "" {
			a.sendResult(msg.ID, "failed", result)
		} else {
			a.sendResult(msg.ID, "completed", result)
		}

	case "edit":
		var payload protocol.EditPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			a.sendResult(msg.ID, "failed", map[string]string{"error": "Invalid edit payload: " + err.Error()})
			return
		}
		result := a.fileExecutor.Edit(payload)
		if result.Error != "" {
			a.sendResult(msg.ID, "failed", result)
		} else {
			a.sendResult(msg.ID, "completed", result)
		}

	case "shell", "actuator/shell":
		// Legacy support
		var payload protocol.ExecPayload
		if err := json.Unmarshal(msg.Payload, &payload); err != nil {
			a.sendResult(msg.ID, "failed", map[string]string{"error": "Invalid shell payload: " + err.Error()})
			return
		}
		if payload.Command == "" {
			a.sendResult(msg.ID, "failed", map[string]string{"error": "No command specified"})
			return
		}
		a.handleExec(msg.ID, payload)

	default:
		a.sendResult(msg.ID, "failed", map[string]string{"error": fmt.Sprintf("Unsupported capability: %s", msg.Capability)})
	}
}

func (a *Actuator) handleExec(id string, payload protocol.ExecPayload) {
	if payload.Command == "" {
		a.sendResult(id, "failed", map[string]string{"error": "No command specified"})
		return
	}

	var stdout, stderr strings.Builder

	session := executor.ExecuteShell(a.registry, payload, a.config.Cwd, executor.ShellCallbacks{
		OnStdout: func(data string) { stdout.WriteString(data) },
		OnStderr: func(data string) { stderr.WriteString(data) },
		OnDone: func(exitCode int, durationMs int64, session *process.Session) {
			a.mu.Lock()
			delete(a.activeCommands, id)
			a.mu.Unlock()

			status := "completed"
			if exitCode != 0 {
				status = "failed"
			}
			a.sendResult(id, status, protocol.ExecResult{
				Stdout:     stdout.String(),
				Stderr:     stderr.String(),
				ExitCode:   exitCode,
				DurationMs: durationMs,
				SessionID:  session.ID,
				Pid:        session.Pid,
			})
		},
		OnError: func(errMsg string, session *process.Session) {
			a.mu.Lock()
			delete(a.activeCommands, id)
			a.mu.Unlock()

			a.sendResult(id, "failed", protocol.ExecResult{
				Error:     errMsg,
				Stdout:    stdout.String(),
				Stderr:    stderr.String(),
				SessionID: session.ID,
				Pid:       session.Pid,
			})
		},
		OnYield: func(session *process.Session) {
			a.sendResult(id, "running", protocol.ExecResult{
				SessionID: session.ID,
				Pid:       session.Pid,
				Stdout:    stdout.String(),
				Stderr:    stderr.String(),
			})
		},
	})

	a.mu.Lock()
	a.activeCommands[id] = func() {
		if session.Pid != 0 && !session.Exited {
			a.registry.KillSession(session.ID)
		}
	}
	a.mu.Unlock()
}

func (a *Actuator) handleWake(msg protocol.WakeDelivery) {
	if a.config.WebhookPort == 0 {
		log.Println("[botster-actuator] Received wake but no webhook port configured — dropping")
		return
	}

	wakeURL := fmt.Sprintf("http://localhost:%d/hooks/wake", a.config.WebhookPort)

	body, _ := json.Marshal(map[string]string{
		"text":   msg.Text,
		"source": msg.Source,
		"ts":     msg.Ts,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", wakeURL, bytes.NewReader(body))
	if err != nil {
		log.Printf("[botster-actuator] Wake delivery failed: %s", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if a.config.WebhookToken != "" {
		req.Header.Set("Authorization", "Bearer "+a.config.WebhookToken)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[botster-actuator] Wake delivery failed to %s: %s", wakeURL, err)
		return
	}
	defer resp.Body.Close()
	log.Printf("[botster-actuator] Wake delivered to %s: %d", wakeURL, resp.StatusCode)
}

func (a *Actuator) sendResult(id, status string, result interface{}) {
	a.send(protocol.CommandResult{
		Type:   "command_result",
		ID:     id,
		Status: status,
		Result: result,
	})
}

func (a *Actuator) send(msg interface{}) {
	a.sendMu.Lock()
	defer a.sendMu.Unlock()

	a.mu.Lock()
	conn := a.conn
	a.mu.Unlock()

	if conn == nil {
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[botster-actuator] Failed to marshal message: %s", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		log.Printf("[botster-actuator] Failed to send message: %s", err)
	}
}

func (a *Actuator) handleTokenRotated(msg protocol.TokenRotated) {
	if msg.Token == "" {
		log.Println("[botster-actuator] CRITICAL: token_rotated message had empty token, ignoring")
		return
	}

	a.mu.Lock()
	a.config.AgentToken = msg.Token
	tokenFile := a.config.TokenFile
	a.mu.Unlock()

	log.Println("[botster-actuator] Token rotated by broker, persisting to disk")

	if tokenFile == "" {
		log.Println("[botster-actuator] WARNING: no --token-file configured, rotated token will not survive restart")
		return
	}

	if err := a.persistToken(tokenFile, msg.Token); err != nil {
		log.Printf("[botster-actuator] CRITICAL: failed to persist rotated token to %s: %s", tokenFile, err)
	} else {
		log.Printf("[botster-actuator] Token persisted to %s", tokenFile)
	}
}

func (a *Actuator) persistToken(path, token string) error {
	return os.WriteFile(path, []byte(token+"\n"), 0600)
}

func (a *Actuator) scheduleReconnect() {
	a.mu.Lock()
	destroyed := a.destroyed
	a.mu.Unlock()

	if destroyed {
		return
	}

	if !a.reconnect.Schedule(func() { a.connect() }) {
		log.Println("[botster-actuator] Max reconnection attempts reached")
	}
}
