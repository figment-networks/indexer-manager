package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	grpc "google.golang.org/grpc"

	"github.com/figment-networks/cosmos-indexer/cmd/worker_cosmos/config"
	"github.com/figment-networks/cosmos-indexer/cmd/worker_cosmos/logger"
	cli "github.com/figment-networks/cosmos-indexer/worker/client/cosmos"
	"github.com/figment-networks/cosmos-indexer/worker/connectivity"
	grpcIndexer "github.com/figment-networks/cosmos-indexer/worker/transport/grpc"
	grpcProtoIndexer "github.com/figment-networks/cosmos-indexer/worker/transport/grpc/indexer"

	"github.com/figment-networks/indexing-engine/metrics"
	"github.com/figment-networks/indexing-engine/metrics/prometheusmetrics"
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
	// Initialize configuration
	cfg, err := initConfig(configFlags.configPath)
	if err != nil {
		log.Fatalf("error initializing config [ERR: %v]", err.Error())
	}

	if cfg.AppEnv == "development" {
		logger.Init("console", "debug", []string{"stderr"})
	} else {
		logger.Init("json", "info", []string{"stderr"})
	}

	defer logger.Sync()

	prom := prometheusmetrics.New()
	err = metrics.AddEngine(prom)
	if err != nil {
		logger.Error(err)
	}
	err = metrics.Hotload(prom.Name())
	if err != nil {
		logger.Error(err)
	}

	workerRunID, err := uuid.NewRandom() // UUID V4
	if err != nil {
		logger.Error(fmt.Errorf("error generating UUID: %w", err))
		return
	}

	managers := strings.Split(cfg.Managers, ",")
	hostname := cfg.Hostname
	if hostname == "" {
		hostname = cfg.Address
	}

	logger.Info(fmt.Sprintf("Self-hostname (%s) is %s:%s ", workerRunID.String(), hostname, cfg.Port))

	c := connectivity.NewWorkerConnections(workerRunID.String(), hostname+":"+cfg.Port, "cosmos", "0.0.1")
	for _, m := range managers {
		c.AddManager(m + "/client_ping")
	}

	logger.Info(fmt.Sprintf("Connecting to managers (%s)", strings.Join(managers, ",")))

	go c.Run(context.Background(), logger.GetLogger(), cfg.ManagerInverval)

	grpcServer := grpc.NewServer()

	workerClient := cli.NewIndexerClient(context.Background(), logger.GetLogger(), cfg.TendermintRPCAddr, cfg.DatahubKey)

	worker := grpcIndexer.NewIndexerServer(workerClient, logger.GetLogger())
	grpcProtoIndexer.RegisterIndexerServiceServer(grpcServer, worker)

	s := &http.Server{
		Addr:         "0.0.0.0:" + cfg.HTTPPort,
		Handler:      metrics.Handler(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	osSig := make(chan os.Signal)
	exit := make(chan string, 2)
	signal.Notify(osSig, syscall.SIGTERM)
	signal.Notify(osSig, syscall.SIGINT)

	go runGRPC(grpcServer, cfg, exit)
	go runHTTP(s, cfg, exit)
	ctx := context.Background()

RUN_LOOP:
	for {
		select {
		case <-osSig:
			grpcServer.GracefulStop()
			s.Shutdown(ctx)
			break RUN_LOOP
		case k := <-exit:
			if k == "grpc" { // (lukanus): when grpc is finished, stop http and vice versa
				s.Shutdown(ctx)
			} else {
				grpcServer.GracefulStop()
			}
			break RUN_LOOP
		}
	}

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

func runGRPC(grpcServer *grpc.Server, cfg *config.Config, exit chan<- string) {
	log.Printf("[GRPC] Listening on 0.0.0.0:%s", cfg.Port)
	lis, err := net.Listen("tcp", "0.0.0.0:"+cfg.Port)
	if err != nil {
		log.Println("failed to listen: %w", err)
		exit <- "grpc"
		return
	}

	// (lukanus): blocking call on grpc server
	grpcServer.Serve(lis)
	exit <- "grpc"
}

func runHTTP(s *http.Server, cfg *config.Config, exit chan<- string) {
	log.Printf("[HTTP] Listening on 0.0.0.0:%s", cfg.HTTPPort)

	if err := s.ListenAndServe(); err != nil {
		log.Println("failed to listen: %w", err)
	}
	exit <- "http"
}
