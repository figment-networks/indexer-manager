package indexer

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/figment-networks/cosmos-indexer/cosmos"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	"github.com/cosmos/cosmos-sdk/x/staking"

	"github.com/figment-networks/cosmos-indexer/model"
	"github.com/figment-networks/indexing-engine/pipeline"
)

const (
	ParserTaskName = "TransactionParser"
)

func NewParserTask() pipeline.Task {
	return &ParserTask{}
}

type ParserTask struct {
}

func (t *ParserTask) GetName() string {
	return ParserTaskName
}

func (t *ParserTask) Run(ctx context.Context, p pipeline.Payload) error {
	fmt.Println("[ParserTask]")

	payload := (p).(*payload)

	raw := payload.RawTransactions

	cdc := makeCodec()
	transactions := make([]*model.Transaction, len(raw))
	for i, rawTx := range raw {
		tx, err := rawToTransaction(rawTx, cdc)
		if err != nil {
			return err
		}
		transactions[i] = tx
	}

	payload.Transactions = transactions
	return nil
}

func rawToTransaction(raw *cosmos.ResultTx, cdc *codec.Codec) (*model.Transaction, error) {
	decoded, err := base64.StdEncoding.DecodeString(raw.TxData)
	if err != nil {
		fmt.Println("decode error:", err)
		return nil, err
	}

	parsed, err := parseTx(cdc, decoded)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return &model.Transaction{
		Height:    util.MustUInt64(raw.Height),
		Hash:      raw.Hash,
		GasWanted: util.MustUInt64(raw.TxResult.GasWanted),
		GasUsed:   util.MustUInt64(raw.TxResult.GasUsed),
		Memo:      parsed.Memo,
	}, nil
}

// todo pass this to parser task?
func makeCodec() *codec.Codec {
	var cdc = codec.New()
	bank.RegisterCodec(cdc)
	staking.RegisterCodec(cdc)
	distr.RegisterCodec(cdc)
	slashing.RegisterCodec(cdc)
	gov.RegisterCodec(cdc)
	crisis.RegisterCodec(cdc)
	auth.RegisterCodec(cdc)
	sdk.RegisterCodec(cdc)
	codec.RegisterCrypto(cdc)
	codec.RegisterEvidences(cdc)
	return cdc
}

// todo probably a better way to parse this.
func parseTx(cdc *codec.Codec, txBytes []byte) (*parsedTx, error) {
	var tx auth.StdTx

	err := cdc.UnmarshalBinaryLengthPrefixed(txBytes, &tx)
	if err != nil {
		return nil, err
	}

	// bz, err := cdc.MarshalJSON(tx)
	// if err != nil {
	// 	return nil, err
	// }

	// buf := bytes.NewBuffer([]byte{})
	// err = json.Indent(buf, bz, "", "  ")
	// if err != nil {
	// 	return nil, err
	// }

	// fmt.Println(buf.String())

	return &parsedTx{
		Memo: tx.GetMemo(),
	}, nil
}

type parsedTx struct {
	Memo string
}
