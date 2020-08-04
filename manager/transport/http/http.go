package http

import (
	"net/http"

	"github.com/figment-networks/cosmos-indexer/manager/client"
)

type HubbleConnector struct {
	cli *client.HubbleClient
}

func NewHubbleConnector(cli *client.HubbleClient) *HubbleConnector {
	return &HubbleConnector{cli}
}

func (hc *HubbleConnector) GetCurrentHeight(w http.ResponseWriter, req *http.Request) {

	// (lukanus): Current == no params :)
	hc.cli.GetCurrentHeight(req.Context())

	w.WriteHeader(http.StatusNotImplemented)
}

func (hc *HubbleConnector) GetCurrentBlock(w http.ResponseWriter, req *http.Request) {

	// (lukanus): Current == no params :)
	hc.cli.GetCurrentBlock(req.Context())
	w.WriteHeader(http.StatusNotImplemented)
}

func (hc *HubbleConnector) GetBlock(w http.ResponseWriter, req *http.Request) {

	keys := req.URL.Query()
	id := keys.Get("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	hc.cli.GetBlock(req.Context(), id)

	w.WriteHeader(http.StatusNotImplemented)
}

func (hc *HubbleConnector) GetBlocks(w http.ResponseWriter, req *http.Request) {
	hc.cli.GetBlocks(req.Context())
	w.WriteHeader(http.StatusNotImplemented)
}

func (hc *HubbleConnector) GetBlockTimes(w http.ResponseWriter, req *http.Request) {
	hc.cli.GetBlockTimes(req.Context())
	w.WriteHeader(http.StatusNotImplemented)
}

func (hc *HubbleConnector) GetBlockTimesInterval(w http.ResponseWriter, req *http.Request) {
	hc.cli.GetBlockTimesInterval(req.Context())
	w.WriteHeader(http.StatusNotImplemented)
}

func (hc *HubbleConnector) GetTransaction(w http.ResponseWriter, req *http.Request) {
	hc.cli.GetTransaction(req.Context())
	w.WriteHeader(http.StatusNotImplemented)
}

func (hc *HubbleConnector) GetTransactions(w http.ResponseWriter, req *http.Request) {
	hc.cli.GetTransactions(req.Context())
	w.WriteHeader(http.StatusNotImplemented)
}

func (hc *HubbleConnector) GetAccounts(w http.ResponseWriter, req *http.Request) {
	hc.cli.GetAccounts(req.Context())
	w.WriteHeader(http.StatusNotImplemented)
}

func (hc *HubbleConnector) GetAccount(w http.ResponseWriter, req *http.Request) {
	hc.cli.GetAccount(req.Context())
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
