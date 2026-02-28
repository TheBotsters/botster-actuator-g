package executor

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"

	pty "github.com/creack/pty/v2"

	"github.com/TheBotsters/botster-actuator-g/internal/process"
	"github.com/TheBotsters/botster-actuator-g/internal/protocol"
)

// ResizePTY resizes the PTY attached to a session.
func ResizePTY(session *process.Session, rows, cols uint16) error {
	if session.Ptmx == nil {
		return fmt.Errorf("session %s has no PTY", session.ID)
	}
	return pty.Setsize(session.Ptmx, &pty.Winsize{Rows: rows, Cols: cols})
}

// ExecuteShellPTY runs a shell command with a PTY attached.
func ExecuteShellPTY(registry *process.Registry, payload protocol.ExecPayload, cwd string, callbacks ShellCallbacks) *process.Session {
	timeoutSec := payload.Timeout
	if timeoutSec <= 0 {
		timeoutSec = defaultTimeout
	}
	if timeoutSec > maxTimeout {
		timeoutSec = maxTimeout
	}

	cmdCwd := cwd
	if payload.Cwd != "" {
		cmdCwd = payload.Cwd
	}

	session := registry.AddSession(payload.Command, cmdCwd, 0)
	startTime := time.Now()

	cmd := exec.Command("sh", "-c", payload.Command)
	cmd.Dir = cmdCwd
	cmd.Env = os.Environ()
	for k, v := range payload.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	ptmx, err := pty.Start(cmd)
	if err != nil {
		code := 1
		registry.MarkExited(session.ID, &code, "")
		callbacks.OnError("Failed to start PTY: "+err.Error(), session)
		return session
	}

	session.Pid = cmd.Process.Pid
	session.Ptmx = ptmx  // keep reference for resize
	session.Stdin = ptmx // writes to ptmx go to the child's stdin
	log.Printf("[shell] Started PTY process %d: %s", session.Pid, payload.Command)

	// Timeout
	timer := time.AfterFunc(time.Duration(timeoutSec)*time.Second, func() {
		_ = cmd.Process.Signal(os.Kill)
		code := 1
		registry.MarkExited(session.ID, &code, "SIGKILL")
		callbacks.OnError("Command timed out", session)
	})

	// Yield timer
	if payload.YieldMs > 0 && callbacks.OnYield != nil {
		time.AfterFunc(time.Duration(payload.YieldMs)*time.Millisecond, func() {
			if !session.Exited {
				registry.MarkBackgrounded(session.ID)
				callbacks.OnYield(session)
			}
		})
	}

	// Read PTY output
	go func() {
		buf := make([]byte, 8192)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				data := string(buf[:n])
				registry.AppendOutput(session.ID, data)
				if callbacks.OnStdout != nil {
					callbacks.OnStdout(data)
				}
			}
			if err != nil {
				if err != io.EOF {
					log.Printf("[shell] PTY read error: %s", err)
				}
				break
			}
		}
	}()

	// Wait for exit
	go func() {
		err := cmd.Wait()
		timer.Stop()
		_ = ptmx.Close()

		durationMs := time.Since(startTime).Milliseconds()

		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = 1
			}
		}

		registry.MarkExited(session.ID, &exitCode, "")
		if callbacks.OnDone != nil {
			callbacks.OnDone(exitCode, durationMs, session)
		}
	}()

	return session
}
