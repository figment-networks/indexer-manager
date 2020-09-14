package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/figment-networks/cosmos-indexer/cmd/manager-migration/config"
	"github.com/golang-migrate/migrate"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

type flags struct {
	configPath string
	version    uint
}

var configFlags = flags{}

func init() {
	flag.StringVar(&configFlags.configPath, "config", "", "Path to config")
	flag.UintVar(&configFlags.version, "version", 0, "Version parameter sets db changes to specified revision (up or down)")
	flag.Parse()
}

func main() {

	// Initialize configuration
	cfg, err := initConfig(configFlags.configPath)
	if err != nil {
		log.Fatal(fmt.Errorf("error initializing config [ERR: %+v]", err))
	}

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	log.Println("getting current directory")
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	srcDir := filepath.Join(dir, "migrations")
	srcPath := fmt.Sprintf("file://%s", srcDir)

	log.Println("Using migrations from: ", srcPath)

	if configFlags.version > 0 {
		log.Println("Migrating to version: ", configFlags.version)
		err = MigrateTo(srcPath, cfg.DatabaseURL, configFlags.version)
	} else {
		err = RunMigrations(srcPath, cfg.DatabaseURL)
	}

	if err != nil {
		log.Fatal(err)
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

func RunMigrations(srcPath, dbURL string) error {
	m, err := migrate.New(srcPath, dbURL)
	if err != nil {
		return err
	}

	defer m.Close()

	return m.Up()
}

func MigrateTo(srcPath, dbURL string, version uint) error {
	m, err := migrate.New(srcPath, dbURL)
	if err != nil {
		return err
	}

	defer m.Close()

	return m.Migrate(version)
}
