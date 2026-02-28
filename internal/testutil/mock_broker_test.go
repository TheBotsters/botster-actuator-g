package testutil

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"nhooyr.io/websocket"

	"github.com/TheBotsters/botster-actuator-g/internal/protocol"
)

func TestMockBroker_PingPong(t *testing.T) {
	mb, err := NewMockBroker()
	if err != nil {
		t.Fatalf("NewMockBroker: %v", err)
	}
	defer mb.Close()

	// Connect a raw WebSocket client (simulating the actuator side).
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, mb.URL()+"/ws?token=test&role=actuator&actuator_id=test", nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "done")

	// Wait for the broker to register the connection.
	if err := mb.WaitForConnection(ctx); err != nil {
		t.Fatalf("WaitForConnection: %v", err)
	}

	// Broker sends ping.
	ts := 1234567890.123
	if err := mb.SendPing(ts); err != nil {
		t.Fatalf("SendPing: %v", err)
	}

	// Client reads the ping and sends a pong.
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	var ping protocol.PingMessage
	if err := json.Unmarshal(data, &ping); err != nil {
		t.Fatalf("unmarshal ping: %v", err)
	}
	if ping.Type != "ping" || ping.Ts != ts {
		t.Fatalf("unexpected ping: %+v", ping)
	}

	pong := protocol.PongMessage{Type: "pong", Ts: ping.Ts}
	pongData, _ := json.Marshal(pong)
	if err := conn.Write(ctx, websocket.MessageText, pongData); err != nil {
		t.Fatalf("Write pong: %v", err)
	}

	// Broker should receive the pong.
	got, err := mb.WaitForPong(2 * time.Second)
	if err != nil {
		t.Fatalf("WaitForPong: %v", err)
	}
	if got.Ts != ts {
		t.Fatalf("pong ts = %v, want %v", got.Ts, ts)
	}
}

func TestMockBroker_CommandResult(t *testing.T) {
	mb, err := NewMockBroker()
	if err != nil {
		t.Fatalf("NewMockBroker: %v", err)
	}
	defer mb.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, mb.URL()+"/ws?token=test&role=actuator&actuator_id=test", nil)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "done")

	if err := mb.WaitForConnection(ctx); err != nil {
		t.Fatalf("WaitForConnection: %v", err)
	}

	// Client sends a command_result.
	result := protocol.CommandResult{
		Type:   "command_result",
		ID:     "cmd-42",
		Status: "completed",
		Result: map[string]string{"stdout": "hello"},
	}
	data, _ := json.Marshal(result)
	if err := conn.Write(ctx, websocket.MessageText, data); err != nil {
		t.Fatalf("Write: %v", err)
	}

	got, err := mb.WaitForResult("cmd-42", 2*time.Second)
	if err != nil {
		t.Fatalf("WaitForResult: %v", err)
	}
	if got.Status != "completed" {
		t.Fatalf("status = %q, want completed", got.Status)
	}
}
