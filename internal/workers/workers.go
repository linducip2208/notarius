// Package workers runs background jobs (notifications, report builds)
// with plain Go goroutines. No external queue required for v1.
package workers

import (
	"context"
	"log/slog"
	"sync"
)

// Job is a unit of background work.
type Job func(ctx context.Context) error

// Pool runs jobs with bounded concurrency.
type Pool struct {
	ch  chan Job
	wg  sync.WaitGroup
	log *slog.Logger
}

func New(n int, log *slog.Logger) *Pool {
	p := &Pool{ch: make(chan Job, 256), log: log}
	for i := 0; i < n; i++ {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			for j := range p.ch {
				if err := j(context.Background()); err != nil {
					p.log.Error("background job failed", "err", err)
				}
			}
		}()
	}
	return p
}

// Submit enqueues a job; it never blocks the request path fatally.
func (p *Pool) Submit(j Job) { p.ch <- j }

// Shutdown drains the queue.
func (p *Pool) Shutdown() { close(p.ch); p.wg.Wait() }
