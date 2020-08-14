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

type HubbleConnector struct {
	cli *client.HubbleClient
}

func NewHubbleConnector(cli *client.HubbleClient) *HubbleConnector {
	return &HubbleConnector{cli}
}

func (hc *HubbleConnector) GetCurrentHeight(w http.ResponseWriter, req *http.Request) {

	nv := client.NetworkVersion{"cosmos", "0.0.1"}

	// (lukanus): Current == no params :)
	hc.cli.GetCurrentHeight(req.Context(), nv)

	w.WriteHeader(http.StatusNotImplemented)
}

func (hc *HubbleConnector) GetCurrentBlock(w http.ResponseWriter, req *http.Request) {

	nv := client.NetworkVersion{"cosmos", "0.0.1"}

	// (lukanus): Current == no params :)
	hc.cli.GetCurrentBlock(req.Context(), nv)
	w.WriteHeader(http.StatusNotImplemented)
}

func (hc *HubbleConnector) GetBlock(w http.ResponseWriter, req *http.Request) {
	nv := client.NetworkVersion{"cosmos", "0.0.1"}

	keys := req.URL.Query()
	id := keys.Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	hc.cli.GetBlock(req.Context(), nv, id)

	w.WriteHeader(http.StatusNotImplemented)
}

func (hc *HubbleConnector) GetBlocks(w http.ResponseWriter, req *http.Request) {
	nv := client.NetworkVersion{"cosmos", "0.0.1"}
	hc.cli.GetBlocks(req.Context(), nv)
	w.WriteHeader(http.StatusNotImplemented)
}

func (hc *HubbleConnector) GetBlockTimes(w http.ResponseWriter, req *http.Request) {
	nv := client.NetworkVersion{"cosmos", "0.0.1"}
	hc.cli.GetBlockTimes(req.Context(), nv)
	w.WriteHeader(http.StatusNotImplemented)
}

func (hc *HubbleConnector) GetBlockTimesInterval(w http.ResponseWriter, req *http.Request) {
	nv := client.NetworkVersion{"cosmos", "0.0.1"}
	hc.cli.GetBlockTimesInterval(req.Context(), nv)
	w.WriteHeader(http.StatusNotImplemented)
}

func (hc *HubbleConnector) GetTransaction(w http.ResponseWriter, req *http.Request) {
	nv := client.NetworkVersion{"cosmos", "0.0.1"}
	hc.cli.GetTransaction(req.Context(), nv, "")
	w.WriteHeader(http.StatusNotImplemented)
}

func (hc *HubbleConnector) InsertTransactions(w http.ResponseWriter, req *http.Request) {

	s := strings.Split(req.URL.Path, "/")
	nv := client.NetworkVersion{"cosmos", "0.0.1"}

	if len(s) > 0 {
		nv.Network = s[2]
	} else {
		nv.Network = req.URL.Path
	}

	log.Println("nv", nv)
	err := hc.cli.InsertTransactions(req.Context(), nv, req.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (hc *HubbleConnector) GetTransactions(w http.ResponseWriter, req *http.Request) {

	strHeight := req.URL.Query().Get("height")
	intHeight, _ := strconv.Atoi(strHeight)

	endHeight := req.URL.Query().Get("end_height")
	intEndHeight, _ := strconv.Atoi(endHeight)

	hash := req.URL.Query().Get("hash")

	network := req.URL.Query().Get("network")
	if network == "" {
		network = "cosmos"
	}

	nv := client.NetworkVersion{network, "0.0.1"}

	ctx, cancel := context.WithTimeout(req.Context(), 1*time.Minute)
	defer cancel()

	hr := shared.HeightRange{
		Epoch:       "",
		StartHeight: int64(intHeight),
		EndHeight:   int64(intEndHeight),
	}
	if hash != "" {
		hr.Hash = hash
	}
	transactions, err := hc.cli.GetTransactions(ctx, nv, hr)

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

func (hc *HubbleConnector) SearchTransactions(w http.ResponseWriter, req *http.Request) {

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
		ts.Limit, _ = strconv.ParseUint(req.Form.Get("limit"), 10, 64)
		ts.Account = req.Form.Get("account")
		ts.Sender = req.Form.Get("sender")
		ts.Receiver = req.Form.Get("receiver")
		ts.BlockHash = req.Form.Get("block_hash")
		ts.Memo = req.Form.Get("memo")
		ts.Sender = req.Form.Get("sender")
		ts.Receiver = req.Form.Get("receiver")

		network = req.Form.Get("network")

	}

	if network == "" {
		network = "cosmos"
	}

	ctx, cancel := context.WithTimeout(req.Context(), 1*time.Minute)
	defer cancel()

	transactions, err := hc.cli.SearchTransactions(ctx, client.NetworkVersion{network, "0.0.1"}, *ts)
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

func (hc *HubbleConnector) GetAccounts(w http.ResponseWriter, req *http.Request) {
	nv := client.NetworkVersion{"cosmos", "0.0.1"}
	hc.cli.GetAccounts(req.Context(), nv)
	w.WriteHeader(http.StatusNotImplemented)
}

func (hc *HubbleConnector) GetAccount(w http.ResponseWriter, req *http.Request) {
	nv := client.NetworkVersion{"cosmos", "0.0.1"}
	hc.cli.GetAccount(req.Context(), nv)
	w.WriteHeader(http.StatusNotImplemented)
}

func (hc *HubbleConnector) AttachToHandler(mux *http.ServeMux) {
	mux.HandleFunc("/height", hc.GetCurrentHeight)
	mux.HandleFunc("/block", hc.GetCurrentBlock)
	mux.HandleFunc("/blocks", hc.GetBlocks)
	mux.HandleFunc("/blocks/:id", hc.GetBlock)
	mux.HandleFunc("/block_times", hc.GetBlockTimes)
	mux.HandleFunc("/block_times_interval", hc.GetBlockTimesInterval)
	mux.HandleFunc("/transactions", hc.GetTransactions)
	mux.HandleFunc("/transactions/", hc.GetTransaction)
	mux.HandleFunc("/transactions_search", hc.SearchTransactions)
	mux.HandleFunc("/accounts", hc.GetAccounts)
	mux.HandleFunc("/accounts/", hc.GetAccount)

	mux.HandleFunc("/transactions_insert/", hc.InsertTransactions)
}
