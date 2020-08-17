package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	"github.com/figment-networks/cosmos-indexer/cmd/worker_coda/config"
	cli "github.com/figment-networks/cosmos-indexer/worker/client/coda"
	"github.com/figment-networks/cosmos-indexer/worker/connectivity"
	grpcIndexer "github.com/figment-networks/cosmos-indexer/worker/transport/grpc"
	grpcProtoIndexer "github.com/figment-networks/cosmos-indexer/worker/transport/grpc/indexer"
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

	grpcServer := grpc.NewServer(
		grpc.KeepaliveEnforcementPolicy(
			keepalive.EnforcementPolicy{
				MinTime:             (time.Duration(4000) * time.Second),
				PermitWithoutStream: true,
			},
		))

	workerRunID, err := uuid.NewRandom() // UUID V4
	if err != nil {
		log.Fatalf("error generating UUID: %v", err)
	}

	managers := strings.Split(cfg.Managers, ",")

	c := connectivity.NewWorkerConnections(workerRunID.String(), cfg.Address, "coda", "0.0.1")

	for _, m := range managers {
		c.AddManager(m + "/client_ping")
	}

	go c.Run(context.Background(), time.Second*10)

	workerClient := cli.NewIndexerClient(context.Background(), cfg.CodaEndpoint)

	worker := grpcIndexer.NewIndexerServer(workerClient)
	grpcProtoIndexer.RegisterIndexerServiceServer(grpcServer, worker)

	lis, err := net.Listen("tcp", cfg.Address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Printf("Listening on %s", cfg.Address)
	// (lukanus): blocking call on grpc server
	grpcServer.Serve(lis)
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
