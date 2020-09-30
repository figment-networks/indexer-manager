package cosmos

import (
	"errors"

	shared "github.com/figment-networks/cosmos-indexer/structs"

	sdk "github.com/cosmos/cosmos-sdk/types"
	crisis "github.com/cosmos/cosmos-sdk/x/crisis"
)

func mapCrisisVerifyInvariantToSub(msg sdk.Msg) (se shared.SubsetEvent, er error) {
	mvi, ok := msg.(crisis.MsgVerifyInvariant)
	if !ok {
		return se, errors.New("Not a verify_invariant type")
	}

	return shared.SubsetEvent{
		Type:   []string{"verify_invariant"},
		Module: "crisis",
		Sender: []shared.EventTransfer{{
			Account: shared.Account{ID: mvi.Sender.String()},
		}},
		Additional: map[string][]string{
			"invariant_route":       {mvi.InvariantRoute},
			"invariant_module_name": {mvi.InvariantModuleName},
		},
	}, nil
}
