// +build !scheduler

package main

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/figment-networks/indexer-manager/cmd/manager/config"
	"github.com/figment-networks/indexer-manager/manager/client"
	"go.uber.org/zap"
)

func attachScheduler(ctx context.Context, db *sql.DB, mux *http.ServeMux, cfg config.Config, logger *zap.Logger, client client.SchedulerContractor) error {
	// noop
	return nil
}
