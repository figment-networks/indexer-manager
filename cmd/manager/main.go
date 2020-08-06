package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/figment-networks/cosmos-indexer/cmd/manager/config"
	"github.com/figment-networks/cosmos-indexer/manager/client"
	"github.com/figment-networks/cosmos-indexer/manager/connectivity"
	"github.com/figment-networks/cosmos-indexer/manager/store"
	grpcTransport "github.com/figment-networks/cosmos-indexer/manager/transport/grpc"
	httpTransport "github.com/figment-networks/cosmos-indexer/manager/transport/http"

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
	ctx := context.Background()
	// Initialize configuration
	cfg, err := initConfig(configFlags.configPath)
	if err != nil {
		log.Fatal(fmt.Errorf("error initializing config [ERR: %+v]", err))
	}

	if configFlags.runMigration {
		err := RunMigrations(cfg.DatabaseURL)
		if err != nil {
			log.Fatal(fmt.Errorf("error running migrations [ERR: %+v]", err))
		}
		return // todo should we continue running worker after migration?
	}

	/*
		db, err := store.New(cfg.DatabaseURL)
		if err != nil {
			log.Fatal(fmt.Errorf("error initializing store [ERR: %+v]", err))
		}
	*/ /*
		pipeline := indexer.NewPipeline(c, db)

		err = pipeline.Start(idxConfig(configFlags, cfg))
		if err != nil {
			panic(fmt.Errorf("error starting pipeline [ERR: %+v]", err))
		}
	*/

	connManager := connectivity.NewManager()

	hubbleClient := client.NewHubbleClient()
	grpcCli := grpcTransport.NewClient(hubbleClient)
	connManager.AddTransport(grpcCli)

	go connManager.Run(ctx, hubbleClient.Out())

	hubbleHTTPTransport := httpTransport.NewHubbleConnector(hubbleClient)

	mux := http.NewServeMux()
	hubbleHTTPTransport.AttachToHandler(mux)

	//	attachChecks(db, mux)

	attachConnectionManager(connManager, mux)

	s := &http.Server{
		Addr:    cfg.Address,
		Handler: mux,
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		//	ReadTimeout:  10 * time.Second,
		//	WriteTimeout: 10 * time.Second,
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

/*
func idxConfig(flags flags, cfg config.Config) *indexer.Config {
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
}*/

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

func attachChecks(db *store.Store, mux *http.ServeMux) {

}

// PingInfo contract is defined here
type PingInfo struct {
	ID           string           `json:"id"`
	Kind         string           `json:"kind"`
	Connectivity ConnectivityInfo `json:"connectivity"`
}
type ConnectivityInfo struct {
	Address string `json:"address"`
	Version string `json:"version"`
	Type    string `json:"type"`
}

func attachConnectionManager(mgr *connectivity.Manager, mux *http.ServeMux) {
	b := &bytes.Buffer{}
	block := &sync.Mutex{}
	dec := json.NewDecoder(b)

	mux.HandleFunc("/client_ping", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Received Request")
		pi := &PingInfo{}
		block.Lock()
		b.Reset()
		_, err := b.ReadFrom(r.Body)
		if err != nil {
			block.Unlock()
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = dec.Decode(pi)
		if err != nil {
			block.Unlock()
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		block.Unlock()

		ipTo := net.ParseIP(r.RemoteAddr)
		fwd := r.Header.Get("X-FORWARDED-FOR")
		if fwd != "" {
			ipTo = net.ParseIP(fwd)
		}
		mgr.Register(pi.ID, pi.Kind, connectivity.WorkerConnection{
			Version: pi.Connectivity.Version,
			Type:    pi.Connectivity.Type,
			Addresses: []connectivity.WorkerAddress{{
				IP:      ipTo,
				Address: pi.Connectivity.Address,
			}},
		})
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/get_workers", func(w http.ResponseWriter, r *http.Request) {
		m, err := json.Marshal(mgr.GetAllWorkers())
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "Error marshaling data"}`))
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(m)
		return
	})
}
