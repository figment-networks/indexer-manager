package main

import (
	"flag"
	"fmt"

	"github.com/figment-networks/cosmos-indexer/cmd/worker/config"
	"github.com/figment-networks/cosmos-indexer/cosmos"
	"github.com/figment-networks/cosmos-indexer/indexer"
)

type flags struct {
	configPath          string
	runMigration        bool
	showVersion         bool
	batchSize           int64
	heightRangeInterval int64
}

var configFlags = flags{}

func init() {
	flag.BoolVar(&configFlags.showVersion, "v", false, "Show application version")
	flag.StringVar(&configFlags.configPath, "config", "", "Path to config")
	flag.BoolVar(&configFlags.runMigration, "migrate", false, "Command to run")
	flag.Int64Var(&configFlags.batchSize, "batch_size", 0, "pipeline batch size")
	flag.Int64Var(&configFlags.heightRangeInterval, "range_int", 0, "pipeline batch size")
	flag.Parse()
}

func main() {
	// Initialize configuration
	cfg, err := initConfig(configFlags.configPath)
	if err != nil {
		panic(fmt.Errorf("error initializing config [ERR: %+v]", err))
	}

	c := cosmos.NewClient(cfg.TendermintRPCAddr, cfg.DatahubKey, nil)

	//	pipeline := indexer.NewPipeline(c, db)
	/*
		err = pipeline.Start(idxConfig(configFlags, cfg))
		if err != nil {
			panic(fmt.Errorf("error starting pipeline [ERR: %+v]", err))
		}
	*/
}

func initConfig(path string) (*config.Config, error) {
	cfg := &config.Config{}
	if path != "" {
		if err := config.FromFile(path, cfg); err != nil {
			return nil, err
		}
	}

	if err := config.FromEnv(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func idxConfig(flags flags, cfg *config.Config) *indexer.Config {
	batchSize := flags.batchSize
	if batchSize == 0 {
		batchSize = cfg.DefaultBatchSize
	}

	heightRange := flags.heightRangeInterval
	if heightRange == 0 {
		heightRange = cfg.DefaultHeightRangeInterval
	}

	return &indexer.Config{
		BatchSize:           batchSize,
		HeightRangeInterval: heightRange,
		StartHeight:         0,
	}
}
