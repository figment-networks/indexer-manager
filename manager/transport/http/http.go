package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/figment-networks/cosmos-indexer/manager/client"
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

func (hc *HubbleConnector) GetTransactions(w http.ResponseWriter, req *http.Request) {

	strHeight := req.URL.Query().Get("height")
	intHeight, _ := strconv.Atoi(strHeight)
	nv := client.NetworkVersion{"cosmos", "0.0.1"}
	transactions, err := hc.cli.GetTransactions(req.Context(), nv, intHeight)

	if err != nil {
		w.Write([]byte(err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	enc := json.NewEncoder(w)
	enc.Encode(transactions)

	w.WriteHeader(http.StatusOK)
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
	mux.HandleFunc("/transactions/:id", hc.GetTransaction)
	mux.HandleFunc("/accounts", hc.GetAccounts)
	mux.HandleFunc("/accounts/:id", hc.GetAccount)
}
