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

	"github.com/figment-networks/cosmos-indexer/cmd/worker_cosmos/config"
	cli "github.com/figment-networks/cosmos-indexer/worker/client/cosmos"
	"github.com/figment-networks/cosmos-indexer/worker/connectivity"
	grpcIndexer "github.com/figment-networks/cosmos-indexer/worker/transport/grpc"
	grpcProtoIndexer "github.com/figment-networks/cosmos-indexer/worker/transport/grpc/indexer"
)

type flags struct {
	configPath   string
	runMigration bool
	showVersion  bool
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
		panic(fmt.Errorf("error initializing config [ERR: %+v]", err))
	}

	grpcServer := grpc.NewServer() /*
		grpc.KeepaliveEnforcementPolicy(
			keepalive.EnforcementPolicy{
				//			MinTime:             (time.Duration(4000) * time.Second),
				PermitWithoutStream: true,
			},
		))*/

	workerRunID, err := uuid.NewRandom() // UUID V4
	if err != nil {
		log.Fatalf("error generating UUID: %v", err)
	}

	managers := strings.Split(cfg.Managers, ",")

	log.Printf("Hostname is %s", cfg.Hostname)
	hostname := cfg.Hostname
	if hostname == "" {
		hostname = cfg.Address
	}
	c := connectivity.NewWorkerConnections(workerRunID.String(), hostname+":"+cfg.Port, "cosmos", "0.0.1")

	for _, m := range managers {
		c.AddManager(m + "/client_ping")
	}

	go c.Run(context.Background(), time.Second*10)
	workerClient := cli.NewIndexerClient(context.Background(), cfg.TendermintRPCAddr, cfg.DatahubKey)

	worker := grpcIndexer.NewIndexerServer(workerClient)
	grpcProtoIndexer.RegisterIndexerServiceServer(grpcServer, worker)

	lis, err := net.Listen("tcp", "0.0.0.0:"+cfg.Port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
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
