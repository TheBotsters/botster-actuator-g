// Package process provides a registry for tracking active and backgrounded shell sessions.
package process

import (
	"io"
	"math/rand"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/TheBotsters/botster-actuator-g/internal/protocol"
)

const (
	MaxOutputChars = 200_000
	TailChars      = 4000
)

// Session tracks an active or completed process.
type Session struct {
	ID           string
	Command      string
	Pid          int
	StartedAt    int64
	Cwd          string
	Exited       bool
	ExitCode     *int
	ExitSignal   string
	Backgrounded bool

	Aggregated strings.Builder
	tail       string
	Stdin      io.Writer

	mu sync.Mutex
}

// Registry tracks all process sessions.
type Registry struct {
	sessions map[string]*Session
	mu       sync.RWMutex
}

// NewRegistry creates a new process registry.
func NewRegistry() *Registry {
	return &Registry{
		sessions: make(map[string]*Session),
	}
}

// AddSession creates and registers a new session.
func (r *Registry) AddSession(command, cwd string, pid int) *Session {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := r.createSlug()
	s := &Session{
		ID:        id,
		Command:   command,
		Pid:       pid,
		StartedAt: time.Now().UnixMilli(),
		Cwd:       cwd,
	}
	r.sessions[id] = s
	return s
}

// GetSession returns a session by ID, or nil if not found.
func (r *Registry) GetSession(id string) *Session {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sessions[id]
}

// ListSessions returns all tracked sessions.
func (r *Registry) ListSessions() []*Session {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Session, 0, len(r.sessions))
	for _, s := range r.sessions {
		result = append(result, s)
	}
	return result
}

// MarkBackgrounded marks a session as backgrounded.
func (r *Registry) MarkBackgrounded(id string) bool {
	s := r.GetSession(id)
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Backgrounded = true
	return true
}

// MarkExited marks a session as exited with the given code and signal.
func (r *Registry) MarkExited(id string, exitCode *int, exitSignal string) bool {
	s := r.GetSession(id)
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Exited = true
	s.ExitCode = exitCode
	s.ExitSignal = exitSignal
	s.Stdin = nil
	return true
}

// AppendOutput appends data to a session's output buffer with truncation.
func (r *Registry) AppendOutput(id string, data string) bool {
	s := r.GetSession(id)
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Aggregated.WriteString(data)
	if s.Aggregated.Len() > MaxOutputChars {
		trimmed := s.Aggregated.String()[s.Aggregated.Len()-MaxOutputChars:]
		s.Aggregated.Reset()
		s.Aggregated.WriteString(trimmed)
	}
	r.updateTail(s)
	return true
}

// Tail returns the tail output for a session.
func (r *Registry) Tail(id string) string {
	s := r.GetSession(id)
	if s == nil {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tail
}

// KillSession sends SIGTERM to a session's process, followed by SIGKILL after 5s.
func (r *Registry) KillSession(id string) bool {
	s := r.GetSession(id)
	if s == nil || s.Pid == 0 {
		return false
	}

	if err := syscall.Kill(s.Pid, syscall.SIGTERM); err != nil {
		return false
	}

	go func() {
		time.Sleep(5 * time.Second)
		s.mu.Lock()
		defer s.mu.Unlock()
		if !s.Exited && s.Pid != 0 {
			_ = syscall.Kill(s.Pid, syscall.SIGKILL)
		}
	}()

	return true
}

// ToProcessInfo converts a Session to a protocol ProcessInfo.
func (r *Registry) ToProcessInfo(s *Session) protocol.ProcessInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	return protocol.ProcessInfo{
		ID:           s.ID,
		Command:      s.Command,
		Pid:          s.Pid,
		StartedAt:    s.StartedAt,
		Cwd:          s.Cwd,
		Exited:       s.Exited,
		ExitCode:     s.ExitCode,
		ExitSignal:   s.ExitSignal,
		Backgrounded: s.Backgrounded,
	}
}

// GetOutput returns the full aggregated output for a session.
func (r *Registry) GetOutput(id string) string {
	s := r.GetSession(id)
	if s == nil {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Aggregated.String()
}

func (r *Registry) updateTail(s *Session) {
	out := s.Aggregated.String()
	if len(out) <= TailChars {
		s.tail = out
	} else {
		s.tail = out[len(out)-TailChars:]
	}
}

func (r *Registry) createSlug() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	for {
		var b strings.Builder
		for i := 0; i < 8; i++ {
			b.WriteByte(chars[rand.Intn(len(chars))])
		}
		slug := b.String()
		if _, exists := r.sessions[slug]; !exists {
			return slug
		}
	}
}
