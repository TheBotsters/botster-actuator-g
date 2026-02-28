package executor

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/TheBotsters/botster-actuator-g/internal/process"
	"github.com/TheBotsters/botster-actuator-g/internal/protocol"
)

func runShellSync(t *testing.T, payload protocol.ExecPayload, cwd string) (stdout string, exitCode int) {
	t.Helper()
	reg := process.NewRegistry()
	var mu sync.Mutex
	var out strings.Builder
	done := make(chan struct{})
	code := -1

	cb := ShellCallbacks{
		OnStdout: func(data string) {
			mu.Lock()
			out.WriteString(data)
			mu.Unlock()
		},
		OnDone: func(ec int, dur int64, s *process.Session) {
			code = ec
			close(done)
		},
		OnError: func(err string, s *process.Session) {
			code = 1
			mu.Lock()
			out.WriteString("ERROR: " + err)
			mu.Unlock()
			select {
			case <-done:
			default:
				close(done)
			}
		},
	}

	ExecuteShell(reg, payload, cwd, cb)
	<-done
	// small delay to let output flush
	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	return out.String(), code
}

func TestShell_Echo(t *testing.T) {
	out, code := runShellSync(t, protocol.ExecPayload{Command: "echo hello"}, t.TempDir())
	if code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if !strings.Contains(out, "hello") {
		t.Fatalf("output: %q", out)
	}
}

func TestShell_NonZeroExit(t *testing.T) {
	_, code := runShellSync(t, protocol.ExecPayload{Command: "exit 42"}, t.TempDir())
	if code != 42 {
		t.Fatalf("expected 42, got %d", code)
	}
}

func TestShell_Timeout(t *testing.T) {
	reg := process.NewRegistry()
	done := make(chan struct{})

	cb := ShellCallbacks{
		OnDone: func(ec int, dur int64, s *process.Session) {
			close(done)
		},
		OnError: func(err string, s *process.Session) {
			select {
			case <-done:
			default:
				close(done)
			}
		},
	}

	start := time.Now()
	ExecuteShell(reg, protocol.ExecPayload{Command: "sleep 30", Timeout: 1}, t.TempDir(), cb)
	<-done
	elapsed := time.Since(start)
	if elapsed > 5*time.Second {
		t.Fatalf("took too long: %v", elapsed)
	}
}

func TestShell_EnvVars(t *testing.T) {
	out, code := runShellSync(t, protocol.ExecPayload{
		Command: "echo $MY_TEST_VAR",
		Env:     map[string]string{"MY_TEST_VAR": "secretvalue"},
	}, t.TempDir())
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out, "secretvalue") {
		t.Fatalf("output: %q", out)
	}
}

func TestShell_WorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	out, code := runShellSync(t, protocol.ExecPayload{Command: "pwd", Cwd: dir}, "/tmp")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(out, dir) {
		t.Fatalf("expected cwd %s in output %q", dir, out)
	}
}

func TestShell_YieldMs(t *testing.T) {
	reg := process.NewRegistry()
	yielded := make(chan string, 1)
	done := make(chan struct{})

	cb := ShellCallbacks{
		OnYield: func(s *process.Session) {
			yielded <- s.ID
		},
		OnDone: func(ec int, dur int64, s *process.Session) {
			close(done)
		},
		OnError: func(err string, s *process.Session) {
			select {
			case <-done:
			default:
				close(done)
			}
		},
	}

	session := ExecuteShell(reg, protocol.ExecPayload{
		Command: "sleep 2",
		YieldMs: 100,
	}, t.TempDir(), cb)

	select {
	case id := <-yielded:
		if id != session.ID {
			t.Fatalf("wrong session id")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("yield not received")
	}

	<-done
}
