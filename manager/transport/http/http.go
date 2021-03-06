package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/figment-networks/indexer-manager/manager/client"
	shared "github.com/figment-networks/indexer-manager/structs"
)

//go:generate swagger generate spec --scan-models -o swagger.json

// ValidationError structure as formated error
type ValidationError struct {
	Msg string `json:"error"`
}

func (ve ValidationError) Error() string {
	return fmt.Sprintf("Bad Request: %s", ve.Msg)
}

// TransactionSearch - A set of fields used as params for search
// swagger:model
type TransactionSearch struct {
	// Network identifier to search in
	//
	// required: true
	// example: cosmos
	Network string `json:"network"`
	// ChainID to search in
	//
	// required: true
	// items.unique: true
	// example: ['cosmoshub-3']
	ChainIDs []string `json:"chain_ids"`
	// Epoch of transaction
	//
	// required: true
	Epoch string `json:"epoch"`
	// Height of the transactions to get
	//
	// min: 0
	Height uint64 `json:"height"`
	// AfterHeight gets all transaction bigger than given height
	// Has to be bigger than BeforeHeight
	//
	// min: 0
	AfterHeight uint64 `json:"after_height"`
	// BeforeHeight gets all transaction lower than given height
	// Has to be lesser than AfterHeight
	//
	// min: 0
	BeforeHeight uint64 `json:"before_height"`
	// Type - the list of types of transactions
	//
	// items.pattern: \w+
	// items.unique: true
	Type []string `json:"type"`
	// BlockHash - the hash of block to get transaction from
	BlockHash string `json:"block_hash"`
	// Hash - the hash of transaction
	Hash string `json:"hash"`
	// Account - the account identifier to look for
	// This searches for all accounts id which exists in transaction including senders, recipients, validators, feeders etc etc
	//
	// items.pattern: \w+
	// items.unique: true
	Account []string `json:"account"`
	// Sender looks for transactions that includes given accountIDs
	//
	// items.pattern: \w+
	// items.unique: true
	Sender []string `json:"sender"`
	// Receiver looks for transactions that includes given accountIDs
	//
	// items.pattern: \w+
	// items.unique: true
	Receiver []string `json:"receiver"`
	// Memo sets full text search for memo field
	Memo string `json:"memo"`
	// The time of transaction (if not given by chain API, the same as block)
	AfterTime time.Time `json:"after_time"`
	// The time of transaction (if not given by chain API, the same as block)
	BeforeTime time.Time `json:"before_time"`
	// Limit of how many requests to get in one request
	//
	// default: 100
	// max: 1000
	Limit uint64 `json:"limit"`
	// Offset the offset number or
	Offset uint64 `json:"offset"`

	// WithRaw - include base64 raw request in search response
	// default: false
	WithRaw bool `json:"with_raw"`

	// WithRawLog - include raw log data in search response
	// default: false
	WithRawLog bool `json:"with_raw_log"`
}

// Connector is main HTTP connector for manager
type Connector struct {
	cli client.ClientContractor
}

// NewConnector is  Connector constructor
func NewConnector(cli client.ClientContractor) *Connector {
	return &Connector{cli}
}

// InsertTransactions is http handler for InsertTransactions method
func (c *Connector) InsertTransactions(w http.ResponseWriter, req *http.Request) {
	nv := client.NetworkVersion{Network: "", Version: "0.0.1"}
	s := strings.Split(req.URL.Path, "/")

	if len(s) > 0 {
		nv.Network = s[2]
	} else {
		nv.Network = req.URL.Path
	}

	err := c.cli.InsertTransactions(req.Context(), nv, req.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetTransactions is http handler for GetTransactions method
func (c *Connector) GetTransactions(w http.ResponseWriter, req *http.Request) {

	enc := json.NewEncoder(w)
	strHeight := req.URL.Query().Get("start_height")
	intHeight, err := strconv.Atoi(strHeight)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		enc.Encode(ValidationError{Msg: "Invalid start_height param: " + err.Error()})
		return
	}

	endHeight := req.URL.Query().Get("end_height")
	intEndHeight, _ := strconv.Atoi(endHeight)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		enc.Encode(ValidationError{Msg: "Invalid end_height param: " + err.Error()})
		return
	}

	hash := req.URL.Query().Get("hash")
	network := req.URL.Query().Get("network")
	chainID := req.URL.Query().Get("chain_id")

	ctx, cancel := context.WithTimeout(req.Context(), 5*time.Minute)
	defer cancel()

	hr := shared.HeightRange{
		StartHeight: uint64(intHeight),
		EndHeight:   uint64(intEndHeight),
		Network:     network,
		ChainID:     chainID,
	}
	if hash != "" {
		hr.Hash = hash
	}
	transactions, err := c.cli.GetTransactions(ctx, client.NetworkVersion{Network: network, Version: "0.0.1", ChainID: chainID}, hr, 1000, false)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		enc.Encode(ValidationError{Msg: "Error getting transaction: " + err.Error()})
		return
	}

	log.Printf("Returning %d transactions", len(transactions))
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	enc.Encode(transactions)
}

