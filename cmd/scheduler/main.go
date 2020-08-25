package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/figment-networks/cosmos-indexer/cmd/scheduler/config"

	"github.com/figment-networks/cosmos-indexer/scheduler/core"
	"github.com/figment-networks/cosmos-indexer/scheduler/destination"
	"github.com/figment-networks/cosmos-indexer/scheduler/persistence"
	"github.com/figment-networks/cosmos-indexer/scheduler/persistence/postgresstore"
	"github.com/figment-networks/cosmos-indexer/scheduler/process"
	"github.com/figment-networks/cosmos-indexer/scheduler/runner/lastdata"
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
		return
	}
	err = db.PingContext(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	d := postgresstore.NewDriver(db)

	sch := process.NewScheduler()

	c := core.NewCore(persistence.CoreStorage{d}, sch)

	scheme := destination.NewScheme()

	managers := strings.Split(cfg.Managers, ",")
	if len(managers) == 0 {
		log.Fatal("There is no manager to connect to")
		return
	}

	go recheck(scheme, managers, time.Second*20)

	// (lukanus): this might be loaded as plugins ;)
	lh := lastdata.NewClient(persistence.Storage{d}, scheme)
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

	s := &http.Server{
		Addr:    cfg.Address,
		Handler: mux,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	log.Printf("Running server on %s", cfg.Address)
	log.Fatal(s.ListenAndServe())

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

func recheck(scheme *destination.Scheme, addresses []string, dur time.Duration) (config.Config, error) {
	for _, a := range addresses {
		scheme.AddManager(a)
	}

	tckr := time.NewTicker(dur)
	for {
		select {
		case <-tckr.C:
			scheme.Refresh()
		}
	}

}
