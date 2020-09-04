package params

import (
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("record not found")
)

type TransactionSearch struct {
	Network   string    `json:"network"`
	Height    uint64    `json:"height"`
	Type      []string  `json:"type"`
	BlockHash string    `json:"block_hash"`
	Account   string    `json:"account"`
	Sender    string    `json:"sender"`
	Receiver  string    `json:"receiver"`
	Memo      string    `json:"memo"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`

	AfterHeight  uint64 `form:"after_height"`
	BeforeHeight uint64 `form:"before_height"`
	Limit        uint64 `json:"limit"`
	Offset       uint64 `json:"offset"`
}
