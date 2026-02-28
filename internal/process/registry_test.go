package process

import (
	"strings"
	"sync"
	"testing"
)

func TestAddSession_AppearsInList(t *testing.T) {
	r := NewRegistry()
	s := r.AddSession("echo hi", "/tmp", 123)
	if s.ID == "" {
		t.Fatal("empty ID")
	}

	list := r.ListSessions()
	if len(list) != 1 {
		t.Fatalf("expected 1, got %d", len(list))
	}
	if list[0].ID != s.ID {
		t.Fatal("ID mismatch")
	}
}

func TestAppendOutput_GetOutput(t *testing.T) {
	r := NewRegistry()
	s := r.AddSession("cmd", "/", 0)
	r.AppendOutput(s.ID, "hello ")
	r.AppendOutput(s.ID, "world")

	out := r.GetOutput(s.ID)
	if out != "hello world" {
		t.Fatalf("got %q", out)
	}
}

func TestMarkExited(t *testing.T) {
	r := NewRegistry()
	s := r.AddSession("cmd", "/", 0)
	code := 42
	r.MarkExited(s.ID, &code, "SIGTERM")

	got := r.GetSession(s.ID)
	if !got.Exited {
		t.Fatal("not exited")
	}
	if *got.ExitCode != 42 {
		t.Fatalf("exit code %d", *got.ExitCode)
	}
	if got.ExitSignal != "SIGTERM" {
		t.Fatalf("signal %q", got.ExitSignal)
	}
}

func TestOutputTruncation(t *testing.T) {
	r := NewRegistry()
	s := r.AddSession("cmd", "/", 0)

	// Write more than MaxOutputChars
	chunk := strings.Repeat("x", 50_000)
	for i := 0; i < 5; i++ {
		r.AppendOutput(s.ID, chunk)
	}
	// 250k written, should be truncated to MaxOutputChars (200k)
	out := r.GetOutput(s.ID)
	if len(out) > MaxOutputChars {
		t.Fatalf("output not truncated: %d chars", len(out))
	}
}

func TestTail(t *testing.T) {
	r := NewRegistry()
	s := r.AddSession("cmd", "/", 0)

	big := strings.Repeat("a", TailChars+1000)
	r.AppendOutput(s.ID, big)

	tail := r.Tail(s.ID)
	if len(tail) != TailChars {
		t.Fatalf("tail len %d, want %d", len(tail), TailChars)
	}
}

func TestConcurrentAccess(t *testing.T) {
	r := NewRegistry()
	s := r.AddSession("cmd", "/", 0)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.AppendOutput(s.ID, "data")
			r.GetOutput(s.ID)
			r.Tail(s.ID)
			r.ListSessions()
		}()
	}
	wg.Wait()

	// Just verify no panic/race
	out := r.GetOutput(s.ID)
	if !strings.Contains(out, "data") {
		t.Fatal("expected output")
	}
}

func TestMarkBackgrounded(t *testing.T) {
	r := NewRegistry()
	s := r.AddSession("cmd", "/", 0)
	r.MarkBackgrounded(s.ID)
	if !s.Backgrounded {
		t.Fatal("not backgrounded")
	}
}
