package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"

	"github.com/figment-networks/cosmos-indexer/manager/connectivity/structs"
	"github.com/figment-networks/cosmos-indexer/manager/store"
	"github.com/figment-networks/cosmos-indexer/manager/store/params"

	shared "github.com/figment-networks/cosmos-indexer/structs"
)

type NetworkVersion struct {
	Network string
	Version string
}

// HubbleContractor a format agnostic
type HubbleContractor interface {
	GetCurrentHeight(ctx context.Context, nv NetworkVersion)
	GetCurrentBlock(ctx context.Context, nv NetworkVersion)
	GetBlock(ctx context.Context, nv NetworkVersion, id string)
	GetBlocks(ctx context.Context, nv NetworkVersion)
	GetBlockTimes(ctx context.Context, nv NetworkVersion)
	GetBlockTimesInterval(ctx context.Context, nv NetworkVersion)
	GetTransaction(ctx context.Context, nv NetworkVersion) error
	GetTransactions(ctx context.Context, nv NetworkVersion, startHeight int) error
	GetAccounts(ctx context.Context, nv NetworkVersion)
	GetAccount(ctx context.Context, nv NetworkVersion)

	InsertTransactions(ctx context.Context, nv NetworkVersion, read io.ReadCloser) error
}

type TaskSender interface {
	Send([]structs.TaskRequest) (*structs.Await, error)
}

type HubbleClient struct {
	sender   TaskSender
	storeEng store.DataStore
}

func NewHubbleClient(storeEng store.DataStore) *HubbleClient {
	return &HubbleClient{storeEng: storeEng}
}

func (hc *HubbleClient) LinkSender(sender TaskSender) {
	hc.sender = sender
}

func (hc *HubbleClient) GetCurrentHeight(ctx context.Context, nv NetworkVersion) {

}

func (hc *HubbleClient) GetCurrentBlock(ctx context.Context, nv NetworkVersion) {

}

func (hc *HubbleClient) GetBlock(ctx context.Context, nv NetworkVersion, id string) {

}

func (hc *HubbleClient) GetBlocks(ctx context.Context, nv NetworkVersion) {

}

func (hc *HubbleClient) GetBlockTimes(ctx context.Context, nv NetworkVersion) {

}

func (hc *HubbleClient) GetBlockTimesInterval(ctx context.Context, nv NetworkVersion) {

}

func (hc *HubbleClient) GetTransaction(ctx context.Context, nv NetworkVersion, id string) ([]shared.Transaction, error) {
	return hc.GetTransactions(ctx, nv, shared.HeightRange{"", "", 0, 0})
}

func (hc *HubbleClient) GetTransactions(ctx context.Context, nv NetworkVersion, heightRange shared.HeightRange) ([]shared.Transaction, error) {

	log.Println("waiting for transactions")

	req := []structs.TaskRequest{}

	times := 1
	diff := float64(heightRange.EndHeight - heightRange.StartHeight)
	if diff > 0 {
		times = int(math.Ceil(diff / 1000))
	}

	for i := 0; i < times; i++ {
		b, _ := json.Marshal(shared.HeightRange{
			StartHeight: heightRange.StartHeight + int64(i*100),
			EndHeight:   heightRange.StartHeight + int64((i+1)*100),
			Hash:        heightRange.Hash,
		})

		req = append(req, structs.TaskRequest{
			Network: nv.Network,
			Version: nv.Version,
			Type:    "GetTransactions",
			Payload: b,
		})
	}

	respAwait, err := hc.sender.Send(req)
	if err != nil {
		log.Println("Error Sending data")
		return nil, err
	}

	defer respAwait.Close()

	trs := []shared.Transaction{}

	buff := &bytes.Buffer{}
	dec := json.NewDecoder(buff)

	var receivedTransactions int
WAIT_FOR_ALL_TRANSACTIONS:
	for {
		select {
		case <-ctx.Done():
			return nil, errors.New("Request timed out")
		case response := <-respAwait.Resp:

			if response.Error.Msg != "" {
				return nil, fmt.Errorf("Error getting response: %s", response.Error.Msg)
			}

			if response.Type != "Transaction" {
				continue
			}

			//	if !ok {
			//		return nil, errors.New("Response closed.")
			//	}
			//log.Printf("Got Response !!! %s ", string(response.Payload))
			buff.Reset()
			buff.ReadFrom(bytes.NewReader(response.Payload))
			m := &shared.Transaction{}
			err := dec.Decode(m)
			if err != nil {
				return nil, fmt.Errorf("Error getting response: %w", err)
			}

			err = hc.storeEng.StoreTransaction(
				shared.TransactionExtra{
					Network:     nv.Network,
					ChainID:     nv.Version,
					Transaction: *m,
				})

			if err != nil {
				log.Printf("Error storing transaction : %+v", err)
			}

			trs = append(trs, *m)
			if response.Final {
				receivedTransactions++
			}

			if receivedTransactions == times {
				break WAIT_FOR_ALL_TRANSACTIONS
			}
		}
	}

	log.Println("finished waiting for transactions")
	return trs, nil
}

func (hc *HubbleClient) SearchTransactions(ctx context.Context, nv NetworkVersion, ts shared.TransactionSearch) ([]shared.Transaction, error) {

	return hc.storeEng.GetTransactions(ctx, params.TransactionSearch{
		Network:   nv.Network,
		Height:    ts.Height,
		Type:      ts.Type,
		BlockHash: ts.BlockHash,
		Account:   ts.Account,
		Sender:    ts.Sender,
		Receiver:  ts.Receiver,
		Memo:      ts.Memo,
		StartTime: ts.StartTime,
		EndTime:   ts.EndTime,
		Limit:     ts.Limit,
	})
}

func (hc *HubbleClient) GetAccounts(ctx context.Context, nv NetworkVersion) {

}

func (hc *HubbleClient) GetAccount(ctx context.Context, nv NetworkVersion) {

}

func (hc *HubbleClient) InsertTransactions(ctx context.Context, nv NetworkVersion, readr io.ReadCloser) error {
	defer readr.Close()
	dec := json.NewDecoder(readr)

	_, err := dec.Token()
	if err != nil {
		return err
	}

	for dec.More() {
		req := shared.Transaction{}

		if err := dec.Decode(&req); err != nil {
			return err
		}

		err = hc.storeEng.StoreTransaction(
			shared.TransactionExtra{
				Network:     nv.Network,
				ChainID:     nv.Version,
				Transaction: req,
			})

		if err != nil {
			return err
		}

	}

	_, err = dec.Token()
	if err != nil {
		return err
	}

	return nil

}
