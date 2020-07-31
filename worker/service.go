package main

import (
	"flag"
	"fmt"

	"github.com/figment-networks/cosmos-indexer/cosmos"
	"github.com/figment-networks/cosmos-indexer/indexer"
	"github.com/figment-networks/cosmos-indexer/store"
	"github.com/figment-networks/cosmos-indexer/worker/config"
	"github.com/figment-networks/cosmos-indexer/worker/migration"
)

type flags struct {
	configPath          string
	runMigration        bool
	showVersion         bool
	batchSize           int64
	heightRangeInterval int64
}

func (c *flags) Setup() {
	flag.BoolVar(&c.showVersion, "v", false, "Show application version")
	flag.StringVar(&c.configPath, "config", "", "Path to config")
	flag.BoolVar(&c.runMigration, "migrate", false, "Command to run")
	flag.Int64Var(&c.batchSize, "batch_size", 0, "pipeline batch size")
	flag.Int64Var(&c.heightRangeInterval, "range_int", 0, "pipeline batch size")
}

func main() {
	flags := flags{}
	flags.Setup()
	flag.Parse()

	if flags.showVersion {
		fmt.Println(config.VersionString())
		return
	}

	// Initialize configuration
	cfg, err := initConfig(flags.configPath)
	if err != nil {
		panic(fmt.Errorf("error initializing config [ERR: %+v]", err))
	}

	if flags.runMigration {
		err := migration.Run(cfg.DatabaseURL)
		if err != nil {
			panic(fmt.Errorf("error running migrations [ERR: %+v]", err))
		}
		return // todo should we continue running worker after migration?
	}

	c := cosmos.NewClient(cfg.TendermintRPCAddr, cfg.DatahubKey, nil)

	db, err := store.New(cfg.DatabaseURL)
	if err != nil {
		panic(fmt.Errorf("error initializing store [ERR: %+v]", err))
	}

	pipeline := indexer.NewPipeline(c, db)

	err = pipeline.Start(idxConfig(flags, cfg))
	if err != nil {
		panic(fmt.Errorf("error starting pipeline [ERR: %+v]", err))
	}
}

func initConfig(path string) (*config.Config, error) {
	cfg := config.New()

	if err := config.FromEnv(cfg); err != nil {
		return nil, err
	}

	if path != "" {
		if err := config.FromFile(path, cfg); err != nil {
			return nil, err
		}
	}

	if err := cfg.Validate(); err != nil {
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
