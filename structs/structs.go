package structs

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/google/uuid"
)

var (
	tenInt = big.NewInt(10)
)

//go:generate swagger generate spec --scan-models -o swagger.json

// Transaction contains the blockchain transaction details
// swagger:model
type Transaction struct {
	// ID of transaction assigned on database write
	ID uuid.UUID `json:"id,omitempty"`
	// Created at
	CreatedAt *time.Time `json:"created_at,omitempty"`
	// Updated at
	UpdatedAt *time.Time `json:"updated_at,omitempty"`

	// Hash of the transaction
	Hash string `json:"hash,omitempty"`
	// BlockHash - hash of the block of transaction
	BlockHash string `json:"block_hash,omitempty"`
	// Height - height of the block of transaction
	Height uint64 `json:"height,omitempty"`

	Epoch string `json:"epoch,omitempty"`
	// ChainID - chain id of transacion
	ChainID string `json:"chain_id,omitempty"`
	// Time - time of transaction
	Time time.Time `json:"time,omitempty"`

	// Fee - Fees for transaction (if applies)
	Fee []TransactionAmount `json:"transaction_fee,omitempty"`
	// GasWanted
	GasWanted uint64 `json:"gas_wanted,omitempty"`
	// GasUsed
	GasUsed uint64 `json:"gas_used,omitempty"`
	// Memo - the description attached to transactions
	Memo string `json:"memo,omitempty"`

	// Version - Version of transaction record
	Version string `json:"version"`
	// Events - Transaction contents
	Events TransactionEvents `json:"events,omitempty"`

	// Raw - Raw transaction bytes
	Raw []byte `json:"raw,omitempty"`

	// RawLog - RawLog transaction's log bytes
	RawLog []byte `json:"raw_log,omitempty"`

	// HasErrors - indicates if Transaction has any errors inside
	HasErrors bool `json:"has_errors"`
}

// TransactionEvents - a set of TransactionEvent
// swagger:model
type TransactionEvents []TransactionEvent

func (te *TransactionEvents) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, &te)
}

// TransactionEvent part of transaction contents
// swagger:model
type TransactionEvent struct {
	// ID UniqueID of event
	ID string `json:"id,omitempty"`
	// The Kind of event
	Kind string `json:"kind,omitempty"`
	// Subcontents of event
	Sub []SubsetEvent `json:"sub,omitempty"`
}

// TransactionAmount structure holding amount information with decimal implementation (numeric * 10 ^ exp)
// swagger:model
type TransactionAmount struct {
	// Textual representation of Amount
	Text string `json:"text,omitempty"`
	// The currency in what amount is returned (if applies)
	Currency string `json:"currency,omitempty"`

	// Numeric part of the amount
	Numeric *big.Int `json:"numeric,omitempty"`
	// Exponential part of amount obviously 0 by default
	Exp int32 `json:"exp,omitempty"`
}

func (t *TransactionAmount) Add(o TransactionAmount) error {
	if t.Currency != o.Currency {
		return fmt.Errorf("coin currency different: %v %v\n", t.Currency, o.Currency)
	}
	expDiff := (t.Exp - o.Exp)

	if expDiff < 0 {
		zerosToAdd := math.Abs(float64(expDiff))
		multiplier := new(big.Int).Exp(tenInt, big.NewInt(int64(zerosToAdd)), nil)
		t.Numeric.Mul(t.Numeric, multiplier)
		t.Numeric.Add(t.Numeric, o.Numeric)
		t.Exp = t.Exp - expDiff
		return nil
	}

	zerosToAdd := int64(expDiff)
	multiplier := new(big.Int).Exp(tenInt, big.NewInt(zerosToAdd), nil)
	tmp := o.Clone()
	tmp.Numeric.Mul(tmp.Numeric, multiplier)
	t.Numeric.Add(tmp.Numeric, t.Numeric)
	return nil
}

