package core

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/figment-networks/cosmos-indexer/scheduler/persistence"
	"github.com/figment-networks/cosmos-indexer/scheduler/process"
	"github.com/figment-networks/cosmos-indexer/scheduler/structures"
	"go.uber.org/zap"

	"github.com/google/uuid"
)

type Status string

var (
	ErrAlreadyEnabled = errors.New("this schedule is already enabled")

	StatusEnabled  Status = "enabled"
	StatusDisabled Status = "disabled"
)

type RunInfo struct {
	structures.RunConfig

	Status Status
}

type Core struct {
	ID      uuid.UUID
	run     map[uuid.UUID]*RunInfo
	runLock sync.RWMutex

	runners map[string]process.Runner

	logger *zap.Logger

	store persistence.CoreStorage

	scheduler *process.Scheduler
}

func NewCore(store persistence.CoreStorage, scheduler *process.Scheduler, logger *zap.Logger) *Core {
	u, _ := uuid.NewRandom()
	return &Core{
		ID:        u,
		store:     store,
		scheduler: scheduler,
		logger:    logger,
		runners:   map[string]process.Runner{},
	}
}

func (c *Core) LoadRunner(name string, runner process.Runner) {
	c.runLock.Lock()
	defer c.runLock.Unlock()

	c.runners[name] = runner
}

func (c *Core) AddSchedules(ctx context.Context, rcs []structures.RunConfig) error {
	c.runLock.Lock()
	defer c.runLock.Unlock()

	for _, r := range rcs {
		if err := c.store.AddConfig(ctx, r); err != nil {
			log.Printf("Add Config errored: %s", err.Error())
		}
	}

	return nil
}

func (c *Core) LoadScheduler(ctx context.Context) ([]structures.RunConfig, error) {
	c.runLock.Lock()
	defer c.runLock.Unlock()

	rcs, err := c.store.GetConfigs(ctx, c.ID)
	if err != nil {
		return nil, err
	}

	for _, s := range rcs {
		runner, ok := c.runners[s.Kind]
		if !ok {
			c.logger.Error(fmt.Sprintf("[Core] There is no such type as %s", s.Kind))
			continue
		}
		r, ok := c.run[s.ID]
		if !ok {
			r = &RunInfo{
				RunConfig: s,
			}
			c.run[s.ID] = r
		}

		if r.Status == StatusEnabled {
			c.logger.Error("[Core] Schedule already enabled")
			continue
		}

		// In fact run scheduler
		go c.scheduler.Run(ctx, s.ID.String(), r.Duration, r.Network, r.Version, runner)
		err := c.store.MarkRunning(ctx, s.RunID, s.ID)
		if err != nil {
			c.logger.Error("[Core] Error setting state running", zap.Error(err))
		}
	}

	return nil, nil
}

func (c *Core) ListSchedule() []structures.RunConfig {
	c.runLock.RLock()
	defer c.runLock.RUnlock()

	m := make([]structures.RunConfig, len(c.run))
	/*for k, v := range c.run {
		m = append(m, v)
	}*/
	return m
}

func (c *Core) EnableSchedule(ctx context.Context, sID uuid.UUID) error {
	c.runLock.Lock()
	defer c.runLock.Unlock()

	r, ok := c.run[sID]
	if !ok {
		return errors.New("there is no such schedule to enable")
	}

	if r.Status == StatusEnabled {
		return ErrAlreadyEnabled
	}

	runner, _ := c.runners[r.Kind]
	go c.scheduler.Run(ctx, sID.String(), r.Duration, r.Network, r.Version, runner)

	return nil
}

func (c *Core) DisableSchedule(new structures.RunConfig) {

}

func (c *Core) handlerListSchedule(w http.ResponseWriter, r *http.Request) {
	//schedule := c.ListSchedule()
	//for _, var := range var {

	//}
}

func (c *Core) handlerAddSchedule(w http.ResponseWriter, r *http.Request) {

}

func (c *Core) RegisterHandles(smux *http.ServeMux) {
	smux.HandleFunc("/scheduler/core/add", c.handlerAddSchedule)
	smux.HandleFunc("/scheduler/core/list", c.handlerListSchedule)
}