package params

import (
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("record not found")
)

type TransactionSearch struct {
	Network   string   `json:"network"`
	ChainID   string   `json:"chain_id"`
	Height    uint64   `json:"height"`
	Type      []string `json:"type"`
	BlockHash string   `json:"block_hash"`
	Account   string   `json:"account"`
	Sender    string   `json:"sender"`
	Receiver  string   `json:"receiver"`
	Memo      string   `json:"memo"`

	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`

	AfterHeight  uint64 `json:"after_height"`
	BeforeHeight uint64 `json:"before_height"`
	Limit        uint64 `json:"limit"`
	Offset       uint64 `json:"offset"`
}
