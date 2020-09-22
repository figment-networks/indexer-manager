package http

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/figment-networks/cosmos-indexer/manager/client"
	shared "github.com/figment-networks/cosmos-indexer/structs"
)

// TransactionSearch - A set of fields used as params for search
type TransactionSearch struct {
	Network string `json:"network"`
	//	AfterID   uint     `form:"after_id"`
	//	BeforeID  uint     `form:"before_id"`
	Height    uint64   `json:"height"`
	Type      []string `json:"type"`
	BlockHash string   `json:"block_hash"`
	Account   string   `json:"account"`
	Sender    string   `json:"sender"`
	Receiver  string   `json:"receiver"`
	Memo      string   `json:"memo"`
	StartTime string   `json:"start_time"`
	EndTime   string   `json:"end_time"`
	Limit     uint     `json:"limit"`
}

// Connector is main HTTP connector for manager
type Connector struct {
	cli client.ClientContractor
}

// NewConnector is  Connector constructor
func NewConnector(cli client.ClientContractor) *Connector {
	return &Connector{cli}
}

// GetTransaction is http handler for GetTransaction method
func (hc *Connector) GetTransaction(w http.ResponseWriter, req *http.Request) {
	nv := client.NetworkVersion{Network: "cosmos", Version: "0.0.1"}
	hc.cli.GetTransaction(req.Context(), nv, "")
	w.WriteHeader(http.StatusNotImplemented)
}

// InsertTransactions is http handler for InsertTransactions method
func (hc *Connector) InsertTransactions(w http.ResponseWriter, req *http.Request) {
	nv := client.NetworkVersion{Network: "cosmos", Version: "0.0.1"}
	s := strings.Split(req.URL.Path, "/")

	if len(s) > 0 {
		nv.Network = s[2]
	} else {
		nv.Network = req.URL.Path
	}

	err := hc.cli.InsertTransactions(req.Context(), nv, req.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetTransactions is http handler for GetTransactions method
func (hc *Connector) GetTransactions(w http.ResponseWriter, req *http.Request) {
	strHeight := req.URL.Query().Get("height")
	intHeight, _ := strconv.Atoi(strHeight)

	endHeight := req.URL.Query().Get("end_height")
	intEndHeight, _ := strconv.Atoi(endHeight)

	hash := req.URL.Query().Get("hash")

	network := req.URL.Query().Get("network")
	if network == "" {
		network = "cosmos"
	}

	nv := client.NetworkVersion{Network: network, Version: "0.0.1"}

	ctx, cancel := context.WithTimeout(req.Context(), 5*time.Minute)
	defer cancel()

	hr := shared.HeightRange{
		Epoch:       "",
		StartHeight: uint64(intHeight),
		EndHeight:   uint64(intEndHeight),
	}
	if hash != "" {
		hr.Hash = hash
	}
	transactions, err := hc.cli.GetTransactions(ctx, nv, hr, 1000, false)

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	log.Printf("Returning %d transactions", len(transactions))
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.Encode(transactions)
}

// SearchTransactions is http handler for SearchTransactions method
func (hc *Connector) SearchTransactions(w http.ResponseWriter, req *http.Request) {
	ct := req.Header.Get("Content-Type")

	network := ""
	ts := &shared.TransactionSearch{}
	if strings.Contains(ct, "json") {
		dec := json.NewDecoder(req.Body)
		err := dec.Decode(ts)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
		if ts.Network != "" {
			network = ts.Network
		}

	} else if strings.Contains(ct, "form") {
		err := req.ParseForm()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}
		ts.Height, _ = strconv.ParseUint(req.Form.Get("height"), 10, 64)
		ts.Account = req.Form.Get("account")
		ts.Sender = req.Form.Get("sender")
		ts.Receiver = req.Form.Get("receiver")
		ts.BlockHash = req.Form.Get("block_hash")
		ts.Memo = req.Form.Get("memo")
		ts.Sender = req.Form.Get("sender")
		ts.Receiver = req.Form.Get("receiver")

		ts.AfterHeight, _ = strconv.ParseUint(req.Form.Get("after_height"), 10, 64)
		ts.BeforeHeight, _ = strconv.ParseUint(req.Form.Get("before_height"), 10, 64)
		ts.Limit, _ = strconv.ParseUint(req.Form.Get("limit"), 10, 64)

		network = req.Form.Get("network")
	}

	if network == "" {
		network = "cosmos"
	}

	if ts.ChainID == "" {

	}

	ctx, cancel := context.WithTimeout(req.Context(), 1*time.Minute)
	defer cancel()

	transactions, err := hc.cli.SearchTransactions(ctx, client.NetworkVersion{Network: network, ChainID: ts.ChainID, Version: "0.0.1"}, *ts)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.Encode(transactions)
}

// ScrapeLatest is http handler for ScrapeLatest method
func (hc *Connector) ScrapeLatest(w http.ResponseWriter, req *http.Request) {
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
	ldResp, err := hc.cli.ScrapeLatest(req.Context(), *ldReq)
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
func (hc *Connector) CheckMissingTransactions(w http.ResponseWriter, req *http.Request) {
	strHeight := req.URL.Query().Get("start_height")
	intHeight, _ := strconv.ParseUint(strHeight, 10, 64)

	endHeight := req.URL.Query().Get("end_height")
	intEndHeight, _ := strconv.ParseUint(endHeight, 10, 64)

	if intHeight == 0 || intEndHeight == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("start_height and end_height query params are not set properly"))
		return
	}

	var err error
	mtr := MissingTransactionsResponse{}
	nv := client.NetworkVersion{Network: "cosmos", Version: "0.0.1", ChainID: "cosmoshub-3"}
	mtr.MissingBlocks, mtr.MissingTransactions, err = hc.cli.CheckMissingTransactions(req.Context(), nv, shared.HeightRange{StartHeight: intHeight, EndHeight: intEndHeight}, 1000)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.Encode(mtr)
}

