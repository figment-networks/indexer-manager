package structs

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

type HeightRange struct {
	Epoch       string
	Hash        string
	StartHeight uint64
	EndHeight   uint64
}

type HeightHash struct {
	Epoch  string
	Height uint64
	Hash   string
}

type LatestDataRequest struct {
	Network string `json:"network"`
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
	Epoch  string    `json:"epoch,omitempty"`
	Time   time.Time `json:"time,omitempty"`

	Fee       []TransactionFee `json:"transaction_fee,omitempty"`
	GasWanted uint64           `json:"gas_wanted,omitempty"`
	GasUsed   uint64           `json:"gas_used,omitempty"`

	Memo string `json:"memo,omitempty"`

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

	Sender    []string            `json:"sender,omitempty"`
	Recipient []string            `json:"recipient,omitempty"`
	Validator map[string][]string `json:"validator,omitempty"`
	Feeder    []string            `json:"feeder,omitempty"`
	Withdraw  map[string][]string `json:"withdraw,omitempty"`

	Nonce      int        `json:"nonce,omitempty"`
	Completion *time.Time `json:"completion,omitempty"`

	Error *SubsetEventError `json:"error,omitempty"`

	Amount *TransactionAmount `json:"amount,omitempty"`
}

type SubsetEventError struct {
	Message string `json:"message,omitempty"`
}

type BlockExtra struct {
	Network string `json:"network,omitempty"`
	ChainID string `json:"chain_id,omitempty"`
	Version string `json:"version"`
	Block   Block  `json:"block,omitempty"`
}

// Block
type Block struct {
	ID        uuid.UUID  `json:"id,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`

	Hash   string    `json:"hash,omitempty"`
	Height uint64    `json:"height,omitempty"`
	Time   time.Time `json:"time,omitempty"`
	Epoch  string    `json:"epoch,omitempty"`
}

type TransactionSearch struct {
	Height    uint64    `json:"height"`
	Epoch     string    `json:"epoch"`
	Type      []string  `json:"type"`
	BlockHash string    `json:"block_hash"`
	Account   string    `json:"account"`
	Sender    string    `json:"sender"`
	Receiver  string    `json:"receiver"`
	Memo      string    `json:"memo"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Limit     uint64    `json:"limit"`

	AfterHeight  uint64 `form:"after_id"`
	BeforeHeight uint64 `form:"before_id"`

	Network string `json:"network"`
}
