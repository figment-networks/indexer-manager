package structs

import "time"

type HeightRange struct {
	Epoch       string
	StartHeight int64
	EndHeight   int64
}

type HeightHash struct {
	Height int64
	Hash   string
}

// Transaction contains the blockchain transaction details
type Transaction struct {
	ID        int64     `json:"id,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`

	Hash      string `json:"hash,omitempty"`
	BlockHash string `json:"block_hash,omitempty"`

	Height uint64    `json:"height,omitempty"`
	Time   time.Time `json:"time,omitempty"`

	GasWanted uint64 `json:"gas_wanted,omitempty"`
	GasUsed   uint64 `json:"gas_used,omitempty"`
	Memo      string `json:"memo,omitempty"`
	Nonce     int    `json:"nonce,omitempty"`

	Events []TransactionEvent `json:"events,omitempty"`
}

type TransactionEvent struct {
	Type string        `json:"type,omitempty"`
	Sub  []SubsetEvent `json:"sub,omitempty"`
}

type TransactionAmount struct {
	Text string `json:"text,omitempty"`
}

type SubsetEvent struct {
	Type      string   `json:"type,omitempty"`
	Sender    []string `json:"sender,omitempty"`
	Recipient []string `json:"recipient,omitempty"`
	Validator []string `json:"validator,omitempty"`

	Action string `json:"action,omitempty"`
	Module string `json:"module,omitempty"`

	Amount *TransactionAmount `json:"amount,omitempty"`
	Fee    uint64             `json:"fee,omitempty"`
	Memo   string             `json:"memo,omitempty"`
}
