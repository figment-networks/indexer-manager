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
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/figment-networks/cosmos-indexer/cmd/scheduler/config"
	"github.com/figment-networks/cosmos-indexer/cmd/scheduler/logger"
	"go.uber.org/zap"

	"github.com/figment-networks/cosmos-indexer/scheduler/core"
	"github.com/figment-networks/cosmos-indexer/scheduler/destination"
	"github.com/figment-networks/cosmos-indexer/scheduler/persistence"
	"github.com/figment-networks/cosmos-indexer/scheduler/persistence/postgresstore"
	"github.com/figment-networks/cosmos-indexer/scheduler/process"
	"github.com/figment-networks/cosmos-indexer/scheduler/runner/lastdata"
	runnerHTTP "github.com/figment-networks/cosmos-indexer/scheduler/runner/transport/http"
	"github.com/figment-networks/cosmos-indexer/scheduler/structures"

	_ "github.com/lib/pq"
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

	if cfg.AppEnv == "development" {
		logger.Init("console", "debug", []string{"stderr"})
	} else {
		logger.Init("json", "info", []string{"stderr"})
	}

	defer logger.Sync()

	logger.Info("[DB] Connecting to database...")
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		logger.Error(err)
		return
	}

	if err := db.PingContext(ctx); err != nil {
		logger.Error(err)
		return
	}
	logger.Info("[DB] Ping successfull...")
	defer db.Close()

	mux := http.NewServeMux()
	d := postgresstore.NewDriver(db)

	sch := process.NewScheduler(logger.GetLogger())

	c := core.NewCore(persistence.CoreStorage{Driver: d}, sch, logger.GetLogger())

	scheme := destination.NewScheme(logger.GetLogger())
	scheme.RegisterHandles(mux)

	managers := strings.Split(cfg.Managers, ",")
	if len(managers) == 0 {
		logger.GetLogger().Error("There is no manager to connect to")
		return
	}

	go recheck(ctx, logger.GetLogger(), scheme, managers, time.Second*20)

	rHTTP := runnerHTTP.NewLastDataHTTPTransport(scheme)
	// (lukanus): this might be loaded as plugins ;)
	lh := lastdata.NewClient(persistence.Storage{Driver: d}, rHTTP)
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

	osSig := make(chan os.Signal)
	exit := make(chan string, 2)
	signal.Notify(osSig, syscall.SIGTERM)
	signal.Notify(osSig, syscall.SIGINT)

	go runHTTP(s, cfg.Address, logger.GetLogger(), exit)

RUN_LOOP:
	for {
		select {
		case <-osSig:
			s.Shutdown(ctx)
			break RUN_LOOP
		case <-exit:
			break RUN_LOOP
		}
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

func recheck(ctx context.Context, logger *zap.Logger, scheme *destination.Scheme, addresses []string, dur time.Duration) (config.Config, error) {
	for _, a := range addresses {
		scheme.AddManager(a)
	}

	tckr := time.NewTicker(dur)
	for {
		select {
		case <-ctx.Done():
			break
		case <-tckr.C:
			ctx, cancel := context.WithTimeout(ctx, time.Second*5)
			if err := scheme.Refresh(ctx); err != nil {
				logger.Error("Error on schema refresh", zap.Error(err))
			}
			cancel()
		}
	}
}

func runHTTP(s *http.Server, address string, logger *zap.Logger, exit chan<- string) {
	defer logger.Sync()

	logger.Info(fmt.Sprintf("[HTTP] Listening on %s", address))

	if err := s.ListenAndServe(); err != nil {
		logger.Error("[HTTP] failed to listen", zap.Error(err))
	}
	exit <- "http"
}
