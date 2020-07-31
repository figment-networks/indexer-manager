package model

// Transaction contains the blockchain transaction details
type Transaction struct {
	*Model

	Height    uint64 `json:"height"`
	Hash      string `json:"hash"`
	GasWanted uint64 `json:"gas_wanted"`
	GasUsed   uint64 `json:"gas_used"`
	Memo      string `json:"memo"`
}
