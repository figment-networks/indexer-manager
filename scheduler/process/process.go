package process

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/figment-networks/cosmos-indexer/scheduler/structures"
	"go.uber.org/zap"
)

type Runner interface {
	Run(ctx context.Context, network, chain, version string) error
	Name() string
}

type Running struct {
	Name string

	CancelFunc context.CancelFunc
}

type Scheduler struct {
	running map[string]Running
	runlock sync.Mutex
	logger  *zap.Logger
}

func NewScheduler(logger *zap.Logger) *Scheduler {
	return &Scheduler{
		running: make(map[string]Running),
		logger:  logger,
	}
}

func (s *Scheduler) Run(ctx context.Context, name string, d time.Duration, network, chainID, version string, r Runner) {
	cCtx, cancel := context.WithCancel(ctx)
	tckr := time.NewTicker(d)

	s.runlock.Lock()
	s.running[name] = Running{
		Name:       name,
		CancelFunc: cancel,
	}
	s.runlock.Unlock()

RunLoop:
	for {
		select {
		case <-tckr.C:
			if err := r.Run(cCtx, network, chainID, version); err != nil {
				var rErr *structures.RunError
				s.logger.Error("[Scheduler - Process] Error running  "+name, zap.Error(err))
				if errors.As(err, &rErr) {
					if !rErr.IsRecoverable() {
						tckr.Stop()
						break RunLoop
					}
				}
			}
		case <-cCtx.Done():
			tckr.Stop()
			break RunLoop
		case <-ctx.Done():
			tckr.Stop()
			break RunLoop
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
		r.CancelFunc()
	}
}
