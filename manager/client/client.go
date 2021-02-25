package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"time"

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
	GetRewards(ctx context.Context, nv NetworkVersion, start, end time.Time, account string) ([]shared.RewardSummary, error)
	GetAccountBalance(ctx context.Context, nv NetworkVersion, start, end time.Time, address string) error
}

type SchedulerContractor interface {
	ScrapeLatest(ctx context.Context, ldr shared.LatestDataRequest) (ldResp shared.LatestDataResponse, er error)
}

type ControllContractor interface {
	InsertTransactions(ctx context.Context, nv NetworkVersion, read io.ReadCloser) error

	CheckMissingTransactions(ctx context.Context, nv NetworkVersion, heightRange shared.HeightRange, mode MissingDiffType, window uint64) (missingBlocks, missingTransactions [][2]uint64, err error)
	GetMissingTransactions(ctx context.Context, nv NetworkVersion, heightRange shared.HeightRange, params GetMissingTxParams) (run *Run, err error)

	GetRunningTransactions(ctx context.Context) (run []Run, err error)
	StopRunningTransactions(ctx context.Context, nv NetworkVersion, clean bool) (err error)
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
	defer hc.recoverPanic()

	return hc.GetTransactions(ctx, nv, shared.HeightRange{Hash: id}, 1, false)
}

// GetTransactions gets transaction range and stores it in the database with respective blocks
func (hc *Client) GetTransactions(ctx context.Context, nv NetworkVersion, heightRange shared.HeightRange, batchLimit uint64, silent bool) ([]shared.Transaction, error) {
	defer hc.recoverPanic()

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
	defer hc.recoverPanic()

	timer := metrics.NewTimer(callDurationSearchTransactions)
	defer timer.ObserveDuration()

	return hc.storeEng.GetTransactions(ctx, params.TransactionSearch{
		Network:      ts.Network,
		ChainIDs:     ts.ChainIDs,
		Epoch:        ts.Epoch,
		Hash:         ts.Hash,
		Height:       ts.Height,
		Type:         params.SearchArr{Value: ts.Type},
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
		WithRawLog:   ts.WithRawLog,
	})
}

