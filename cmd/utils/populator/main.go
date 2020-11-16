package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"plugin"

	"github.com/figment-networks/indexer-manager/cmd/common/logger"
	"github.com/figment-networks/indexer-manager/manager/store/params"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

var (
	network    string
	chain      string
	pluginPath string
	dbpath     string
	from       uint64
	to         uint64
)

func init() {
	flag.StringVar(&network, "network", "", "Network  name")
	flag.StringVar(&chain, "chain", "", "Chain  name")
	flag.StringVar(&pluginPath, "plugin", "", "Path to plugin file")
	flag.StringVar(&dbpath, "db", "", "Database connection string")
	flag.Uint64Var(&from, "from", 0, "Starting height")
	flag.Uint64Var(&to, "to", 0, "End height")
	flag.Parse()
}

func main() {
	ctx := context.Background()
	p, err := plugin.Open(pluginPath)
	if err != nil {
		log.Fatal(err)
	}
	decodeFee, err := p.Lookup("DecodeFee")
	if err != nil {
		log.Fatal(err)
	}
	logger.Init("json", "info", []string{"stderr"}, nil)
	zlog := logger.GetLogger()
	defer zlog.Sync()
	// connect to database
	logger.Info("[DB] Connecting to database...")
	db, err := sql.Open("postgres", dbpath)
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
	if err := getTransactions(ctx, zlog, db, decodeFee, network, chain, from, to); err != nil {
		logger.Error(err)
		return
	}

}

func getTransactions(ctx context.Context, zLog *zap.Logger, db *sql.DB, decodeFee plugin.Symbol, network, chain string, from, to uint64) error {

	DecodeFee := decodeFee.(func(logger *zap.Logger, reader io.Reader) []map[string]interface{})
	rows, err := db.QueryContext(ctx, "SELECT id, height, raw FROM public.transaction_events WHERE network = $1 AND chain_id = $2 AND height => $3 AND height <= $4 ORDER BY height ASC LIMIT 1000", network, chain, from, to)
	switch {
	case err == sql.ErrNoRows:
		return params.ErrNotFound
	case err != nil:
		return fmt.Errorf("query error: %w", err)
	default:
	}

	defer rows.Close()

	var id uuid.UUID
	var raw []byte
	var height uint64

	var iteration uint64
	for rows.Next() {
		if err := rows.Scan(&id, &height, &raw); err != nil {
			return err
		}
		structs := DecodeFee(zLog, bytes.NewReader(raw))
		if len(structs) > 0 {
			stxt, err := json.Marshal(structs)
			if err != nil {
				return err
			}

			// Updates fee
			_, err = db.ExecContext(ctx, "UPDATE public.transaction_events SET fee = $1 WHERE id = $2", stxt, id)
			if err != nil {
				return err
			}
		}
		if iteration%1000 == 0 {
			log.Printf("%+v", height)
		}

		iteration++
	}

	return nil
}
