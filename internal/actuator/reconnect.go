package actuator

import (
	"math"
	"math/rand"
	"time"
	"log"
)

// ReconnectManager handles exponential backoff with jitter for WebSocket reconnection.
type ReconnectManager struct {
	attempt     int
	baseMs      float64
	maxMs       float64
	maxAttempts int
	timer       *time.Timer
}

// ReconnectOptions configures reconnection behavior.
type ReconnectOptions struct {
	BaseMs      float64
	MaxMs       float64
	MaxAttempts int
}

// NewReconnectManager creates a new ReconnectManager with the given options.
func NewReconnectManager(opts ReconnectOptions) *ReconnectManager {
	if opts.BaseMs == 0 {
		opts.BaseMs = 1000
	}
	if opts.MaxMs == 0 {
		opts.MaxMs = 30000
	}
	if opts.MaxAttempts == 0 {
		opts.MaxAttempts = -1 // infinite
	}
	return &ReconnectManager{
		baseMs:      opts.BaseMs,
		maxMs:       opts.MaxMs,
		maxAttempts: opts.MaxAttempts,
	}
}

// Schedule schedules a reconnection attempt. Returns false if max attempts reached.
func (r *ReconnectManager) Schedule(fn func()) bool {
	if r.maxAttempts > 0 && r.attempt >= r.maxAttempts {
		return false
	}

	delay := math.Min(
		r.baseMs*math.Pow(2, float64(r.attempt))+rand.Float64()*1000,
		r.maxMs,
	)
	r.attempt++

	log.Printf("[reconnect] Attempt %d in %dms", r.attempt, int(delay))
	r.timer = time.AfterFunc(time.Duration(delay)*time.Millisecond, fn)
	return true
}

// Reset resets the attempt counter and cancels any pending reconnection.
func (r *ReconnectManager) Reset() {
	r.attempt = 0
	if r.timer != nil {
		r.timer.Stop()
		r.timer = nil
	}
}

// Destroy cancels any pending reconnection and resets state.
func (r *ReconnectManager) Destroy() {
	r.Reset()
}
