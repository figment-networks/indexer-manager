package cosmos

import (
	"errors"
	"strconv"

	shared "github.com/figment-networks/cosmos-indexer/structs"

	sdk "github.com/cosmos/cosmos-sdk/types"
	evidence "github.com/cosmos/cosmos-sdk/x/evidence"
)

func mapEvidenceSubmitEvidenceToSub(msg sdk.Msg) (se shared.SubsetEvent, er error) {
	mse, ok := msg.(evidence.MsgSubmitEvidence)
	if !ok {
		return se, errors.New("Not a submit_evidence type")
	}

	return shared.SubsetEvent{
		Type:   []string{"submit_evidence"},
		Module: "evidence",
		Node:   map[string][]shared.Account{"submitter": {{ID: mse.Submitter.String()}}},
		Additional: map[string][]string{
			"evidence_consensus":       {mse.Evidence.GetConsensusAddress().String()},
			"evidence_height":          {strconv.FormatInt(mse.Evidence.GetHeight(), 10)},
			"evidence_total_power":     {strconv.FormatInt(mse.Evidence.GetTotalPower(), 10)},
			"evidence_validator_power": {strconv.FormatInt(mse.Evidence.GetValidatorPower(), 10)},
		},
	}, nil
}
