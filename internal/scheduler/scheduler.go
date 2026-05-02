package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.Mutex
	running   bool
	onTick    func()
	cronSched *cron.Cron
}

func New(onTick func()) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		ctx:    ctx,
		cancel: cancel,
		onTick: onTick,
	}
}

func (s *Scheduler) StartInterval(seconds int) {
	s.Stop()
	s.mu.Lock()
	s.running = true
	s.mu.Unlock()
	go func() {
		ticker := time.NewTicker(time.Duration(seconds) * time.Second)
		defer ticker.Stop()
		s.onTick()
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
				s.onTick()
			}
		}
	}()
}

func (s *Scheduler) StartCron(expr string) {
	s.Stop()
	if expr == "" {
		return
	}
	s.mu.Lock()
	s.running = true
	s.mu.Unlock()
	s.cronSched = cron.New()
	s.cronSched.AddFunc(expr, s.onTick)
	s.cronSched.Start()
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cronSched != nil {
		s.cronSched.Stop()
		s.cronSched = nil
	}
	s.cancel()
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.running = false
}

func BackgroundContext() context.Context {
	return context.Background()
}

func (s *Scheduler) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}
