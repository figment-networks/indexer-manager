package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/figment-networks/indexing-engine/metrics"
	"go.uber.org/zap"

	"github.com/figment-networks/indexer-manager/manager/connectivity/structs"
	"github.com/figment-networks/indexer-manager/manager/store"
	"github.com/figment-networks/indexer-manager/manager/store/params"
	shared "github.com/figment-networks/indexer-manager/structs"
)

//go:generate mockgen -destination=./mocks/mock_client.go  -package=mocks github.com/figment-networks/indexer-manager/manager/client TaskSender

// SelfCheck Flag describing should manager check anyway the latest version for network it has
var SelfCheck = true

var ErrIntegrityCheckFailed = errors.New("integrity check failed")

type NetworkVersion struct {
	Network string
	ChainID string
	Version string
}

// ClientContractor a format agnostic
type ClientContractor interface {
	SchedulerContractor
	ControllContractor

	SearchTransactions(ctx context.Context, ts shared.TransactionSearch) ([]shared.Transaction, error)
	GetTransaction(ctx context.Context, nv NetworkVersion, id string) ([]shared.Transaction, error)
	GetTransactions(ctx context.Context, nv NetworkVersion, heightRange shared.HeightRange, batchLimit uint64, silent bool) ([]shared.Transaction, error)
}

type SchedulerContractor interface {
	ScrapeLatest(ctx context.Context, ldr shared.LatestDataRequest) (ldResp shared.LatestDataResponse, er error)
}

type ControllContractor interface {
	InsertTransactions(ctx context.Context, nv NetworkVersion, read io.ReadCloser) error

	CheckMissingTransactions(ctx context.Context, nv NetworkVersion, heightRange shared.HeightRange, window uint64) (missingBlocks, missingTransactions [][2]uint64, err error)
	GetMissingTransactions(ctx context.Context, nv NetworkVersion, heightRange shared.HeightRange, window uint64, async bool, force bool) (run *Run, err error)

	GetRunningTransactions(ctx context.Context) (run []Run, err error)
}

type TaskSender interface {
	Send([]structs.TaskRequest) (*structs.Await, error)
}

type Client struct {
	sender   TaskSender
	storeEng store.DataStore
	logger   *zap.Logger
	runner   *Runner
}

func NewClient(storeEng store.DataStore, logger *zap.Logger, runner *Runner) *Client {
	c := &Client{
		storeEng: storeEng,
		logger:   logger,
	}
	if runner != nil {
		c.runner = runner
		go c.runner.Run()
	}

	return c
}

func (hc *Client) LinkSender(sender TaskSender) {
	hc.sender = sender
}

func (hc *Client) GetTransaction(ctx context.Context, nv NetworkVersion, id string) ([]shared.Transaction, error) {
	return hc.GetTransactions(ctx, nv, shared.HeightRange{Hash: id}, 1, false)
}

// GetTransactions gets transaction range and stores it in the database with respective blocks
func (hc *Client) GetTransactions(ctx context.Context, nv NetworkVersion, heightRange shared.HeightRange, batchLimit uint64, silent bool) ([]shared.Transaction, error) {
	timer := metrics.NewTimer(callDurationGetTransactions)
	defer timer.ObserveDuration()

	times := uint64(1)
	req := []structs.TaskRequest{}

	if heightRange.Hash != "" {
		b, _ := json.Marshal(shared.HeightRange{
			Hash:    heightRange.Hash,
			Network: nv.Network,
			ChainID: nv.ChainID,
		})
		req = append(req, structs.TaskRequest{
			Network: nv.Network,
			ChainID: nv.ChainID,
			Version: nv.Version,
			Type:    shared.ReqIDGetTransactions,
			Payload: b,
		})
	} else {
		diff := float64(heightRange.EndHeight - heightRange.StartHeight)
		if diff == 0 {
			if heightRange.EndHeight == 0 {
				return nil, errors.New("No transaction to get, bad request")
			}
			requestsToGetMetric.Observe(1)

			b, _ := json.Marshal(shared.HeightRange{
				Network:     nv.Network,
				ChainID:     nv.ChainID,
				StartHeight: heightRange.StartHeight,
				EndHeight:   heightRange.EndHeight,
				Hash:        heightRange.Hash,
			})

			req = append(req, structs.TaskRequest{
				Network: nv.Network,
				ChainID: nv.ChainID,
				Version: nv.Version,
				Type:    shared.ReqIDGetTransactions,
				Payload: b,
			})
		} else {
			requestsToGetMetric.Observe(diff)

			var i uint64
			for {
				startH := heightRange.StartHeight + i*uint64(batchLimit)
				endH := heightRange.StartHeight + i*uint64(batchLimit) + uint64(batchLimit) - 1

				if heightRange.EndHeight > 0 && endH > heightRange.EndHeight {
					endH = heightRange.EndHeight
				}

				b, _ := json.Marshal(shared.HeightRange{
					StartHeight: startH,
					EndHeight:   endH,
					Hash:        heightRange.Hash,
					Network:     nv.Network,
					ChainID:     nv.ChainID,
				})

				req = append(req, structs.TaskRequest{
					Network: nv.Network,
					ChainID: nv.ChainID,
					Version: nv.Version,
					Type:    shared.ReqIDGetTransactions,
					Payload: b,
				})

				i++
				if heightRange.EndHeight == endH || heightRange.EndHeight == 0 {
					break
				}
			}
		}
	}

	hc.logger.Info("[Client] Sending request data:", zap.Any("request", req))
	respAwait, err := hc.sender.Send(req)
	if err != nil {
		hc.logger.Error("[Client] Error sending data", zap.Error(err))
		return nil, fmt.Errorf("error sending data in getTransaction: %w", err)
	}

	defer respAwait.Close()

	trs := []shared.Transaction{}

	buff := &bytes.Buffer{}
	dec := json.NewDecoder(buff)

	var receivedFinals uint64
WaitForAllData:
	for {
		select {
		case <-ctx.Done():
			return nil, errors.New("Request timed out")
		case response := <-respAwait.Resp:
			hc.logger.Debug("[Client] Get Transaction received data:", zap.Any("type", response.Type), zap.Any("response", response))

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
					if !silent {
						trs = append(trs, *t)
					}
				case "Block":
					b := &shared.Block{}
					if err := hc.storeBlock(dec, nv.Network, nv.Version, b); err != nil {
						return trs, fmt.Errorf("error storing block: %w", err)
					}
				}
			}

			if response.Final {
				receivedFinals++
			}

			if receivedFinals == times {
				hc.logger.Info("[Client] Received All for", zap.Any("request", req))
				break WaitForAllData
			}
		}
	}
	return trs, nil
}

