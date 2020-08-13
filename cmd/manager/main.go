package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/figment-networks/cosmos-indexer/cmd/manager/config"
	"github.com/figment-networks/cosmos-indexer/manager/client"
	"github.com/figment-networks/cosmos-indexer/manager/connectivity"
	"github.com/figment-networks/cosmos-indexer/manager/store"
	"github.com/figment-networks/cosmos-indexer/manager/store/postgres"
	grpcTransport "github.com/figment-networks/cosmos-indexer/manager/transport/grpc"
	httpTransport "github.com/figment-networks/cosmos-indexer/manager/transport/http"
	/*
		"github.com/golang-migrate/migrate/v4"
		"github.com/golang-migrate/migrate/v4/database/postgres"
		_ "github.com/golang-migrate/migrate/v4/database/postgres"
		_ "github.com/golang-migrate/migrate/v4/source/file"
	*/)

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
	//	flag.Int64Var(&configFlags.batchSize, "batch_size", 0, "pipeline batch size")
	//	flag.Int64Var(&configFlags.heightRangeInterval, "range_int", 0, "pipeline batch size")
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

	pgsqlDriver := postgres.New(ctx, db)
	managerStore := store.New(pgsqlDriver)
	go managerStore.Run(ctx, time.Second*5)

	connManager := connectivity.NewManager()

	grpcCli := grpcTransport.NewClient()
	connManager.AddTransport(grpcCli)
	go connManager.Run(ctx)

	hubbleClient := client.NewHubbleClient(managerStore)
	hubbleClient.LinkSender(connManager)
	hubbleHTTPTransport := httpTransport.NewHubbleConnector(hubbleClient)

	mux := http.NewServeMux()
	hubbleHTTPTransport.AttachToHandler(mux)

	attachChecks(managerStore, mux)

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

		log.Printf("Received Ping from %s:%s (%s) ", pi.Kind, pi.Connectivity.Version, pi.Connectivity.Type)
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
