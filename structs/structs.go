package structs

import "time"

type HeightRange struct {
	Epoch       string
	StartHeight int64
	EndHeight   int64
}

// Transaction contains the blockchain transaction details
type Transaction struct {
	ID        int64     `json:"id,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`

	Hash      string `json:"hash,omitempty"`
	BlockHash string `json:"block_hash,omitempty"`

	Type   string    `json:"type,omitempty"`
	Height uint64    `json:"height,omitempty"`
	Time   time.Time `json:"time,omitempty"`

	GasWanted uint64 `json:"gas_wanted,omitempty"`
	GasUsed   uint64 `json:"gas_used,omitempty"`
	Memo      string `json:"memo,omitempty"`

	Sender   string `json:"sender,omitempty"`
	Receiver string `json:"receiver,omitempty"`
	Amount   uint64 `json:"amount,omitempty"`
	Fee      uint64 `json:"fee,omitempty"`
	Nonce    int    `json:"nonce,omitempty"`
}
