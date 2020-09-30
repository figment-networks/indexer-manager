package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"time"
)

type Readiness struct {
	DB ReadinessDB `json:"db"`
}

type ReadinessDB struct {
	ReadinessPostgres `json:"postgres"`
}

type ReadinessPostgres struct {
	Status   string `json:"status"`
	Duration string `json:"duration"`
	Error    string `json:"error"`
}

func readinessPostgresCheck(ctx context.Context, db *sql.DB) ReadinessPostgres {
	tCtx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()

	t := time.Now()
	err := db.PingContext(tCtx)
	rp := ReadinessPostgres{Status: "ok", Duration: time.Since(t).String()}
	if err != nil {
		rp.Status = "err"
		rp.Error = err.Error()
	}
	return rp
}

// attachHealthCheck basic healthcheck with basic simple readiness endpoint
func attachHealthCheck(ctx context.Context, mux *http.ServeMux, db *sql.DB) {
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/readiness", func(w http.ResponseWriter, r *http.Request) {
		enc := json.NewEncoder(w)

		rp := readinessPostgresCheck(ctx, db)
		if rp.Error != "" {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
		enc.Encode(Readiness{DB: ReadinessDB{ReadinessPostgres: rp}})
	})
}
