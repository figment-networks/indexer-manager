package cosmos

import (
	"errors"

	shared "github.com/figment-networks/cosmos-indexer/structs"

	sdk "github.com/cosmos/cosmos-sdk/types"
	slashing "github.com/cosmos/cosmos-sdk/x/slashing"
)

func mapSlashingUnjailToSub(msg sdk.Msg) (se shared.SubsetEvent, er error) {
	unjail, ok := msg.(slashing.MsgUnjail)
	if !ok {
		return se, errors.New("Not a unjail type")
	}

	return shared.SubsetEvent{
		Type:   []string{"unjail"},
		Module: "slashing",
		Node:   map[string][]shared.Account{"validator": {{ID: unjail.ValidatorAddr.String()}}},
	}, nil
}
