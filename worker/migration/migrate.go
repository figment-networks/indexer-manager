package migration

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func Run(dbURL string) error {
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
