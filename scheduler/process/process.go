package process

import (
	"context"
	"sync"
	"time"
)

type Runner interface {
	Run(ctx context.Context, network, version string) error
}

type Running struct {
	Name string

	CancelF context.CancelFunc
}

type Scheduler struct {
	running map[string]Running
	runlock sync.Mutex
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		running: make(map[string]Running),
	}
}

func (s *Scheduler) Run(ctx context.Context, name string, d time.Duration, network, version string, r Runner) {

	cCtx, cancel := context.WithCancel(ctx)
	tckr := time.NewTicker(d)

	s.runlock.Lock()
	s.running[name] = Running{
		Name:    name,
		CancelF: cancel,
	}
	s.runlock.Unlock()

RUN_LOOP:
	for {
		select {
		case <-tckr.C:
			if err := r.Run(cCtx, network, version); err != nil {
				tckr.Stop()
				break RUN_LOOP
			}
		case <-ctx.Done():
			tckr.Stop()
			break RUN_LOOP
		}
	}

	s.runlock.Lock()
	delete(s.running, name)
	s.runlock.Unlock()
}

func (s *Scheduler) Stop(ctx context.Context, name string) {

	s.runlock.Lock()
	defer s.runlock.Unlock()

	r, ok := s.running[name]
	if ok {
		r.CancelF()
	}

}
