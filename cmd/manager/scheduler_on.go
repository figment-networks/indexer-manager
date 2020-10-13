// +build scheduler

package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/figment-networks/indexer-manager/cmd/manager/config"
	"github.com/figment-networks/indexer-manager/manager/client"
	"go.uber.org/zap"

	schedulerCore "github.com/figment-networks/indexer-manager/scheduler/core"
	schedulerDestination "github.com/figment-networks/indexer-manager/scheduler/destination"
	schedulerPersistence "github.com/figment-networks/indexer-manager/scheduler/persistence"
	schedulerPostgres "github.com/figment-networks/indexer-manager/scheduler/persistence/postgresstore"
	schedulerProcess "github.com/figment-networks/indexer-manager/scheduler/process"
	schedulerLastData "github.com/figment-networks/indexer-manager/scheduler/runner/lastdata"
	runnerEmbedded "github.com/figment-networks/indexer-manager/scheduler/runner/transport/embedded"
	schedulerStructures "github.com/figment-networks/indexer-manager/scheduler/structures"
)

func attachScheduler(ctx context.Context, db *sql.DB, mux *http.ServeMux, cfg config.Config, logger *zap.Logger, client client.SchedulerContractor) error {
	logger.Info("[Manager-Scheduler] Adding scheduler...")

	d := schedulerPostgres.NewDriver(db)
	sch := schedulerProcess.NewScheduler(logger)
	c := schedulerCore.NewCore(schedulerPersistence.CoreStorage{Driver: d}, sch, logger)
	scheme := schedulerDestination.NewScheme(logger)
	scheme.RegisterHandles(mux)

	rInternal := runnerEmbedded.NewLastDataInternalTransport(client)

	lh := schedulerLastData.NewClient(schedulerPersistence.Storage{Driver: d}, rInternal)
	c.LoadRunner("lastdata", lh)

	if cfg.SchedulerInitialConfigPath != "" {
		logger.Info("[Manager-Scheduler] Loading initial config")

		files, err := ioutil.ReadDir(cfg.SchedulerInitialConfigPath)
		if err != nil {
			return err
		}

		rcs := []schedulerStructures.RunConfig{}
		for _, fileInfo := range files {
			if fileInfo.IsDir() {
				continue
			}

			file, err := os.Open(path.Join(cfg.SchedulerInitialConfigPath, fileInfo.Name()))
			if err != nil {
				return err
			}

			rcp := []schedulerStructures.RunConfigParams{}
			dec := json.NewDecoder(file)
			err = dec.Decode(&rcp)
			file.Close()
			if err != nil {
				return err
			}

			for _, rConf := range rcp {
				duration, err := time.ParseDuration(rConf.Duration)
				if err != nil {
					return err
				}
				rcs = append(rcs, schedulerStructures.RunConfig{
					Network:  rConf.Network,
					ChainID:  rConf.ChainID,
					Kind:     rConf.Kind,
					Version:  "0.0.1",
					Duration: duration,
				})
			}
		}

		if err := c.AddSchedules(ctx, rcs); err != nil {
			return err
		}
	}
	logger.Info("[Manager-Scheduler] Running Load")

	go reloadScheduler(ctx, logger, c)
	return nil
}

func reloadScheduler(ctx context.Context, logger *zap.Logger, c *schedulerCore.Core) {
	tckr := time.NewTicker(10 * time.Second)
	for range tckr.C {
		if err := c.LoadScheduler(ctx); err != nil {
			logger.Error("[Manager-Scheduler] Error during loading of scheduler", zap.Error(err))
			logger.Sync()
		}
	}
}