// GetRunningTransactions gets currently running transactions
func (hc *Connector) GetRunningTransactions(w http.ResponseWriter, req *http.Request) {
	run, err := hc.cli.GetRunningTransactions(req.Context())
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

// GetMissingTransactions is http handler for GetMissingTransactions method
func (hc *Connector) GetMissingTransactions(w http.ResponseWriter, req *http.Request) {
	nv := client.NetworkVersion{Network: "cosmos", Version: "0.0.1", ChainID: "cosmoshub-3"}

	strHeight := req.URL.Query().Get("start_height")
	intHeight, _ := strconv.ParseUint(strHeight, 10, 64)

	endHeight := req.URL.Query().Get("end_height")
	intEndHeight, _ := strconv.ParseUint(endHeight, 10, 64)

	async := req.URL.Query().Get("async")

	force := (req.URL.Query().Get("force") != "")

	if intHeight == 0 || intEndHeight == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("start_height and end_height query params are not set properly"))
		return
	}

	if async == "" {
		_, err := hc.cli.GetMissingTransactions(req.Context(), nv,
			shared.HeightRange{StartHeight: intHeight, EndHeight: intEndHeight}, 1000, false, force)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}

	run, err := hc.cli.GetMissingTransactions(req.Context(), nv,
		shared.HeightRange{StartHeight: intHeight, EndHeight: intEndHeight}, 1000, true, force)
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

// AttachToHandler attaches handlers to http server's mux
func (c *Connector) AttachToHandler(mux *http.ServeMux) {
	mux.HandleFunc("/transactions", c.GetTransactions)
	mux.HandleFunc("/transactions/", c.GetTransaction)
	mux.HandleFunc("/transactions_search", c.SearchTransactions)

	mux.HandleFunc("/transactions_insert/", c.InsertTransactions)
	mux.HandleFunc("/scrape_latest", c.ScrapeLatest)

	mux.HandleFunc("/check_missing", c.CheckMissingTransactions)
	mux.HandleFunc("/get_missing", c.GetMissingTransactions)
	mux.HandleFunc("/get_running", c.GetRunningTransactions)

}