func (t *TransactionAmount) Sub(o TransactionAmount) error {
	if t.Currency != o.Currency {
		return fmt.Errorf("coin currency different: %v %v\n", t.Currency, o.Currency)

	}

	expDiff := (t.Exp - o.Exp)

	if expDiff < 0 {
		zerosToAdd := math.Abs(float64(expDiff))
		multiplier := new(big.Int).Exp(tenInt, big.NewInt(int64(zerosToAdd)), nil)
		t.Numeric.Mul(t.Numeric, multiplier)
		t.Numeric.Sub(t.Numeric, o.Numeric)
		t.Exp = t.Exp - expDiff
		return nil
	}

	zerosToAdd := int64(expDiff)
	multiplier := new(big.Int).Exp(tenInt, big.NewInt(zerosToAdd), nil)
	tmp := o.Clone()
	tmp.Numeric.Mul(tmp.Numeric, multiplier)
	t.Numeric.Sub(t.Numeric, tmp.Numeric)
	return nil
}

func (t *TransactionAmount) Clone() TransactionAmount {
	tmp := TransactionAmount{
		Currency: t.Currency,
		Numeric:  &big.Int{},
		Exp:      t.Exp,
	}
	if t.Numeric == nil {
		tmp.Numeric.Set(big.NewInt(0))
		return tmp
	}
	tmp.Numeric.Set(t.Numeric)
	return tmp
}

// SubsetEvent - structure storing main contents of transacion
// swagger:model
type SubsetEvent struct {
	// Type of transaction
	Type   []string `json:"type,omitempty"`
	Action string   `json:"action,omitempty"`
	// Collection from where transaction came from
	Module string `json:"module,omitempty"`
	// List of sender accounts with optional amounts
	Sender []EventTransfer `json:"sender,omitempty"`
	// List of recipient accounts with optional amounts
	Recipient []EventTransfer `json:"recipient,omitempty"`
	// The list of all accounts that took part in the subsetevent
	Node map[string][]Account `json:"node,omitempty"`
	// Transaction nonce
	Nonce string `json:"nonce,omitempty"`
	// Completion time
	Completion *time.Time `json:"completion,omitempty"`
	// List of Amounts
	Amount map[string]TransactionAmount `json:"amount,omitempty"`
	// List of Transfers with amounts and optional recipients
	Transfers map[string][]EventTransfer `json:"transfers,omitempty"`
	// Optional error if occurred
	Error *SubsetEventError `json:"error,omitempty"`
	// Set of additional parameters attached to transaction (used as last resort)
	Additional map[string][]string `json:"additional,omitempty"`
	// SubEvents because some messages are in fact carying another messages inside
	Sub []SubsetEvent `json:"sub,omitempty"`
}

// EventTransfer - Account and Amounts pair
// swagger:model
type EventTransfer struct {
	Account Account             `json:"account,omitempty"`
	Amounts []TransactionAmount `json:"amounts,omitempty"`
}

// Account - Extended Account information
// swagger:model
type Account struct {
	// Unique account identifier
	ID string `json:"id"`
	// External optional account details (if applies)
	Details *AccountDetails `json:"detail,omitempty"`
}

// AccountDetails External optional account details (if applies)
// swagger:model
type AccountDetails struct {
	Description string `json:"description,omitempty"`
	Contact     string `json:"contact,omitempty"`
	Name        string `json:"name,omitempty"`
	Website     string `json:"website,omitempty"`
}

// SubsetEventError  error structure for event
// swagger:model
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
	Network  string   `json:"network"`
	ChainIDs []string `json:"chain_ids"`
	Epoch    string   `json:"epoch"`

	Height     uint64    `json:"height"`
	Type       []string  `json:"type"`
	BlockHash  string    `json:"block_hash"`
	Hash       string    `json:"hash"`
	Account    []string  `json:"account"`
	Sender     []string  `json:"sender"`
	Receiver   []string  `json:"receiver"`
	Memo       string    `json:"memo"`
	BeforeTime time.Time `json:"before_time"`
	AfterTime  time.Time `json:"after_time"`
	Limit      uint64    `json:"limit"`
	Offset     uint64    `json:"offset"`

	AfterHeight  uint64 `form:"after_id"`
	BeforeHeight uint64 `form:"before_id"`

	WithRaw    bool `json:"with_raw"`
	WithRawLog bool `json:"with_raw_log"`
}
