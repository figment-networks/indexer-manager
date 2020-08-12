package structs

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

type HeightRange struct {
	Epoch       string
	StartHeight int64
	EndHeight   int64
}

type HeightHash struct {
	Height int64
	Hash   string
}

type TransactionExtra struct {
	Network     string      `json:"network,omitempty"`
	ChainID     string      `json:"chain_id,omitempty"`
	Transaction Transaction `json:"transaction,omitempty"`
}

// Transaction contains the blockchain transaction details
type Transaction struct {
	ID        uuid.UUID  `json:"id,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`

	Hash      string `json:"hash,omitempty"`
	BlockHash string `json:"block_hash,omitempty"`

	Height uint64    `json:"height,omitempty"`
	Time   time.Time `json:"time,omitempty"`

	Fee       []TransactionFee `json:"transaction_fee,omitempty"`
	GasWanted uint64           `json:"gas_wanted,omitempty"`
	GasUsed   uint64           `json:"gas_used,omitempty"`

	Memo  string `json:"memo,omitempty"`
	Nonce int    `json:"nonce,omitempty"`

	Version string            `json:"version"`
	Events  TransactionEvents `json:"events,omitempty"`
}

type TransactionEvent struct {
	ID   string        `json:"id,omitempty"`
	Kind string        `json:"kind,omitempty"`
	Sub  []SubsetEvent `json:"sub,omitempty"`
}

type TransactionEvents []TransactionEvent

func (te *TransactionEvents) Scan(value interface{}) error {

	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &te)
}

type TransactionFee struct {
	Text     string  `json:"text,omitempty"`
	Numeric  float64 `json:"numeric,omitempty"`
	Currency string  `json:"currency,omitempty"`
}

type TransactionAmount struct {
	Text     string  `json:"text,omitempty"`
	Numeric  float64 `json:"numeric,omitempty"`
	Currency string  `json:"currency,omitempty"`
}

type SubsetEvent struct {
	Type   string `json:"type,omitempty"`
	Action string `json:"action,omitempty"`
	Module string `json:"module,omitempty"`

	Sender    []string `json:"sender,omitempty"`
	Recipient []string `json:"recipient,omitempty"`
	Validator []string `json:"validator,omitempty"`
	Feeder    []string `json:"feeder,omitempty"`

	Amount *TransactionAmount `json:"amount,omitempty"`
}

// Block
type Block struct {
	ID        int64      `json:"id,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`

	Hash   string    `json:"hash,omitempty"`
	Height uint64    `json:"height,omitempty"`
	Time   time.Time `json:"time,omitempty"`
}

type TransactionSearch struct {
	//	AfterID   uint     `form:"after_id"`
	//	BeforeID  uint     `form:"before_id"`
	Height    uint64    `json:"height"`
	Type      []string  `json:"type"`
	BlockHash string    `json:"block_hash"`
	Account   string    `json:"account"`
	Sender    string    `json:"sender"`
	Receiver  string    `json:"receiver"`
	Memo      string    `json:"memo"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Limit     uint64    `json:"limit"`

	Network string `json:"network"`
}