// SearchTransactions is http handler for SearchTransactions method
func (c *Connector) SearchTransactions(w http.ResponseWriter, req *http.Request) {

	ct := req.Header.Get("Content-Type")
	enc := json.NewEncoder(w)

	ts := &TransactionSearch{}

	// (lukanus): make json the default payload
	if ct != "" && !strings.Contains(ct, "json") {
		w.WriteHeader(http.StatusNotAcceptable)
		enc.Encode(ValidationError{Msg: "Not supported content type"})
		return
	}

	dec := json.NewDecoder(req.Body)
	if err := dec.Decode(ts); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		enc.Encode(ValidationError{Msg: "Invalid json request " + err.Error()})
		return
	}
	// (lukanus): enforce 100 limit by default
	if ts.Limit == 0 {
		ts.Limit = 100
	}
	if err := validateSearchParams(ts); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		enc.Encode(ValidationError{Msg: err.Error()})
		return
	}

	ctx, cancel := context.WithTimeout(req.Context(), 1*time.Minute)
	defer cancel()

	transactions, err := c.cli.SearchTransactions(ctx, shared.TransactionSearch{
		Network:      ts.Network,
		ChainIDs:     ts.ChainIDs,
		Epoch:        ts.Epoch,
		Height:       ts.Height,
		Type:         ts.Type,
		Hash:         ts.Hash,
		BlockHash:    ts.BlockHash,
		Account:      ts.Account,
		Sender:       ts.Sender,
		Receiver:     ts.Receiver,
		Memo:         ts.Memo,
		BeforeTime:   ts.BeforeTime,
		AfterTime:    ts.AfterTime,
		Limit:        ts.Limit,
		Offset:       ts.Offset,
		AfterHeight:  ts.AfterHeight,
		BeforeHeight: ts.BeforeHeight,
		WithRaw:      ts.WithRaw,
		WithRawLog:   ts.WithRawLog,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		enc.Encode(ValidationError{Msg: err.Error()})
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	enc.Encode(transactions)
}

// ScrapeLatest is http handler for ScrapeLatest method
func (c *Connector) ScrapeLatest(w http.ResponseWriter, req *http.Request) {
	ct := req.Header.Get("Content-Type")

	ldReq := &shared.LatestDataRequest{}
	if strings.Contains(ct, "json") {
		dec := json.NewDecoder(req.Body)
		err := dec.Decode(ldReq)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
	}
	ldResp, err := c.cli.ScrapeLatest(req.Context(), *ldReq)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.Encode(ldResp)
}

// MissingTransactionsResponse the response
type MissingTransactionsResponse struct {
	MissingTransactions [][2]uint64 `json:"missing_transactions"`
	MissingBlocks       [][2]uint64 `json:"missing_blocks"`
}

// CheckMissingTransactions is http handler for CheckMissingTransactions method
func (c *Connector) CheckMissingTransactions(w http.ResponseWriter, req *http.Request) {
	strHeight := req.URL.Query().Get("start_height")
	intHeight, _ := strconv.ParseUint(strHeight, 10, 64)

	endHeight := req.URL.Query().Get("end_height")
	intEndHeight, _ := strconv.ParseUint(endHeight, 10, 64)

	w.Header().Add("Content-Type", "application/json")

	if intHeight == 0 || intEndHeight == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"start_height and end_height query params are not set properly"}`))
		return
	}

	network := req.URL.Query().Get("network")
	chainID := req.URL.Query().Get("chain_id")

	if network == "" || chainID == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"network and chain_id parameters are required"}`))
		return
	}

	var err error
	mtr := MissingTransactionsResponse{}
	mtr.MissingBlocks, mtr.MissingTransactions, err = c.cli.CheckMissingTransactions(req.Context(), client.NetworkVersion{Network: network, Version: "0.0.1", ChainID: chainID}, shared.HeightRange{StartHeight: intHeight, EndHeight: intEndHeight}, client.MissingDiffTypeSQLHybrid, 999)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.Encode(mtr)
}

