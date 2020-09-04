package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/figment-networks/indexing-engine/metrics"
	"go.uber.org/zap"

	"github.com/figment-networks/cosmos-indexer/manager/connectivity/structs"
	"github.com/figment-networks/cosmos-indexer/manager/store"
	"github.com/figment-networks/cosmos-indexer/manager/store/params"
	shared "github.com/figment-networks/cosmos-indexer/structs"
)

// SelfCheck Flag describing should manager check anyway the latest version for network it has
var SelfCheck = true

var ErrIntegrityCheckFailed = errors.New("integrity check failed")

type NetworkVersion struct {
	Network string
	Version string
}

// HubbleContractor a format agnostic
type HubbleContractor interface {
	SchedulerContractor

	GetAccounts(ctx context.Context, nv NetworkVersion)
	GetAccount(ctx context.Context, nv NetworkVersion)
	GetCurrentHeight(ctx context.Context, nv NetworkVersion)
	GetCurrentBlock(ctx context.Context, nv NetworkVersion)
	GetBlock(ctx context.Context, nv NetworkVersion, id string)
	GetBlocks(ctx context.Context, nv NetworkVersion)
	GetBlockTimes(ctx context.Context, nv NetworkVersion)
	GetBlockTimesInterval(ctx context.Context, nv NetworkVersion)

	SearchTransactions(ctx context.Context, nv NetworkVersion, ts shared.TransactionSearch) ([]shared.Transaction, error)
	GetTransaction(ctx context.Context, nv NetworkVersion, id string) ([]shared.Transaction, error)
	GetTransactions(ctx context.Context, nv NetworkVersion, heightRange shared.HeightRange) ([]shared.Transaction, error)
	InsertTransactions(ctx context.Context, nv NetworkVersion, read io.ReadCloser) error
}

type SchedulerContractor interface {
	ScrapeLatest(ctx context.Context, ldr shared.LatestDataRequest) (ldResp shared.LatestDataResponse, er error)
}

type TaskSender interface {
	Send([]structs.TaskRequest) (*structs.Await, error)
}

type Client struct {
	sender   TaskSender
	storeEng store.DataStore
	logger   *zap.Logger
}

func NewClient(storeEng store.DataStore, logger *zap.Logger) *Client {
	return &Client{storeEng: storeEng, logger: logger}
}

func (hc *Client) LinkSender(sender TaskSender) {
	hc.sender = sender
}

func (hc *Client) GetAccounts(ctx context.Context, nv NetworkVersion) {}

func (hc *Client) GetAccount(ctx context.Context, nv NetworkVersion) {}

func (hc *Client) GetCurrentHeight(ctx context.Context, nv NetworkVersion) {}

func (hc *Client) GetCurrentBlock(ctx context.Context, nv NetworkVersion) {}

func (hc *Client) GetBlock(ctx context.Context, nv NetworkVersion, id string) {}

func (hc *Client) GetBlocks(ctx context.Context, nv NetworkVersion) {}

func (hc *Client) GetBlockTimes(ctx context.Context, nv NetworkVersion) {}

func (hc *Client) GetBlockTimesInterval(ctx context.Context, nv NetworkVersion) {}

func (hc *Client) GetTransaction(ctx context.Context, nv NetworkVersion, id string) ([]shared.Transaction, error) {
	return hc.GetTransactions(ctx, nv, shared.HeightRange{Hash: id})
}

