package limiter

import (
	"context"
	"sync/atomic"

	"github.com/hashicorp/go-multierror"

	"github.com/dtynn/londobell/common"
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

// NewParallel constructs a parallel executor
func NewParallel(ctx context.Context, capability int) *Parallel {
	innerCtx, innerCancel := context.WithCancel(ctx)
	return &Parallel{
		ctx:     innerCtx,
		cancel:  innerCancel,
		limiter: New(capability),
	}
}

// Parallel executes jobs with given context and concurrent limit
type Parallel struct {
	ctx      context.Context
	cancel   context.CancelFunc
	limiter  *Limiter
	wg       multierror.Group
	errCount int32
}

// P executes a given job in parallel
func (p *Parallel) P(fn func(ctx context.Context) error) {
	p.wg.Go(func() error {
		if !p.limiter.Acquire(p.ctx) {
			return nil
		}

		defer func() {
			p.limiter.Release(p.ctx)
		}()

		select {
		case <-p.ctx.Done():
			return nil

		default:

		}

		err := fn(p.ctx)
		if err == nil {
			return nil
		}

		p.cancel()

		if atomic.AddInt32(&p.errCount, 1) == 1 {
			return err
		}

		return common.NonCtxCanceledErr(err)
	})
}

// Wait waits for all scheduled jobs to be done
func (p *Parallel) Wait() error {
	err := p.wg.Wait()
	if err != nil {
		return err
	}

	return nil
}

// Finish finalizes the context
func (p *Parallel) Finish() {
	p.cancel()
}
