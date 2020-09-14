package structs

import (
	"time"
)

const (
	ReqIDGetTransactions = "GetTransactions"
	ReqIDLatestData      = "GetLatest"
)

type LatestDataRequest struct {
	Network string `json:"network"`
	ChainID string `json:"chain_id"`
	Version string `json:"version"`

	LastHash   string    `json:"lastHash"`
	LastEpoch  string    `json:"lastEpoch"`
	LastHeight uint64    `json:"lastHeight"`
	LastTime   time.Time `json:"lastTime"`
	Nonce      []byte    `json:"nonce"`

	SelfCheck bool `json:"selfCheck"`
}

type LatestDataResponse struct {
	LastHash   string    `json:"lastHash"`
	LastHeight uint64    `json:"lastHeight"`
	LastTime   time.Time `json:"lastTime"`
	LastEpoch  string    `json:"lastEpoch"`
	Nonce      []byte    `json:"nonce"`
}
