package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"time"

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
	ChainID string
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
	GetTransactions(ctx context.Context, nv NetworkVersion, heightRange shared.HeightRange, perRequest int, silent bool) ([]shared.Transaction, error)
	InsertTransactions(ctx context.Context, nv NetworkVersion, read io.ReadCloser) error

	CheckMissingTransactions(ctx context.Context, nv NetworkVersion, heightRange shared.HeightRange, window uint64) (missingBlocks, missingTransactions [][2]uint64, err error)
	GetMissingTransactions(ctx context.Context, nv NetworkVersion, heightRange shared.HeightRange, window uint64, async bool, force bool) (run *Run, err error)

	GetRunningTransactions(ctx context.Context) (run []Run, err error)
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
	runner   *Runner
}

func NewClient(storeEng store.DataStore, logger *zap.Logger) *Client {
	c := &Client{
		storeEng: storeEng,
		logger:   logger,
		runner:   NewRunner(),
	}
	go c.runner.Run()
	return c
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
	return hc.GetTransactions(ctx, nv, shared.HeightRange{Hash: id}, 1, false)
}

// GetTransactions gets transaction range and stores it in the database with respective blocks
func (hc *Client) GetTransactions(ctx context.Context, nv NetworkVersion, heightRange shared.HeightRange, perRequest int, silent bool) ([]shared.Transaction, error) {
	timer := metrics.NewTimer(callDurationGetTransactions)
	defer timer.ObserveDuration()

	times := 1
	req := []structs.TaskRequest{}

	if heightRange.Hash != "" {
		b, _ := json.Marshal(shared.HeightRange{Hash: heightRange.Hash})
		req = append(req, structs.TaskRequest{
			Network: nv.Network,
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
				StartHeight: heightRange.StartHeight,
				EndHeight:   heightRange.EndHeight,
				Hash:        heightRange.Hash,
			})

			req = append(req, structs.TaskRequest{
				Network: nv.Network,
				Version: nv.Version,
				Type:    shared.ReqIDGetTransactions,
				Payload: b,
			})
		} else {
			requestsToGetMetric.Observe(diff)

			if diff > 0 {
				times = int(math.Ceil(diff / float64(perRequest)))
			}

			for i := 0; i < times; i++ {
				endH := heightRange.StartHeight + uint64((i+1)*perRequest)
				if heightRange.EndHeight > 0 && endH > heightRange.EndHeight {
					endH = heightRange.EndHeight
				}

				b, _ := json.Marshal(shared.HeightRange{
					StartHeight: heightRange.StartHeight + uint64(i*perRequest),
					EndHeight:   endH,
					Hash:        heightRange.Hash,
				})

				req = append(req, structs.TaskRequest{
					Network: nv.Network,
					Version: nv.Version,
					Type:    shared.ReqIDGetTransactions,
					Payload: b,
				})
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

	var receivedFinals int
WAIT_FOR_ALL_DATA:
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
				break WAIT_FOR_ALL_DATA
			}
		}
	}
	return trs, nil
}

