package coda

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcutil/base58"
	"github.com/figment-networks/cosmos-indexer/structs"
	cStruct "github.com/figment-networks/cosmos-indexer/worker/connectivity/structs"
	"github.com/google/uuid"
)

const (
	TxTypePayment     = "payment"
	TxTypeDelegation  = "delegation"
	TxTypeFee         = "fee"
	TxTypeSnarkFee    = "snark_fee"
	TxTypeBlockReward = "block_reward"
)

var (
	errNoProtocolState   = errors.New("no protocol state")
	errNoConsensusState  = errors.New("no consensus state")
	errNoBlockchainState = errors.New("no blockchain state")
)

// BlockHeight returns a parsed block height
func BlockHeight(input *Block) uint64 {
	// NOTE: Coda's height starts at height=2!
	return MustUInt64(input.ProtocolState.ConsensusState.BlockHeight)
}

// BlockTime returns a parsed block time
func BlockTime(input *Block) time.Time {
	return MustTime(input.ProtocolState.BlockchainState.Date)
}

func blockCheck(input *Block) error {
	if input.ProtocolState == nil {
		return errNoProtocolState
	}
	if input.ProtocolState.ConsensusState == nil {
		return errNoConsensusState
	}
	if input.ProtocolState.BlockchainState == nil {
		return errNoBlockchainState
	}
	return nil
}

func UserTransaction(block *Block, t *UserCommand) structs.Transaction {
	ttype := TxTypePayment
	if t.IsDelegation {
		ttype = TxTypeDelegation
	}

	tran := structs.Transaction{
		Time:      BlockTime(block),
		Height:    BlockHeight(block),
		Hash:      t.ID,
		BlockHash: block.StateHash,
	}
	am, _ := strconv.ParseFloat(t.Amount, 10)

	tran.Events = structs.TransactionEvents{{
		ID:   "0",
		Kind: ttype,
		Sub: []structs.SubsetEvent{{
			Type:      ttype,
			Sender:    []string{t.From},
			Recipient: []string{t.To},
			Nonce:     t.Nonce,
			Amount: &structs.TransactionAmount{
				Text:    t.Amount,
				Numeric: am,
			},
		}},
	}}

	if t.Fee != "" {
		fFee, _ := strconv.ParseFloat(t.Fee, 10)
		tran.Fee = []structs.TransactionFee{{
			Text:    t.Fee,
			Numeric: fFee,
		}}
	}
	if text := ParseMemoText(t.Memo); len(text) > 0 {
		tran.Memo = text
	}

	return tran
}

func BlockRewardTransaction(block *Block) structs.Transaction {

	tran := structs.Transaction{
		Time:      BlockTime(block),
		Height:    BlockHeight(block),
		Hash:      SHA1(block.StateHash + block.Transactions.CoinbaseReceiver.PublicKey),
		BlockHash: block.StateHash,
	}

	am, _ := strconv.ParseFloat(block.Transactions.Coinbase, 10)
	tran.Events = structs.TransactionEvents{{
		ID:   "0",
		Kind: TxTypeBlockReward,
		Sub: []structs.SubsetEvent{{
			Type:      TxTypeBlockReward,
			Recipient: []string{block.Transactions.CoinbaseReceiver.PublicKey},
			Amount: &structs.TransactionAmount{
				Text:    block.Transactions.Coinbase,
				Numeric: am,
			},
		}},
	}}

	return tran
}

func MapFeeTransaction(block *Block, transfer *FeeTransfer, snark bool) structs.Transaction {
	tran := structs.Transaction{
		Time:      BlockTime(block),
		Height:    BlockHeight(block),
		Hash:      SHA1(strings.Join([]string{block.StateHash, transfer.Recipient, transfer.Fee}, "")),
		BlockHash: block.StateHash,
	}

	ttype := TxTypeFee
	if snark {
		ttype = TxTypeSnarkFee
	}
	am, _ := strconv.ParseFloat(transfer.Fee, 10)

	sEv := structs.SubsetEvent{
		Type:      ttype,
		Recipient: []string{transfer.Recipient},
		Amount: &structs.TransactionAmount{
			Text:    transfer.Fee,
			Numeric: am,
		},
	}
	if snark {
		sEv.Sender = []string{block.Creator}
	}

	tran.Events = structs.TransactionEvents{{
		ID:   "0",
		Kind: ttype,
		Sub:  []structs.SubsetEvent{sEv},
	}}

	return tran
}

