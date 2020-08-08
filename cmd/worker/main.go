package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/google/uuid"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	"github.com/figment-networks/cosmos-indexer/cmd/worker/config"
	"github.com/figment-networks/cosmos-indexer/worker/api/tendermint"
	"github.com/figment-networks/cosmos-indexer/worker/client"
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

	//	c := cosmos.NewClient(cfg.TendermintRPCAddr, cfg.DatahubKey, nil)

	//	pipeline := indexer.NewPipeline(c, db)
	/*
		err = pipeline.Start(idxConfig(configFlags, cfg))
		if err != nil {
			panic(fmt.Errorf("error starting pipeline [ERR: %+v]", err))
		}
	*/
	/*
		creds, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
		if err != nil {
			log.Fatalf("Failed to generate credentials %v", err)
		}
		opts = []grpc.ServerOption{grpc.Creds(creds)}
	*/
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

	c := connectivity.NewWorkerConnections(workerRunID.String(), cfg.Address)
	c.AddManager("localhost:8085/client_ping")

	go c.Run(context.Background(), time.Second*10)

	tendermintClient := tendermint.NewClient(cfg.TendermintRPCAddr, cfg.DatahubKey, nil)

	workerClient := client.NewIndexerClient(context.Background(), tendermintClient)

	worker := grpcIndexer.NewIndexerServer(workerClient)
	grpcProtoIndexer.RegisterIndexerServiceServer(grpcServer, worker)

	//address := fmt.Sprintf("localhost:%d", 3000)
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

/*
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
*/
