package limiter

import (
	"context"
)

// New returns a limiter with given capacity
func New(capacity int) *Limiter {
	return &Limiter{
		ch: make(chan struct{}, capacity),
	}
}

// Limiter is used for concurrent contorl
type Limiter struct {
	ch chan struct{}
}

// Acquire adds a concurrent counter
func (l *Limiter) Acquire(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false

	case l.ch <- struct{}{}:
		return true
	}
}

// Release reduces a concurrent counter
func (l *Limiter) Release(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false

	case <-l.ch:
		return true
	}
}
