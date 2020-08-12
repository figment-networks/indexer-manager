package coda

import (
	"encoding/json"
	"os"
)

type Genesis struct {
	Accounts []GenesisAccount
}

type GenesisAccount struct {
	PK       string  `json:"pk"`
	SK       *string `json:"sk"`
	Balance  string  `json:"balance"`
	Delegate *string `json:"delegate"`
}

// ReadGenesisFile reads and returns genesis file records
func ReadGenesisFile(path string) (*Genesis, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	accounts := []GenesisAccount{}
	if err := json.NewDecoder(f).Decode(&accounts); err != nil {
		return nil, err
	}
	return &Genesis{accounts}, nil
}
