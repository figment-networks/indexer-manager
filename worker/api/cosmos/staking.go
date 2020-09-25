package cosmos

import (
	"errors"

	shared "github.com/figment-networks/cosmos-indexer/structs"

	sdk "github.com/cosmos/cosmos-sdk/types"
	staking "github.com/cosmos/cosmos-sdk/x/staking"
)

func mapStakingUndelegateToSub(msg sdk.Msg) (se shared.SubsetEvent, err error) {
	u, ok := msg.(staking.MsgUndelegate)
	if !ok {
		return se, errors.New("Not a begin_unbonding type")
	}

	return shared.SubsetEvent{
		Type:   []string{"begin_unbonding"},
		Module: "staking",
		Node: map[string][]shared.Account{
			"delegator": {{ID: u.DelegatorAddress.String()}},
			"validator": {{ID: u.ValidatorAddress.String()}},
		},
		Amount: map[string]shared.TransactionAmount{
			"undelegate": {
				Currency: u.Amount.Denom,
				Numeric:  u.Amount.Amount.BigInt(),
				Text:     u.Amount.String(),
			},
		},
	}, err
}

func mapStakingDelegateToSub(msg sdk.Msg) (se shared.SubsetEvent, err error) {
	d, ok := msg.(staking.MsgDelegate)
	if !ok {
		return se, errors.New("Not a delegate type")
	}
	return shared.SubsetEvent{
		Type:   []string{"delegate"},
		Module: "staking",
		Node: map[string][]shared.Account{
			"delegator": {{ID: d.DelegatorAddress.String()}},
			"validator": {{ID: d.ValidatorAddress.String()}},
		},
		Amount: map[string]shared.TransactionAmount{
			"delegate": {
				Currency: d.Amount.Denom,
				Numeric:  d.Amount.Amount.BigInt(),
				Text:     d.Amount.String(),
			},
		},
	}, err
}

func mapStakingBeginRedelegateToSub(msg sdk.Msg) (se shared.SubsetEvent, err error) {
	br, ok := msg.(staking.MsgBeginRedelegate)
	if !ok {
		return se, errors.New("Not a begin_redelegate type")
	}

	return shared.SubsetEvent{
		Type:   []string{"begin_redelegate"},
		Module: "staking",
		Node: map[string][]shared.Account{
			"delegator":             {{ID: br.DelegatorAddress.String()}},
			"validator_destination": {{ID: br.ValidatorDstAddress.String()}},
			"validator_source":      {{ID: br.ValidatorDstAddress.String()}},
		},
		Amount: map[string]shared.TransactionAmount{
			"delegate": {
				Currency: br.Amount.Denom,
				Numeric:  br.Amount.Amount.BigInt(),
				Text:     br.Amount.String(),
			},
		},
	}, err
}

func mapStakingCreateValidatorToSub(msg sdk.Msg) (se shared.SubsetEvent, err error) {
	ev, ok := msg.(staking.MsgCreateValidator)
	if !ok {
		return se, errors.New("Not a create_validator type")
	}
	return shared.SubsetEvent{
		Type:   []string{"create_validator"},
		Module: "distribution",
		Node: map[string][]shared.Account{
			"delegator": {{ID: ev.DelegatorAddress.String()}},
			"validator": {
				{
					ID: ev.ValidatorAddress.String(),
					Details: &shared.AccountDetails{
						Name:        ev.Description.Moniker,
						Description: ev.Description.Details,
						Contact:     ev.Description.SecurityContact,
						Website:     ev.Description.Website,
					},
				},
			},
		},
		Amount: map[string]shared.TransactionAmount{
			"self_delegation": {
				Currency: ev.Value.Denom,
				Numeric:  ev.Value.Amount.BigInt(),
				Text:     ev.Value.String(),
			},
			"self_delegation_min": {
				Text:    ev.MinSelfDelegation.String(),
				Numeric: ev.MinSelfDelegation.BigInt(),
			},
			"commission_rate": {
				Text:    ev.Commission.Rate.String(),
				Numeric: ev.Commission.Rate.Int,
			},
			"commission_max_rate": {
				Text:    ev.Commission.MaxRate.String(),
				Numeric: ev.Commission.MaxRate.Int,
			},
			"commission_max_change_rate": {
				Text:    ev.Commission.MaxChangeRate.String(),
				Numeric: ev.Commission.MaxChangeRate.Int,
			}},
	}, err
}

func mapStakingEditValidatorToSub(msg sdk.Msg) (se shared.SubsetEvent, err error) {
	ev, ok := msg.(staking.MsgEditValidator)
	if !ok {
		return se, errors.New("Not a edit_validator type")
	}
	return shared.SubsetEvent{
		Type:   []string{"edit_validator"},
		Module: "distribution",
		Node: map[string][]shared.Account{
			"validator": {
				{
					ID: ev.ValidatorAddress.String(),
					Details: &shared.AccountDetails{
						Name:        ev.Description.Moniker,
						Description: ev.Description.Details,
						Contact:     ev.Description.SecurityContact,
						Website:     ev.Description.Website,
					},
				},
			},
		},
		Amount: map[string]shared.TransactionAmount{
			"self_delegation_min": {
				Text:    ev.MinSelfDelegation.String(),
				Numeric: ev.MinSelfDelegation.BigInt(),
			},
			"commission_rate": {
				Text:    ev.CommissionRate.String(),
				Numeric: ev.CommissionRate.Int,
			}},
	}, err
}
