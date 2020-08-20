package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/figment-networks/cosmos-indexer/cmd/scheduler/config"

	"github.com/figment-networks/cosmos-indexer/scheduler/core"
	"github.com/figment-networks/cosmos-indexer/scheduler/persistence"
	"github.com/figment-networks/cosmos-indexer/scheduler/persistence/postgresstore"
	"github.com/figment-networks/cosmos-indexer/scheduler/process"
	"github.com/figment-networks/cosmos-indexer/scheduler/runner/lasthash"
	"github.com/figment-networks/cosmos-indexer/scheduler/structures"
)

type flags struct {
	configPath  string
	showVersion bool
}

var configFlags = flags{}

func init() {
	flag.BoolVar(&configFlags.showVersion, "v", false, "Show application version")
	flag.StringVar(&configFlags.configPath, "config", "", "Path to config")
	flag.Parse()
}

func main() {
	ctx := context.Background()
	// Initialize configuration
	cfg, err := initConfig(configFlags.configPath)
	if err != nil {
		log.Fatal(fmt.Errorf("error initializing config [ERR: %+v]", err))
	}

	log.Println("Connecting to ", cfg.DatabaseURL)
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	err = db.PingContext(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	d := postgresstore.NewDriver(db)

	sch := process.NewScheduler()
	c := core.NewCore(persistence.CoreStorage{d}, sch)

	// (lukanus): this might be loaded as plugins ;)
	lh := lasthash.NewClient(persistence.Storage{d})

	c.LoadRunner("lasthash", lh)

	if cfg.InitialConfig != "" {
		file, err := os.Open(cfg.InitialConfig)
		if err != nil {
			log.Fatal(err)
		}

		rcs := []structures.RunConfig{}
		dec := json.NewDecoder(file)
		err = dec.Decode(&rcs)
		file.Close()
		if err != nil {
			log.Fatal(err)
		}

		c.AddSchedules(ctx, rcs)
	}

}

func initConfig(path string) (config.Config, error) {
	cfg := &config.Config{}

	if path != "" {
		if err := config.FromFile(path, cfg); err != nil {
			return *cfg, err
		}
	}

	if err := config.FromEnv(cfg); err != nil {
		return *cfg, err
	}

	return *cfg, nil
}
