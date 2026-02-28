// Package testutil provides test helpers for the actuator, including a mock broker.
package testutil

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"nhooyr.io/websocket"

	"github.com/TheBotsters/botster-actuator-g/internal/protocol"
)

// MockBroker is a WebSocket server that simulates the broker for testing.
type MockBroker struct {
	listener net.Listener
	server   *http.Server

	mu     sync.Mutex
	conn   *websocket.Conn
	connCh chan struct{} // closed when a client connects
	closed bool

	results  map[string]*protocol.CommandResult
	resultCh chan resultEntry
	pongCh   chan protocol.PongMessage
}

type resultEntry struct {
	id     string
	result *protocol.CommandResult
}

// NewMockBroker starts a mock broker listening on a random port.
func NewMockBroker() (*MockBroker, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}

	mb := &MockBroker{
		listener: ln,
		connCh:   make(chan struct{}),
		results:  make(map[string]*protocol.CommandResult),
		resultCh: make(chan resultEntry, 64),
		pongCh:   make(chan protocol.PongMessage, 16),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", mb.handleWS)
	mb.server = &http.Server{Handler: mux}

	go func() {
		if err := mb.server.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("[mock-broker] serve error: %s", err)
		}
	}()

	return mb, nil
}

// URL returns the HTTP base URL (use as BrokerURL in actuator config).
func (mb *MockBroker) URL() string {
	return "http://" + mb.listener.Addr().String()
}

// WaitForConnection blocks until a client connects or ctx is done.
func (mb *MockBroker) WaitForConnection(ctx context.Context) error {
	select {
	case <-mb.connCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// SendPing sends a ping message to the connected actuator.
func (mb *MockBroker) SendPing(ts float64) error {
	return mb.sendJSON(protocol.PingMessage{Type: "ping", Ts: ts})
}

// SendCommand sends a command_delivery to the connected actuator.
func (mb *MockBroker) SendCommand(cmd protocol.CommandDelivery) error {
	cmd.Type = "command_delivery"
	return mb.sendJSON(cmd)
}

// WaitForResult waits for a command_result with the given ID.
func (mb *MockBroker) WaitForResult(id string, timeout time.Duration) (*protocol.CommandResult, error) {
	mb.mu.Lock()
	if r, ok := mb.results[id]; ok {
		delete(mb.results, id)
		mb.mu.Unlock()
		return r, nil
	}
	mb.mu.Unlock()

	deadline := time.After(timeout)
	for {
		select {
		case entry := <-mb.resultCh:
			if entry.id == id {
				return entry.result, nil
			}
			mb.mu.Lock()
			mb.results[entry.id] = entry.result
			mb.mu.Unlock()
		case <-deadline:
			return nil, fmt.Errorf("timeout waiting for result %s", id)
		}
	}
}

// WaitForPong waits for a pong message within the timeout.
func (mb *MockBroker) WaitForPong(timeout time.Duration) (*protocol.PongMessage, error) {
	select {
	case p := <-mb.pongCh:
		return &p, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for pong")
	}
}

// Close shuts down the mock broker.
func (mb *MockBroker) Close() {
	mb.mu.Lock()
	mb.closed = true
	conn := mb.conn
	mb.mu.Unlock()

	if conn != nil {
		_ = conn.Close(websocket.StatusNormalClosure, "closing")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = mb.server.Shutdown(ctx)
}

func (mb *MockBroker) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Printf("[mock-broker] accept error: %s", err)
		return
	}
	conn.SetReadLimit(10 * 1024 * 1024)

	mb.mu.Lock()
	mb.conn = conn
	mb.mu.Unlock()

	select {
	case <-mb.connCh:
	default:
		close(mb.connCh)
	}

	mb.readLoop(conn)
}

func (mb *MockBroker) readLoop(conn *websocket.Conn) {
	for {
		_, data, err := conn.Read(context.Background())
		if err != nil {
			mb.mu.Lock()
			closed := mb.closed
			mb.mu.Unlock()
			if !closed {
				log.Printf("[mock-broker] read error: %s", err)
			}
			return
		}

		var env protocol.InboundMessage
		if err := json.Unmarshal(data, &env); err != nil {
			continue
		}

		switch env.Type {
		case "pong":
			var pong protocol.PongMessage
			if err := json.Unmarshal(data, &pong); err == nil {
				select {
				case mb.pongCh <- pong:
				default:
				}
			}
		case "command_result":
			var res protocol.CommandResult
			if err := json.Unmarshal(data, &res); err == nil {
				select {
				case mb.resultCh <- resultEntry{id: res.ID, result: &res}:
				default:
					mb.mu.Lock()
					mb.results[res.ID] = &res
					mb.mu.Unlock()
				}
			}
		}
	}
}

func (mb *MockBroker) sendJSON(v interface{}) error {
	mb.mu.Lock()
	conn := mb.conn
	mb.mu.Unlock()

	if conn == nil {
		return fmt.Errorf("no connection")
	}

	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return conn.Write(ctx, websocket.MessageText, data)
}