// GetRunningTransactions gets currently running transactions
func (c *Connector) GetRunningTransactions(w http.ResponseWriter, req *http.Request) {
	run, err := c.cli.GetRunningTransactions(req.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	enc := json.NewEncoder(w)

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	enc.Encode(run)
}

// StopRunningTransactions stops currently running transactions
func (c *Connector) StopRunningTransactions(w http.ResponseWriter, req *http.Request) {
	clean := (req.URL.Query().Get("clean") != "")
	network := req.URL.Query().Get("network")
	chainID := req.URL.Query().Get("chain_id")

	if network == "" || chainID == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"network and chain_id parameters are required"}`))
		return
	}

	if err := c.cli.StopRunningTransactions(req.Context(), client.NetworkVersion{Network: network, Version: "0.0.1", ChainID: chainID}, clean); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"state":"success"}`))
}

// GetMissingTransactions is http handler for GetMissingTransactions method
func (c *Connector) GetMissingTransactions(w http.ResponseWriter, req *http.Request) {

	strHeight := req.URL.Query().Get("start_height")
	intHeight, _ := strconv.ParseUint(strHeight, 10, 64)

	endHeight := req.URL.Query().Get("end_height")
	intEndHeight, _ := strconv.ParseUint(endHeight, 10, 64)

	async := req.URL.Query().Get("async")

	force := (req.URL.Query().Get("force") != "")

	overwriteAll := (req.URL.Query().Get("overwrite_all") != "")
	simplified := (req.URL.Query().Get("simplified") != "")

	network := req.URL.Query().Get("network")
	chainID := req.URL.Query().Get("chain_id")

	if network == "" || chainID == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"network and chain_id parameters are required"}`))
		return
	}

	if intHeight == 0 || intEndHeight == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("start_height and end_height query params are not set properly"))
		return
	}

	if async == "" {
		_, err := c.cli.GetMissingTransactions(req.Context(), client.NetworkVersion{Network: network, Version: "0.0.1", ChainID: chainID},
			shared.HeightRange{
				StartHeight: intHeight,
				EndHeight:   intEndHeight,
				Network:     network,
				ChainID:     chainID},
			client.GetMissingTxParams{
				Window:       1000,
				Async:        false,
				Force:        force,
				OverwriteAll: overwriteAll,
				Simplified:   simplified,
			})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	run, err := c.cli.GetMissingTransactions(req.Context(), client.NetworkVersion{Network: network, Version: "0.0.1", ChainID: chainID},
		shared.HeightRange{
			StartHeight: intHeight,
			EndHeight:   intEndHeight,
			Network:     network,
			ChainID:     chainID},
		client.GetMissingTxParams{
			Window:       1000,
			Async:        true,
			Force:        force,
			OverwriteAll: overwriteAll,
			Simplified:   simplified,
		})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	enc := json.NewEncoder(w)
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if simplified {
		enc.Encode(client.Run{ // remove whole progress
			NV:               run.NV,
			HeightRange:      run.HeightRange,
			ProgressSummary:  run.ProgressSummary,
			Success:          run.Success,
			Finished:         run.Finished,
			StartedTime:      run.StartedTime,
			FinishTime:       run.FinishTime,
			LastProgressTime: run.LastProgressTime,
		})
		return
	}

	enc.Encode(run)

}

// GetRewards calculates daily rewards for provided address
func (c *Connector) GetRewards(w http.ResponseWriter, req *http.Request) {
	network := req.URL.Query().Get("network")
	chainID := req.URL.Query().Get("chain_id")
	if network == "" || chainID == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"network and chain_id parameters are required"}`))
		return
	}

	account := req.URL.Query().Get("account")
	if account == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"account parameter is required"}`))
		return
	}

	start := req.URL.Query().Get("start_time")
	end := req.URL.Query().Get("end_time")
	if start == "" || end == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"start_time and end_time parameters are required"}`))
		return
	}
	startTime, err := time.Parse(time.RFC3339Nano, start)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"start_time parameter should be in the format \"2006-01-02T15:04:05.999999999Z07:00\""}`))
		return
	}

	endTime, err := time.Parse(time.RFC3339Nano, end)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"end_time parameter should be in the format \"2006-01-02T15:04:05.999999999Z07:00\""}`))
		return
	}

	resp, err := c.cli.GetRewards(req.Context(), client.NetworkVersion{Network: network, Version: "0.0.1", ChainID: chainID}, startTime, endTime, account)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	enc := json.NewEncoder(w)
	enc.Encode(resp)

	w.Header().Add("Content-Type", "application/json")
}

