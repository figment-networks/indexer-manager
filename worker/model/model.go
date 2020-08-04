package model

import "time"

// Model contains basic model data
type Model struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type HeightRange struct {
	StartHeight int64
	EndHeight   int64
}

// Transaction contains the blockchain transaction details
type Transaction struct {
	*Model

	Height    uint64 `json:"height"`
	Hash      string `json:"hash"`
	GasWanted uint64 `json:"gas_wanted"`
	GasUsed   uint64 `json:"gas_used"`
	Memo      string `json:"memo"`
}