func (hc *Client) GetTransactions(ctx context.Context, nv NetworkVersion, heightRange shared.HeightRange) ([]shared.Transaction, error) {
	timer := metrics.NewTimer(callDurationGetTransactions)
	defer timer.ObserveDuration()

	times := 1
	req := []structs.TaskRequest{}

	if heightRange.Hash != "" {
		b, _ := json.Marshal(shared.HeightRange{Hash: heightRange.Hash})
		req = append(req, structs.TaskRequest{
			Network: nv.Network,
			Version: nv.Version,
			Type:    "GetTransactions",
			Payload: b,
		})
	} else {
		diff := float64(heightRange.EndHeight - heightRange.StartHeight)
		if diff == 0 && heightRange.Hash == "" {
			return nil, errors.New("No transaction to get, bad request")
		}
		requestsToGetMetric.Observe(diff)

		if diff > 0 {
			times = int(math.Ceil(diff / 100))
		}

		for i := 0; i < times; i++ {
			endH := heightRange.StartHeight + uint64((i+1)*100)
			if heightRange.EndHeight > 0 && endH > heightRange.EndHeight {
				endH = heightRange.EndHeight
			}

			b, _ := json.Marshal(shared.HeightRange{
				StartHeight: heightRange.StartHeight + uint64(i*100),
				EndHeight:   endH,
				Hash:        heightRange.Hash,
			})

			req = append(req, structs.TaskRequest{
				Network: nv.Network,
				Version: nv.Version,
				Type:    "GetTransactions",
				Payload: b,
			})
		}
	}

	respAwait, err := hc.sender.Send(req)
	if err != nil {
		hc.logger.Error("[Client] Error sending data", zap.Error(err))
		return nil, fmt.Errorf("error sending data in getTransaction: %w", err)
	}

	defer respAwait.Close()

	trs := []shared.Transaction{}

	buff := &bytes.Buffer{}
	dec := json.NewDecoder(buff)

	var receivedTransactions int
WAIT_FOR_ALL_DATA:
	for {
		select {
		case <-ctx.Done():
			return nil, errors.New("Request timed out")
		case response := <-respAwait.Resp:
			if response.Error.Msg != "" {
				return trs, fmt.Errorf("error getting response: %s", response.Error.Msg)
			}

			if response.Type == "Transaction" || response.Type == "Block" {
				buff.Reset()
				buff.ReadFrom(bytes.NewReader(response.Payload))
				switch response.Type {
				case "Transaction":
					t := &shared.Transaction{}
					if err := hc.storeTransaction(dec, nv.Network, nv.Version, t); err != nil {
						return trs, fmt.Errorf("error storing transaction: %w", err)
					}
					trs = append(trs, *t)
				case "Block":
					b := &shared.Block{}
					if err := hc.storeBlock(dec, nv.Network, nv.Version, b); err != nil {
						return trs, fmt.Errorf("error storing block: %w", err)
					}
				}
			}

			if response.Final {
				receivedTransactions++
			}

			if receivedTransactions == times {
				break WAIT_FOR_ALL_DATA
			}
		}
	}
	return trs, nil
}

func (hc *Client) SearchTransactions(ctx context.Context, nv NetworkVersion, ts shared.TransactionSearch) ([]shared.Transaction, error) {
	timer := metrics.NewTimer(callDurationSearchTransactions)
	defer timer.ObserveDuration()
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
		Offset:    ts.Offset,
	})
}

func (hc *Client) InsertTransactions(ctx context.Context, nv NetworkVersion, readr io.ReadCloser) error {
	timer := metrics.NewTimer(callDurationInsertTransactions)
	defer timer.ObserveDuration()

	defer readr.Close()
	dec := json.NewDecoder(readr)

	_, err := dec.Token()
	if err != nil {
		return fmt.Errorf("error decoding json - wrong format: %w", err)

	}

	inserted := 0
	for dec.More() {
		req := shared.Transaction{}

		if err := dec.Decode(&req); err != nil {
			return fmt.Errorf("error decoding transaction %w", err)
		}

		err = hc.storeEng.StoreTransaction(
			shared.TransactionExtra{
				Network:     nv.Network,
				ChainID:     nv.Version,
				Transaction: req,
			})

		if err != nil {
			return fmt.Errorf("error storing transaction: %w", err)
		}

		inserted++

	}

	_, err = dec.Token()
	return err

}

