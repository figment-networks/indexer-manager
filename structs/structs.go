package structs

import (
	"encoding/json"
	"errors"
	"math/big"
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

type TransactionWithMeta struct {
	Network     string      `json:"network,omitempty"`
	Version     string      `json:"version,omitempty"`
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

	Height  uint64    `json:"height,omitempty"`
	Epoch   string    `json:"epoch,omitempty"`
	ChainID string    `json:"chain_id,omitempty"`
	Time    time.Time `json:"time,omitempty"`

	Fee       []TransactionAmount `json:"transaction_fee,omitempty"`
	GasWanted uint64              `json:"gas_wanted,omitempty"`
	GasUsed   uint64              `json:"gas_used,omitempty"`

	Memo string `json:"memo,omitempty"`

	Version string            `json:"version"`
	Events  TransactionEvents `json:"events,omitempty"`
	Raw     []byte            `json:"raw,omitempty"`
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

type TransactionAmount struct {
	Text     string `json:"text,omitempty"`
	Currency string `json:"currency,omitempty"`

	// decimal implementation (numeric * 10 ^ exp)
	Numeric *big.Int `json:"numeric,omitempty"`
	Exp     int32    `json:"exp,omitempty"`
}

type SubsetEvent struct {
	Type   string `json:"type,omitempty"`
	Action string `json:"action,omitempty"`
	Module string `json:"module,omitempty"`

	Sender    []EventTransfer `json:"sender,omitempty"`
	Recipient []EventTransfer `json:"recipient,omitempty"`

	Node map[string][]Account `json:"node,omitempty"`

	Nonce      int        `json:"nonce,omitempty"`
	Completion *time.Time `json:"completion,omitempty"`

	Error *SubsetEventError `json:"error,omitempty"`

	Amount map[string]TransactionAmount `json:"amount,omitempty"`
}

type EventTransfer struct {
	Account Account             `json:"account,omitempty"`
	Amounts []TransactionAmount `json:"amounts,omitempty"`
}

type Account struct {
	ID      string          `json:"id"`
	Details *AccountDetails `json:"detail,omitempty"`
}

type AccountDetails struct {
	Description string `json:"description,omitempty"`
	Contact     string `json:"contact,omitempty"`
	Name        string `json:"name,omitempty"`
	Website     string `json:"website,omitempty"`
}
type SubsetEventError struct {
	Message string `json:"message,omitempty"`
}

type BlockWithMeta struct {
	Network string `json:"network,omitempty"`
	ChainID string `json:"chain_id,omitempty"`
	Version string `json:"version,omitempty"`
	Block   Block  `json:"block,omitempty"`
}

// Block
type Block struct {
	ID        uuid.UUID  `json:"id,omitempty"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
	UpdatedAt *time.Time `json:"updated_at,omitempty"`

	Hash    string    `json:"hash,omitempty"`
	Height  uint64    `json:"height,omitempty"`
	Time    time.Time `json:"time,omitempty"`
	Epoch   string    `json:"epoch,omitempty"`
	ChainID string    `json:"chain_id,omitempty"`

	NumberOfTransactions uint64 `json:"num_txs,omitempty"`
}

type TransactionSearch struct {
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
	Offset    uint64    `json:"offset"`

	AfterHeight  uint64 `form:"after_id"`
	BeforeHeight uint64 `form:"before_id"`

	Network string `json:"network"`
	ChainID string `json:"chain_id"`
	Epoch   string `json:"epoch"`

	WithRaw bool `json:"with_raw"`
}
