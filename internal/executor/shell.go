package executor

import (
	"context"
	"io"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/TheBotsters/botster-actuator-g/internal/process"
	"github.com/TheBotsters/botster-actuator-g/internal/protocol"
)

const (
	defaultTimeout = 1800 // 30 minutes
	maxTimeout     = 3600 // 1 hour
)

// ShellCallbacks receives events from shell execution.
type ShellCallbacks struct {
	OnStdout func(data string)
	OnStderr func(data string)
	OnDone   func(exitCode int, durationMs int64, session *process.Session)
	OnError  func(err string, session *process.Session)
	OnYield  func(session *process.Session)
}

// ExecuteShell runs a shell command and reports results via callbacks.
// If PTY is requested, delegates to ExecuteShellPTY.
func ExecuteShell(registry *process.Registry, payload protocol.ExecPayload, cwd string, callbacks ShellCallbacks) *process.Session {
	if payload.Pty {
		return ExecuteShellPTY(registry, payload, cwd, callbacks)
	}

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

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutSec)*time.Second)

	cmd := exec.CommandContext(ctx, "sh", "-c", payload.Command)
	cmd.Dir = cmdCwd

	// Merge environment
	cmd.Env = os.Environ()
	for k, v := range payload.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		code := 1
		registry.MarkExited(session.ID, &code, "")
		callbacks.OnError("Failed to create stdout pipe: "+err.Error(), session)
		return session
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		code := 1
		registry.MarkExited(session.ID, &code, "")
		callbacks.OnError("Failed to create stderr pipe: "+err.Error(), session)
		return session
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		code := 1
		registry.MarkExited(session.ID, &code, "")
		callbacks.OnError("Failed to create stdin pipe: "+err.Error(), session)
		return session
	}
	session.Stdin = stdin

	if err := cmd.Start(); err != nil {
		cancel()
		code := 1
		registry.MarkExited(session.ID, &code, "")
		callbacks.OnError("Failed to spawn: "+err.Error(), session)
		return session
	}

	session.Pid = cmd.Process.Pid
	log.Printf("[shell] Started process %d: %s", session.Pid, payload.Command)

	// Set up yield timer
	if payload.YieldMs > 0 && callbacks.OnYield != nil {
		time.AfterFunc(time.Duration(payload.YieldMs)*time.Millisecond, func() {
			if !session.Exited {
				registry.MarkBackgrounded(session.ID)
				callbacks.OnYield(session)
			}
		})
	}

	// Read stdout/stderr in goroutines
	go func() {
		buf := make([]byte, 8192)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				data := string(buf[:n])
				registry.AppendOutput(session.ID, data)
				if callbacks.OnStdout != nil {
					callbacks.OnStdout(data)
				}
			}
			if err != nil {
				break
			}
		}
	}()

	go func() {
		buf := make([]byte, 8192)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				data := string(buf[:n])
				registry.AppendOutput(session.ID, data)
				if callbacks.OnStderr != nil {
					callbacks.OnStderr(data)
				}
			}
			if err != nil {
				break
			}
		}
	}()

	// Wait for completion in a goroutine
	go func() {
		defer cancel()
		err := cmd.Wait()
		durationMs := time.Since(startTime).Milliseconds()

		if ctx.Err() == context.DeadlineExceeded {
			code := 1
			registry.MarkExited(session.ID, &code, "SIGTERM")
			callbacks.OnError("Command timed out after "+string(rune(timeoutSec+'0'))+"s", session)
			return
		}

		exitCode := 0
		exitSignal := ""
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				exitCode = exitErr.ExitCode()
			} else {
				exitCode = 1
			}
		}

		registry.MarkExited(session.ID, &exitCode, exitSignal)
		if callbacks.OnDone != nil {
			callbacks.OnDone(exitCode, durationMs, session)
		}
	}()

	return session
}

// WriteToSession writes data to a session's stdin.
func WriteToSession(session *process.Session, data string) error {
	if session.Stdin == nil {
		return io.ErrClosedPipe
	}
	_, err := session.Stdin.Write([]byte(data))
	return err
}