// SearchTransactions is the search
func (hc *Client) SearchTransactions(ctx context.Context, ts shared.TransactionSearch) ([]shared.Transaction, error) {
	timer := metrics.NewTimer(callDurationSearchTransactions)
	defer timer.ObserveDuration()

	return hc.storeEng.GetTransactions(ctx, params.TransactionSearch{
		Network:      ts.Network,
		ChainIDs:     ts.ChainIDs,
		Epoch:        ts.Epoch,
		Height:       ts.Height,
		Type:         ts.Type,
		BlockHash:    ts.BlockHash,
		Account:      ts.Account,
		Sender:       ts.Sender,
		Receiver:     ts.Receiver,
		Memo:         ts.Memo,
		AfterTime:    ts.AfterTime,
		BeforeTime:   ts.BeforeTime,
		AfterHeight:  ts.AfterHeight,
		BeforeHeight: ts.BeforeHeight,
		Limit:        ts.Limit,
		Offset:       ts.Offset,
		WithRaw:      ts.WithRaw,
	})
}

// InsertTransactions inserts external transactions batch
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
			shared.TransactionWithMeta{
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

// ScrapeLatest scrapes latest data using the latest known block from database
func (hc *Client) ScrapeLatest(ctx context.Context, ldr shared.LatestDataRequest) (ldResp shared.LatestDataResponse, er error) {
	timer := metrics.NewTimer(callDurationScrapeLatest)
	defer timer.ObserveDuration()

	hc.logger.Debug("[Client] ScrapeLatest", zap.Any("received", ldr))

	// (lukanus): self consistency check (optional), so we don;t care about an error
	if ldr.SelfCheck {
		lastBlock, err := hc.storeEng.GetLatestBlock(ctx, shared.BlockWithMeta{ChainID: ldr.ChainID, Network: ldr.Network, Version: ldr.Version})
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
		ChainID: ldr.ChainID,
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
WaitForAllData:
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
				break WaitForAllData
			}
		}
	}

	missing, err := hc.storeEng.BlockContinuityCheck(ctx, shared.BlockWithMeta{ChainID: ldr.Version, Network: ldr.Network}, ldr.LastHeight, 0)
	if err != nil {
		return ldResp, err
	}

	if len(missing) > 0 {
		hc.logger.Error("[Client] Block Continuity check failed", zap.Any("missing", missing))
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
		shared.TransactionWithMeta{
			Network:     network,
			Version:     version,
			Transaction: *m,
		})

	if err != nil {
		return fmt.Errorf("error storing transaction: %w %+v", err, m)
	}
	return nil
}

func (hc *Client) storeBlock(dec *json.Decoder, network string, version string, m *shared.Block) error {

	if err := dec.Decode(m); err != nil {
		return fmt.Errorf("error decoding block: %w", err)
	}

	err := hc.storeEng.StoreBlock(
		shared.BlockWithMeta{
			Network: network,
			Version: version,
			ChainID: m.ChainID,
			Block:   *m,
		})

	if err != nil {
		return fmt.Errorf("error storing block: %w", err)
	}
	return nil
}

func getRanges(in []uint64) (ranges [][2]uint64) {

	sort.SliceStable(in, func(i, j int) bool { return in[i] < in[j] })

	ranges = [][2]uint64{}
	var temp = [2]uint64{}
	for i, height := range in {

		if i == 0 {
			temp[0] = height
			temp[1] = height
			continue
		}

		if temp[1]+1 == height {
			temp[1] = height
			continue
		}

		ranges = append(ranges, temp)
		temp[0] = height
		temp[1] = height

	}
	if temp[1] != 0 {
		ranges = append(ranges, temp)
	}

	return ranges
}

// groupRanges Groups ranges to fit the window of X records
func groupRanges(ranges [][2]uint64, window uint64) (out [][2]uint64) {
	pregroup := [][2]uint64{}

	// (lukanus): first slice all the bigger ranges to get max(window)
	for _, r := range ranges {
		diff := r[1] - r[0]
		if diff > window {
			current := r[0]
		SliceDiff:
			for {
				next := current + window
				if next <= r[1] {
					pregroup = append(pregroup, [2]uint64{current, next})
					current = next + 1
					continue
				}

				pregroup = append(pregroup, [2]uint64{current, r[1]})
				break SliceDiff
			}
		} else {
			pregroup = append(pregroup, r)
		}
	}

	out = [][2]uint64{}

	var temp = [2]uint64{}
	for i, n := range pregroup {
		if i == 0 {
			temp[0] = n[0]
			temp[1] = n[1]
			continue
		}

		diff := n[1] - temp[0]
		if diff <= window {
			temp[1] = n[1]
			continue
		}
		out = append(out, temp)
		temp = [2]uint64{n[0], n[1]}
	}

	if temp[1] != 0 {
		out = append(out, temp)
	}
	return out

}
