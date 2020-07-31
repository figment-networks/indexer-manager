package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/figment-networks/cosmos-indexer/cmd/manager/config"
	"github.com/figment-networks/cosmos-indexer/indexer"
	"github.com/figment-networks/cosmos-indexer/store"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
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

	if configFlags.runMigration {
		err := RunMigrations(cfg.DatabaseURL)
		if err != nil {
			panic(fmt.Errorf("error running migrations [ERR: %+v]", err))
		}
		return // todo should we continue running worker after migration?
	}
	/*
		c := cosmos.NewClient(cfg.TendermintRPCAddr, cfg.DatahubKey, nil)
	*/
	db, err := store.New(cfg.DatabaseURL)
	if err != nil {
		panic(fmt.Errorf("error initializing store [ERR: %+v]", err))
	}

	pipeline := indexer.NewPipeline(c, db)

	err = pipeline.Start(idxConfig(configFlags, cfg))
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

func RunMigrations(dbURL string) error {
	log.Println("getting current directory")
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	srcDir := filepath.Join(dir, "migrations")
	srcPath := fmt.Sprintf("file://%s", srcDir) //todo pass file location in config? or move this package or migrations folder
	log.Println("using migrations from", srcPath)
	m, err := migrate.New(srcPath, dbURL)
	if err != nil {
		return err
	}

	log.Println("running migrations")
	return m.Up()
}
