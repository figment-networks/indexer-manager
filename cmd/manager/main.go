package main

import (
	"context"
	"crypto/tls"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/figment-networks/cosmos-indexer/manager/client"
	"github.com/figment-networks/cosmos-indexer/manager/connectivity"
	"github.com/figment-networks/cosmos-indexer/manager/store"
	"github.com/figment-networks/cosmos-indexer/manager/store/postgres"
	grpcTransport "github.com/figment-networks/cosmos-indexer/manager/transport/grpc"
	httpTransport "github.com/figment-networks/cosmos-indexer/manager/transport/http"
	"github.com/figment-networks/indexing-engine/metrics"
	"github.com/figment-networks/indexing-engine/metrics/prometheusmetrics"

	"github.com/figment-networks/cosmos-indexer/cmd/common/logger"
	"github.com/figment-networks/cosmos-indexer/cmd/manager/config"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
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

	// setup metrics
	prom := prometheusmetrics.New()
	err = metrics.AddEngine(prom)
	if err != nil {
		logger.Error(err)
	}
	err = metrics.Hotload(prom.Name())
	if err != nil {
		logger.Error(err)
	}

	// connect to database
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

	pgsqlDriver := postgres.NewDriver(ctx, db)
	managerStore := store.New(pgsqlDriver)
	go managerStore.Run(ctx, time.Second*5)

	// Initialize manager
	mID, _ := uuid.NewRandom()
	connManager := connectivity.NewManager(mID.String(), logger.GetLogger())

	// setup grpc transport
	grpcCli := grpcTransport.NewClient()
	connManager.AddTransport(grpcCli)

	client.InitMetrics()
	hClient := client.NewClient(managerStore, logger.GetLogger(), client.NewRunner())
	hClient.LinkSender(connManager)
	HTTPTransport := httpTransport.NewConnector(hClient)

	mux := http.NewServeMux()
	HTTPTransport.AttachToHandler(mux)

	connManager.AttachToMux(mux)

	attachHealthCheck(ctx, mux, db)
	attachProfiling(mux)

	// (lukanus): only after passing param, conditionally enable scheduler
	// this is for the scenario when manager is *the only* instance working.
	if cfg.EnableScheduler {
		if err := attachScheduler(ctx, db, mux, cfg, logger.GetLogger(), hClient); err != nil {
			log.Fatal(err)
		}
	}

	mux.Handle("/metrics", metrics.Handler())

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

RunLoop:
	for {
		select {
		case <-osSig:
			s.Shutdown(ctx)
			break RunLoop
		case <-exit:
			break RunLoop
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

	if cfg.DatabaseURL != "" {
		return *cfg, nil
	}

	if err := config.FromEnv(cfg); err != nil {
		return *cfg, err
	}

	return *cfg, nil
}

func runHTTP(s *http.Server, address string, logger *zap.Logger, exit chan<- string) {
	defer logger.Sync()

	logger.Info(fmt.Sprintf("[HTTP] Listening on %s", address))

	if err := s.ListenAndServe(); err != nil {
		logger.Error("[HTTP] failed to listen", zap.Error(err))
	}
	exit <- "http"
}