// Transactions returns a list of transactions from the coda input
func MapTransactions(taskID uuid.UUID, block *Block, resp chan cStruct.OutResp) error {
	if block.Transactions == nil {
		return nil
	}
	if block.Transactions.UserCommands == nil {
		return nil
	}

	//countAll := getTransactionCount(block)

	// Add the block reward transaction
	if block.Transactions.Coinbase != "0" && block.Transactions.CoinbaseReceiver != nil {
		t := BlockRewardTransaction(block)
		//		if err != nil {
		//			return err
		//		}
		resp <- cStruct.OutResp{
			ID:      taskID,
			Type:    "Transaction",
			Payload: t,
		}
	}

	// Add user transactions
	commands := block.Transactions.UserCommands
	for _, cmd := range commands {
		t := UserTransaction(block, cmd)
		//	if err != nil {
		//		return err
		//	}

		resp <- cStruct.OutResp{
			ID:      taskID,
			Type:    "Transaction",
			Payload: t,
		}
	}

	// Add snarker fees transactions
	snarkerIDs := map[string]bool{}
	for _, job := range block.SnarkJobs {
		snarkerIDs[job.Prover] = true
	}

	feeTransfers := block.Transactions.FeeTransfer
	for _, transfer := range feeTransfers {
		var feeTx structs.Transaction
		var err error

		_, ok := snarkerIDs[transfer.Recipient]
		feeTx = MapFeeTransaction(block, transfer, ok)

		if err != nil {
			return err
		}
		resp <- cStruct.OutResp{
			ID:      taskID,
			Type:    "Transaction",
			Payload: feeTx,
		}
	}

	return nil
}

/*
func getTransactionCount(block *Block) uint64 {
	var count = 0
	// Add the block reward transaction
	if block.Transactions.Coinbase != "0" && block.Transactions.CoinbaseReceiver != nil {
		count++
	}
	return uint64(count + len(block.Transactions.UserCommands) + len(block.Transactions.FeeTransfer))
}*/

func FeeTransfers(block *Block) ([]TransactionFeeTransfer, error) {
	if block.Transactions == nil {
		return nil, nil
	}
	if block.Transactions.FeeTransfer == nil {
		return nil, nil
	}

	transfers := block.Transactions.FeeTransfer
	result := make([]TransactionFeeTransfer, len(transfers))

	for i, t := range transfers {
		result[i].Height = BlockHeight(block)
		result[i].Time = BlockTime(block)
		result[i].Recipient = t.Recipient
		result[i].Amount = MustUInt64(t.Fee)
	}

	return result, nil
}

// ParseMemoText returns memo plaintext from base58 data
func ParseMemoText(input string) string {
	data := base58.Decode(input)
	data = bytes.ReplaceAll(data[3:32], []byte("\x00"), []byte(""))
	result := strings.ToValidUTF8(string(data), "")
	return result
}

func SHA1(input string) string {
	h := sha1.New()
	h.Write([]byte(input))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// MustUInt64 returns an UInt64 value without an error
func MustUInt64(input string) uint64 {
	v, err := strconv.ParseUint(input, 10, 64)
	if err != nil {
		v = 0
	}
	return v
}

// MustTime returns a time from a string
func MustTime(input string) time.Time {
	t, err := ParseTime(input)
	if err != nil {
		return time.Time{}
	}
	return *t
}

// ParseTime returns a timestamp from a string
func ParseTime(input string) (*time.Time, error) {
	msec, err := strconv.ParseInt(input, 10, 64)
	if err != nil {
		return nil, err
	}
	t := time.Unix(0, msec*1000000)
	return &t, nil
}

type TransactionFeeTransfer struct {
	Height    uint64    `json:"height"`
	Time      time.Time `json:"time"`
	Recipient string    `json:"recipient"`
	Amount    uint64    `json:"amount"`
}

func (t TransactionFeeTransfer) Validate() error {
	if t.Height == 0 {
		return errors.New("height is invalid")
	}
	if t.Time.IsZero() {
		return errors.New("time is invalid")
	}
	if t.Recipient == "" {
		return errors.New("recipient is required")
	}
	return nil
}