// InsertTransactions inserts external transactions batch
func (hc *Client) InsertTransactions(ctx context.Context, nv NetworkVersion, readr io.ReadCloser) error {
	defer hc.recoverPanic()

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
	defer hc.recoverPanic()

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

// GetRewards calulates reward summaries for 24h segments for given time range
func (hc *Client) GetRewards(ctx context.Context, nv NetworkVersion, start, end time.Time, account string) (rewards []shared.RewardSummary, err error) {
	defer hc.recoverPanic()

	timer := metrics.NewTimer(callDurationGetTransactions)
	defer timer.ObserveDuration()

	blockWithMeta := shared.BlockWithMeta{
		Network: nv.Network,
		Version: nv.Version,
		ChainID: nv.ChainID,
	}

	var prevDayEndHeight uint64
	var dayStart, dayEnd time.Time
	reqs := []structs.TaskRequest{}

	type dataRow struct {
		dayStart    time.Time
		startHeight uint64
		endHeight   uint64
	}
	var rows []dataRow

	bl, err := hc.storeEng.GetBlockForMinTime(ctx, blockWithMeta, start)
	if err != nil {
		return rewards, err
	}
	req, err := hc.createTaskRequest(ctx, nv, account, bl.Height-1, shared.ReqIDGetReward)
	if err != nil {
		return rewards, err
	}
	reqs = append(reqs, req)

	prevDayEndHeight = bl.Height - 1
	dayStart = start
	for {
		if dayStart == end {
			break
		}
		dayEnd = dayStart.Add(time.Hour * 24)
		if dayEnd.After(end) {
			dayEnd = end
		}

		bl, err := hc.storeEng.GetBlockForMinTime(ctx, blockWithMeta, dayEnd)

		if err != nil {
			return rewards, err
		}

		req, err := hc.createTaskRequest(ctx, nv, account, bl.Height, shared.ReqIDGetReward)
		if err != nil {
			return rewards, err
		}
		reqs = append(reqs, req)

		rows = append(rows, dataRow{
			dayStart:    dayStart,
			startHeight: prevDayEndHeight,
			endHeight:   bl.Height,
		})

		prevDayEndHeight = bl.Height
		dayStart = dayEnd
	}

	hc.logger.Info("[Client] Sending request data:", zap.Any("requests", reqs))
	respAwait, err := hc.sender.Send(reqs)
	if err != nil {
		hc.logger.Error("[Client] Error sending data", zap.Error(err))
		return rewards, fmt.Errorf("error sending data in GetRewards: %w", err)
	}

	defer respAwait.Close()

	buff := &bytes.Buffer{}
	dec := json.NewDecoder(buff)

	rewardsMap := make(map[uint64][]shared.TransactionAmount)

	var receivedFinals int
WaitForAllData:
	for {
		select {
		case <-ctx.Done():
			return rewards, errors.New("Request timed out")
		case response := <-respAwait.Resp:
			hc.logger.Debug("[Client] Get Reward received data:", zap.Any("type", response.Type), zap.Any("response", response))

			if response.Error.Msg != "" {
				return rewards, fmt.Errorf("error getting response: %s", response.Error.Msg)
			}

			if response.Type == "Reward" {
				buff.Reset()
				buff.ReadFrom(bytes.NewReader(response.Payload))
				b := &shared.GetRewardResponse{}
				err := dec.Decode(b)
				if err != nil {
					return rewards, fmt.Errorf("error decoding reward: %w", err)
				}
				rewardsMap[b.Height] = b.Rewards
			}

			if response.Final {
				receivedFinals++
			}

			if receivedFinals == len(reqs) {
				hc.logger.Info("[Client] Received All for", zap.Any("requests", reqs))
				break WaitForAllData
			}
		}
	}

	// reward earned = rewards balance at end of day - rewards balance at end of prev day + sum rewards from txs from prev end to end
	rewardTxTypes := []string{"delegate", "withdraw_delegator_reward", "begin_unbonding", "begin_redelegate"}
	for _, row := range rows {
		prevDayEndRewards := rewardsMap[row.startHeight]
		dayEndRewards := rewardsMap[row.endHeight]

		txs, err := hc.storeEng.GetTransactions(ctx, params.TransactionSearch{
			Network:      nv.Network,
			ChainIDs:     []string{nv.ChainID},
			Type:         params.SearchArr{Value: rewardTxTypes, Any: true},
			Account:      []string{account},
			AfterHeight:  row.startHeight,
			BeforeHeight: row.endHeight,
		})
		if err != nil {
			return rewards, err
		}

		totalCurrencyMap := make(map[string]shared.TransactionAmount)
		for _, total := range dayEndRewards {
			totalCurrencyMap[total.Currency] = total.Clone()
		}

		for _, tx := range txs {
			for _, ev := range tx.Events {
				for _, sub := range ev.Sub {
					evtrs, ok := sub.Transfers["reward"]
					if !ok {
						continue
					}
					for _, evtr := range evtrs {
						for _, amt := range evtr.Amounts {
							total, ok := totalCurrencyMap[amt.Currency]
							if !ok {
								totalCurrencyMap[amt.Currency] = amt.Clone()
								continue
							}
							err := total.Add(amt)
							if err != nil {
								return rewards, err
							}
						}
					}
				}
			}
		}

		for _, prev := range prevDayEndRewards {
			total, ok := totalCurrencyMap[prev.Currency]
			if !ok {
				totalCurrencyMap[prev.Currency] = prev.Clone()
				continue
			}
			err = total.Sub(prev)
			if err != nil {
				return rewards, err
			}
		}

		totals := []shared.TransactionAmount{}
		for _, amt := range totalCurrencyMap {
			totals = append(totals, amt)
		}
		rewards = append(rewards, shared.RewardSummary{
			Time:   row.dayStart,
			Start:  row.startHeight,
			End:    row.endHeight,
			Amount: totals,
		})
	}

	return rewards, nil
}

// GetAccountBalance calculates balance summaries for 24h segments for given time range
func (hc *Client) GetAccountBalance(ctx context.Context, nv NetworkVersion, start, end time.Time, account string) (rewards []shared.RewardSummary, err error) {
	defer hc.recoverPanic()

	timer := metrics.NewTimer(callDurationAccountBalance)
	defer timer.ObserveDuration()

	blockWithMeta := shared.BlockWithMeta{
		Network: nv.Network,
		Version: nv.Version,
		ChainID: nv.ChainID,
	}

	var prevDayEndHeight uint64
	var dayStart, dayEnd time.Time
	reqs := []structs.TaskRequest{}

	type dataRow struct {
		dayStart    time.Time
		startHeight uint64
		endHeight   uint64
	}
	var rows []dataRow

	bl, err := hc.storeEng.GetBlockForMinTime(ctx, blockWithMeta, start)
	if err != nil {
		return rewards, err
	}
	req, err := hc.createTaskRequest(ctx, nv, account, bl.Height-1, shared.ReqIDAccountBalance)
	if err != nil {
		return rewards, err
	}
	reqs = append(reqs, req)

	prevDayEndHeight = bl.Height - 1
	dayStart = start
	for {
		if dayStart == end {
			break
		}
		dayEnd = dayStart.Add(time.Hour * 24)
		if dayEnd.After(end) {
			dayEnd = end
		}

		bl, err := hc.storeEng.GetBlockForMinTime(ctx, blockWithMeta, dayEnd)

		if err != nil {
			return rewards, err
		}

		req, err := hc.createTaskRequest(ctx, nv, account, bl.Height, shared.ReqIDAccountBalance)
		if err != nil {
			return rewards, err
		}
		reqs = append(reqs, req)

		rows = append(rows, dataRow{
			dayStart:    dayStart,
			startHeight: prevDayEndHeight,
			endHeight:   bl.Height,
		})

		prevDayEndHeight = bl.Height
		dayStart = dayEnd
	}

	hc.logger.Info("[Client] Sending request data:", zap.Any("requests", reqs))
	respAwait, err := hc.sender.Send(reqs)
	if err != nil {
		hc.logger.Error("[Client] Error sending data", zap.Error(err))
		return rewards, fmt.Errorf("error sending data in GetAccountBalance: %w", err)
	}

	defer respAwait.Close()

	buff := &bytes.Buffer{}
	dec := json.NewDecoder(buff)

	rewardsMap := make(map[uint64][]shared.TransactionAmount)

	var receivedFinals int
WaitForAllData:
	for {
		select {
		case <-ctx.Done():
			return rewards, errors.New("Request timed out")
		case response := <-respAwait.Resp:
			hc.logger.Debug("[Client] Get Reward received data:", zap.Any("type", response.Type), zap.Any("response", response))

			if response.Error.Msg != "" {
				return rewards, fmt.Errorf("error getting response: %s", response.Error.Msg)
			}

			if response.Type == "Reward" {
				buff.Reset()
				buff.ReadFrom(bytes.NewReader(response.Payload))
				b := &shared.GetRewardResponse{}
				err := dec.Decode(b)
				if err != nil {
					return rewards, fmt.Errorf("error decoding reward: %w", err)
				}
				rewardsMap[b.Height] = b.Rewards
			}

			if response.Final {
				receivedFinals++
			}

			if receivedFinals == len(reqs) {
				hc.logger.Info("[Client] Received All for", zap.Any("requests", reqs))
				break WaitForAllData
			}
		}
	}

	/*
		// reward earned = rewards balance at end of day - rewards balance at end of prev day + sum rewards from txs from prev end to end
		rewardTxTypes := []string{"delegate", "withdraw_delegator_reward", "begin_unbonding", "begin_redelegate"}
		for _, row := range rows {
			prevDayEndRewards := rewardsMap[row.startHeight]
			dayEndRewards := rewardsMap[row.endHeight]

			txs, err := hc.storeEng.GetTransactions(ctx, params.TransactionSearch{
				Network:      nv.Network,
				ChainIDs:     []string{nv.ChainID},
				Type:         params.SearchArr{Value: rewardTxTypes, Any: true},
				Account:      []string{account},
				AfterHeight:  row.startHeight,
				BeforeHeight: row.endHeight,
			})
			if err != nil {
				return rewards, err
			}

			totalCurrencyMap := make(map[string]shared.TransactionAmount)
			for _, total := range dayEndRewards {
				totalCurrencyMap[total.Currency] = total.Clone()
			}

			for _, tx := range txs {
				for _, ev := range tx.Events {
					for _, sub := range ev.Sub {
						evtrs, ok := sub.Transfers["reward"]
						if !ok {
							continue
						}
						for _, evtr := range evtrs {
							for _, amt := range evtr.Amounts {
								total, ok := totalCurrencyMap[amt.Currency]
								if !ok {
									totalCurrencyMap[amt.Currency] = amt.Clone()
									continue
								}
								err := total.Add(amt)
								if err != nil {
									return rewards, err
								}
							}
						}
					}
				}
			}

			for _, prev := range prevDayEndRewards {
				total, ok := totalCurrencyMap[prev.Currency]
				if !ok {
					totalCurrencyMap[prev.Currency] = prev.Clone()
					continue
				}
				err = total.Sub(prev)
				if err != nil {
					return rewards, err
				}
			}

			totals := []shared.TransactionAmount{}
			for _, amt := range totalCurrencyMap {
				totals = append(totals, amt)
			}
			rewards = append(rewards, shared.RewardSummary{
				Time:   row.dayStart,
				Start:  row.startHeight,
				End:    row.endHeight,
				Amount: totals,
			})
		}
	*/

	return rewards, nil
}

func (hc *Client) createTaskRequest(ctx context.Context, nv NetworkVersion, account string, height uint64, reqType string) (structs.TaskRequest, error) {
	ha, err := json.Marshal(shared.HeightAccount{
		Height:  height,
		Account: account,
		Network: nv.Network,
		ChainID: nv.ChainID,
	})
	if err != nil {
		return structs.TaskRequest{}, err
	}

	return structs.TaskRequest{
		Network: nv.Network,
		ChainID: nv.ChainID,
		Version: nv.Version,
		Type:    reqType,
		Payload: ha,
	}, nil
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

func (hc *Client) recoverPanic() {
	if p := recover(); p != nil {
		hc.logger.Error("[Client] Panic ", zap.Any("contents", p))
		hc.logger.Sync()
	}
}