// GetAccountBalance calculates daily balance for provided address
func (c *Connector) GetAccountBalance(w http.ResponseWriter, req *http.Request) {
	network := req.URL.Query().Get("network")
	chainID := req.URL.Query().Get("chain_id")
	if network == "" || chainID == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"network and chain_id parameters are required"}`))
		return
	}

	account := req.URL.Query().Get("account")
	if account == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"account parameter is required"}`))
		return
	}

	start := req.URL.Query().Get("start_time")
	end := req.URL.Query().Get("end_time")
	if start == "" || end == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"start_time and end_time parameters are required"}`))
		return
	}
	startTime, errStart := time.Parse(time.RFC3339Nano, start)
	endTime, errEnd := time.Parse(time.RFC3339Nano, end)
	if errStart != nil || errEnd != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"time range parameters should be in the format \"2006-01-02T15:04:05.999999999Z07:00\""}`))
		return
	}

	resp, err := c.cli.GetAccountBalance(req.Context(), client.NetworkVersion{Network: network, Version: "0.0.1", ChainID: chainID}, startTime, endTime, account)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	enc := json.NewEncoder(w)
	enc.Encode(resp)

	w.Header().Add("Content-Type", "application/json")
}

// AttachToHandler attaches handlers to http server's mux
func (c *Connector) AttachToHandler(mux *http.ServeMux) {
	mux.HandleFunc("/transactions_search", c.SearchTransactions)

	mux.HandleFunc("/transactions", c.GetTransactions)
	mux.HandleFunc("/transactions_insert/", c.InsertTransactions)
	mux.HandleFunc("/scrape_latest", c.ScrapeLatest)

	mux.HandleFunc("/check_missing", c.CheckMissingTransactions)
	mux.HandleFunc("/get_missing", c.GetMissingTransactions)
	mux.HandleFunc("/get_running", c.GetRunningTransactions)
	mux.HandleFunc("/stop_running", c.StopRunningTransactions)

	mux.HandleFunc("/rewards", c.GetRewards)
	mux.HandleFunc("/account/balance", c.GetAccountBalance)
}

func validateSearchParams(ts *TransactionSearch) error {

	if ts.Network == "" {
		return ValidationError{Msg: "network parameter is mandatory"}
	}

	if ts.Height > 0 &&
		(ts.AfterHeight > 0 || ts.BeforeHeight > 0) {
		return ValidationError{Msg: `When height parameter is set following parameters has to be empty ["after_height","before_height"]`}
	}

	if ts.AfterHeight > 0 && ts.AfterHeight >= ts.BeforeHeight {
		return ValidationError{Msg: "before_height has to be bigger than after_height"}
	}

	for _, ty := range ts.Type {
		if ty == "" {
			return ValidationError{Msg: "given type cannot be an empty string"}
		}
	}

	for _, ac := range ts.Account {
		if ac == "" {
			return ValidationError{Msg: "given account cannot be an empty string"}
		}
	}

	for _, ac := range ts.Sender {
		if ac == "" {
			return ValidationError{Msg: "given sender cannot be an empty string"}
		}
	}

	for _, ac := range ts.Receiver {
		if ac == "" {
			return ValidationError{Msg: "given receiver cannot be an empty string"}
		}
	}

	if !ts.AfterTime.IsZero() && !ts.AfterTime.Before(ts.BeforeTime) {
		return ValidationError{Msg: "after_time has to be after than before_time "}
	}

	if ts.Limit > 1000 {
		return ValidationError{Msg: "limit has to be smaller than 1000"}

	}

	return nil
}