// SearchTransactions is the search
func (hc *Client) SearchTransactions(ctx context.Context, nv NetworkVersion, ts shared.TransactionSearch) ([]shared.Transaction, error) {
	timer := metrics.NewTimer(callDurationSearchTransactions)
	defer timer.ObserveDuration()

	return hc.storeEng.GetTransactions(ctx, params.TransactionSearch{
		Network:      nv.Network,
		ChainID:      nv.ChainID,
		Height:       ts.Height,
		Type:         ts.Type,
		BlockHash:    ts.BlockHash,
		Account:      ts.Account,
		Sender:       ts.Sender,
		Receiver:     ts.Receiver,
		Memo:         ts.Memo,
		StartTime:    ts.StartTime,
		EndTime:      ts.EndTime,
		AfterHeight:  ts.AfterHeight,
		BeforeHeight: ts.BeforeHeight,
		Limit:        ts.Limit,
		Offset:       ts.Offset,
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
		lastBlock, err := hc.storeEng.GetLatestBlock(ctx, shared.BlockWithMeta{ChainID: ldr.Version, Network: ldr.Network})
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

// CheckMissingTransactions checks consistency of database if every transaction is written correctly (all blocks + correct transaction number from blocks)
func (hc *Client) CheckMissingTransactions(ctx context.Context, nv NetworkVersion, heightRange shared.HeightRange, window uint64) (missingBlocks, missingTransactions [][2]uint64, err error) {
	timer := metrics.NewTimer(callDurationCheckMissing)
	defer timer.ObserveDuration()

	blockContinuity, err := hc.storeEng.BlockContinuityCheck(ctx, shared.BlockWithMeta{Version: nv.Version, Network: nv.Network, ChainID: nv.ChainID}, heightRange.StartHeight, heightRange.EndHeight)
	if err != nil && err != params.ErrNotFound {
		return nil, nil, err
	}

	if len(blockContinuity) > 0 {
		missingBlocks = groupRanges(blockContinuity, window)
	}

	clockTransactionCheck, err := hc.storeEng.BlockTransactionCheck(ctx, shared.BlockWithMeta{Version: nv.Version, Network: nv.Network, ChainID: nv.ChainID}, heightRange.StartHeight, heightRange.EndHeight)
	if err != nil && err != params.ErrNotFound {
		return nil, nil, err
	}
	if len(clockTransactionCheck) > 0 {
		transactionRange := getRanges(clockTransactionCheck)
		missingTransactions = groupRanges(transactionRange, window)
	}

	return missingBlocks, missingTransactions, err
}

func (hc *Client) GetRunningTransactions(ctx context.Context) (run []Run, err error) {
	out := hc.runner.GetRunning()
	return out.Run, out.Err
}

// GetMissingTransactions gets missing transactions for givent height range using CheckMissingTransactions
func (hc *Client) GetMissingTransactions(ctx context.Context, nv NetworkVersion, heightRange shared.HeightRange, window uint64, async bool, force bool) (run *Run, err error) {

	if !async {
		return nil, hc.getMissingTransactions(ctx, nv, heightRange, window, nil)
	}

	hc.logger.Info("[Client] GetMissingTransactions StartProcess", zap.Any("range", heightRange), zap.Any("network", nv))

	isNew, progress, err := hc.runner.StartProcess(nv, heightRange, force)
	if err != nil {
		return nil, err
	}

	if !isNew {
		hc.logger.Info("[Client] Already Exists", zap.Any("range", heightRange), zap.Any("progress", progress))
		return progress, err
	}
	nCtx := progress.Ctx
	go hc.getMissingTransactions(nCtx, nv, heightRange, window, progress)

	hc.logger.Info("[Client] Returning Progress", zap.Any("range", heightRange), zap.Any("network", nv))
	<-time.After(time.Second)
	return progress, nil

}

func (hc *Client) getMissingTransactions(ctx context.Context, nv NetworkVersion, heightRange shared.HeightRange, window uint64, progress *Run) (err error) {

	timer := metrics.NewTimer(callDurationGetMissing)
	defer timer.ObserveDuration()

	hc.logger.Info("[Client] GetMissingTransactions CheckMissingTransactions", zap.Any("range", heightRange), zap.Any("network", nv))
	now := time.Now()
	missingBlocks, missingTransactions, err := hc.CheckMissingTransactions(ctx, nv, heightRange, 1000)
	if err != nil {
		if progress != nil {
			progress.Report(shared.HeightRange{}, time.Since(now), []error{err}, true)
		}
		hc.logger.Error("[Client] GetMissingTransactions CheckMissingTransactions error", zap.Error(err), zap.Any("range", heightRange), zap.Any("network", nv))
		return fmt.Errorf("Error checking missing transactions:  %w ", err)
	}

	for _, blocks := range missingBlocks {

		// (lukanus): has to be blocking op
		missingRange := shared.HeightRange{StartHeight: blocks[0], EndHeight: blocks[1]}
		now := time.Now()

		hc.logger.Info("[Client] GetMissingTransactions missingBlocks GetTransactions", zap.Any("range", missingRange), zap.Any("network", nv))
		_, err := hc.GetTransactions(ctx, nv, missingRange, 1000, true)
		if err != nil {
			if progress != nil {
				progress.Report(missingRange, time.Since(now), []error{err}, true)
			}
			hc.logger.Error("[Client] GetMissingTransactions missingBlocks GetTransactions error", zap.Error(err), zap.Any("range", missingRange), zap.Any("network", nv))
			return fmt.Errorf("error getting missing transactions from missing blocks:  %w ", err)
		}
		if progress != nil {
			hc.logger.Info("[Client] GetMissingTransactions missingBlocks GetTransactions Success", zap.Any("range", missingRange), zap.Any("network", nv))
			progress.Report(missingRange, time.Since(now), nil, false)
		}
	}

	if len(missingBlocks) > 0 {
		hc.logger.Info("[Client] GetMissingTransactions CheckMissingTransactions #2", zap.Any("range", heightRange), zap.Any("network", nv))
		now = time.Now()
		missingBlocks, missingTransactions, err = hc.CheckMissingTransactions(ctx, nv, heightRange, 1000)
		if err != nil {
			if progress != nil {
				progress.Report(shared.HeightRange{}, time.Since(now), []error{err}, true)
			}
			hc.logger.Error("[Client] GetMissingTransactions CheckMissingTransactions error #2", zap.Error(err), zap.Any("range", heightRange), zap.Any("network", nv))
			return fmt.Errorf("error checking missing transactions (rerun):  %w ", err)
		}
	}

	for _, transactions := range missingTransactions {
		// (lukanus): has to be blocking op

		missingRange := shared.HeightRange{StartHeight: transactions[0], EndHeight: transactions[1]}

		hc.logger.Info("[Client] GetMissingTransactions missingTransactions GetTransactions", zap.Any("range", missingRange), zap.Any("network", nv))
		now := time.Now()
		_, err := hc.GetTransactions(ctx, nv, shared.HeightRange{StartHeight: transactions[0], EndHeight: transactions[1]}, 1000, true)
		if err != nil {
			if progress != nil {
				progress.Report(missingRange, time.Since(now), []error{err}, true)
			}
			hc.logger.Error("[Client] GetMissingTransactions missingTransactions GetTransactions error", zap.Error(err), zap.Any("range", missingRange), zap.Any("network", nv))
			return fmt.Errorf("error getting missing transactions:  %w ", err)
		}

		if progress != nil {
			hc.logger.Info("[Client] GetMissingTransactions missingTransactions GetTransactions success", zap.Any("range", missingRange), zap.Any("network", nv))
			progress.Report(missingRange, time.Since(now), nil, false)
		}
	}

	if progress != nil {
		progress.Report(shared.HeightRange{}, 0, nil, true)
	}
	return
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
		SLICE_DIFF:
			for {
				next := current + window
				if next <= r[1] {
					pregroup = append(pregroup, [2]uint64{current, next})
					current = next + 1
					continue
				}

				pregroup = append(pregroup, [2]uint64{current, r[1]})
				break SLICE_DIFF
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
