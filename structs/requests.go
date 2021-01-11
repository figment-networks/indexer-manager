package structs

import (
	"time"
)

const (
	ReqIDGetTransactions = "GetTransactions"
	ReqIDLatestData      = "GetLatest"
	ReqIDGetReward       = "GetReward"
)

type HeightRange struct {
	Epoch       string
	Hash        string
	StartHeight uint64
	EndHeight   uint64

	ChainID string
	Network string
}

type HeightHash struct {
	Epoch  string
	Height uint64
	Hash   string

	ChainID string
	Network string
}

type HeightAccount struct {
	Epoch   string
	Height  uint64
	Account string

	ChainID string
	Network string
}

type TransactionWithMeta struct {
	Network     string      `json:"network,omitempty"`
	Version     string      `json:"version,omitempty"`
	ChainID     string      `json:"chain_id,omitempty"`
	Transaction Transaction `json:"transaction,omitempty"`
}

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

type GetRewardResponse struct {
	Height  uint64              `json:"height"`
	Rewards []TransactionAmount `json:"rewards"`
}

type RewardSummary struct {
	Start  uint64              `json:"start"`
	End    uint64              `json:"end"`
	Time   time.Time           `json:"time"`
	Amount []TransactionAmount `json:"amount"`
}