func (hc *Client) ScrapeLatest(ctx context.Context, ldr shared.LatestDataRequest) (ldResp shared.LatestDataResponse, er error) {
	timer := metrics.NewTimer(callDurationScrapeLatest)
	defer timer.ObserveDuration()

	hc.logger.Debug("[Client] ScrapeLatest", zap.Any("received", ldr))

	// (lukanus): self consistency check (optional), so we don;t care about an error
	if ldr.SelfCheck {
		lastBlock, err := hc.storeEng.GetLatestBlock(ctx, shared.BlockExtra{ChainID: ldr.Version, Network: ldr.Network})
		if err == nil {
			if lastBlock.Hash != "" {
				ldr.LastHash = lastBlock.Hash
			}

			if lastBlock.Height > 0 {
				ldr.LastHeight = lastBlock.Height
			}

			if !lastBlock.Time.IsZero() {
				ldr.LastTime = lastBlock.Time
			}
		} else if err != params.ErrNotFound {
			return ldResp, fmt.Errorf("error getting latest transaction data : %w", err)
		}
		hc.logger.Debug("[Client] ScrapeLatest", zap.Any("last block", lastBlock))
	}

	taskReq, err := json.Marshal(ldr)
	if err != nil {
		return ldResp, fmt.Errorf("error marshaling data : %w", err)
	}

	respAwait, err := hc.sender.Send([]structs.TaskRequest{{
		Network: ldr.Network,
		Version: ldr.Version,
		Type:    "GetLatest",
		Payload: taskReq,
	}})

	if err != nil {
		return ldResp, fmt.Errorf("error sending data in getTransaction: %w", err)
	}
	defer respAwait.Close()

	buff := &bytes.Buffer{}
	dec := json.NewDecoder(buff)

	ldResp = shared.LatestDataResponse{}
WAIT_FOR_ALL_DATA:
	for {
		select {
		case <-ctx.Done():
			return ldResp, errors.New("request timed out")
		case response := <-respAwait.Resp:
			if response.Error.Msg != "" {
				return ldResp, fmt.Errorf("error getting response: %s", response.Error.Msg)
			}

			if response.Type == "Transaction" || response.Type == "Block" {
				buff.Reset()
				buff.ReadFrom(bytes.NewReader(response.Payload))
				switch response.Type {
				case "Transaction":
					t := &shared.Transaction{}
					if err := hc.storeTransaction(dec, ldr.Network, ldr.Version, t); err != nil {
						return ldResp, fmt.Errorf("error storing transaction: %w", err)
					}
				case "Block":
					b := &shared.Block{}
					if err := hc.storeBlock(dec, ldr.Network, ldr.Version, b); err != nil {
						return ldResp, fmt.Errorf("error storing block: %w", err)
					}
					if ldResp.LastTime.IsZero() || ldResp.LastHeight <= b.Height {
						ldResp.LastEpoch = b.Epoch
						ldResp.LastHash = b.Hash
						ldResp.LastHeight = b.Height
						ldResp.LastTime = b.Time
					}
				}
			}

			if response.Final {
				break WAIT_FOR_ALL_DATA
			}
		}
	}

	missing, err := hc.storeEng.BlockContinuityCheck(ctx, shared.BlockExtra{ChainID: ldr.Version, Network: ldr.Network}, ldr.LastHeight)
	if err != nil {
		return ldResp, err
	}

	if len(missing) > 0 {
		hc.logger.Debug("[Client] ScrapeLatest", zap.Any("missing", missing))
		return ldResp, ErrIntegrityCheckFailed
	}

	return ldResp, nil
}

func (hc *Client) storeTransaction(dec *json.Decoder, network string, version string, m *shared.Transaction) error {
	err := dec.Decode(m)
	if err != nil {
		return fmt.Errorf("error decoding transaction: %w", err)
	}

	err = hc.storeEng.StoreTransaction(
		shared.TransactionExtra{
			Network:     network,
			ChainID:     version,
			Transaction: *m,
		})

	if err != nil {
		return fmt.Errorf("error storing transaction: %w", err)
	}
	return nil
}

func (hc *Client) storeBlock(dec *json.Decoder, network string, version string, m *shared.Block) error {

	if err := dec.Decode(m); err != nil {
		return fmt.Errorf("error decoding block: %w", err)
	}

	err := hc.storeEng.StoreBlock(
		shared.BlockExtra{
			Network: network,
			ChainID: version,
			Block:   *m,
		})

	if err != nil {
		return fmt.Errorf("error storing block: %w", err)
	}
	return nil
}
