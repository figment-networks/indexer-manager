package client

import (
	"context"
	"fmt"
	"time"

	"github.com/figment-networks/indexer-manager/manager/store/params"
	shared "github.com/figment-networks/indexer-manager/structs"
	"github.com/figment-networks/indexing-engine/metrics"
	"go.uber.org/zap"
)

type MissingDiffType string

const (
	MissingDiffTypeSQLJoin   = "MissingDiffTypeSQLJoin"
	MissingDiffTypeSQLHybrid = "MissingDiffTypeSQLHybrid"
)

const (
	SQLHybridRangeLimit = 100000
)

// CheckMissingTransactions checks consistency of database if every transaction is written correctly (all blocks + correct transaction number from blocks)
func (hc *Client) CheckMissingTransactions(ctx context.Context, nv NetworkVersion, heightRange shared.HeightRange, mode MissingDiffType, window uint64) (missingBlocks, missingTransactions [][2]uint64, err error) {
	timer := metrics.NewTimer(callDurationCheckMissing)
	defer timer.ObserveDuration()

	if mode == MissingDiffTypeSQLHybrid {
		diff := heightRange.EndHeight - heightRange.StartHeight
		if diff < SQLHybridRangeLimit {
			return hc.checkConsistency(ctx, nv, heightRange, window)
		}

		//  run SQLHybridRangeLimit batches in parallel
		out := make(chan consistencyResp, 40)
		defer close(out)
		var i uint64
		for {
			startH := heightRange.StartHeight + i*uint64(SQLHybridRangeLimit)
			endH := heightRange.StartHeight + i*uint64(SQLHybridRangeLimit) + uint64(SQLHybridRangeLimit) - 1

			if heightRange.EndHeight > 0 && endH > heightRange.EndHeight {
				endH = heightRange.EndHeight
			}
			go hc.checkConsistencyAsync(ctx, nv, shared.HeightRange{StartHeight: startH, EndHeight: endH}, window, out, int(i))
			i++
			if heightRange.EndHeight == endH || heightRange.EndHeight == 0 {
				break
			}
		}

		var received uint64
		// (lukanus): slices are pointer types anyway :)
		allRespBlocks := make([][][2]uint64, i)
		allRespTransactions := make([][][2]uint64, i)

		var errs []error
		for sum := range out {
			received++

			if sum.Err != nil {
				errs = append(errs, sum.Err)
			}

			if len(sum.MissingBlocks) > 0 {
				allRespBlocks[sum.Ident] = sum.MissingBlocks
			}

			if len(sum.MissingTransactions) > 0 {
				allRespTransactions[sum.Ident] = sum.MissingTransactions
			}

			if i == received {
				break
			}
		}

		if len(errs) > 0 {
			err = fmt.Errorf("CheckMissingTransactions errors")
		}

		for i := range allRespBlocks {
			missingBlocks = append(missingBlocks, allRespBlocks[i]...)
		}

		for i := range allRespTransactions {
			missingTransactions = append(missingTransactions, allRespTransactions[i]...)
		}
		return missingBlocks, missingTransactions, err

	}

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

type consistencyResp struct {
	MissingBlocks       [][2]uint64
	MissingTransactions [][2]uint64
	Err                 error
	Ident               int
}

func (hc *Client) checkConsistencyAsync(ctx context.Context, nv NetworkVersion, heightRange shared.HeightRange, window uint64, out chan<- consistencyResp, ident int) {
	cR := consistencyResp{Ident: ident}
	cR.MissingBlocks, cR.MissingTransactions, cR.Err = hc.checkConsistency(ctx, nv, heightRange, window)
	out <- cR
}

func (hc *Client) checkConsistency(ctx context.Context, nv NetworkVersion, heightRange shared.HeightRange, window uint64) (missingBlocks, missingTransactions [][2]uint64, err error) {
	hc.logger.Debug("[CLIENT] Checks consistency for ", zap.Uint64("start", heightRange.StartHeight), zap.Uint64("end", heightRange.EndHeight))

	start := time.Now()
	blocks, err := hc.storeEng.GetBlocksHeightsWithNumTx(ctx, shared.BlockWithMeta{Version: nv.Version, Network: nv.Network, ChainID: nv.ChainID}, heightRange.StartHeight, heightRange.EndHeight)
	if err != nil && err != params.ErrNotFound {
		return nil, nil, err
	}

	if len(blocks) == 0 {
		missingBlocks = groupRanges([][2]uint64{{heightRange.StartHeight, heightRange.EndHeight}}, window)
		hc.logger.Debug("[CLIENT] No blocks for ", zap.Uint64("start", heightRange.StartHeight), zap.Uint64("end", heightRange.EndHeight), zap.Duration("took", time.Since(start)))
		return missingBlocks, missingTransactions, nil
	}

	txs, err := hc.storeEng.GetTransactionsHeightsWithTxCount(ctx, shared.BlockWithMeta{Version: nv.Version, Network: nv.Network, ChainID: nv.ChainID}, heightRange.StartHeight, heightRange.EndHeight)
	if err != nil && err != params.ErrNotFound {
		return nil, nil, err
	}
	database := time.Since(start)
	hc.logger.Debug("[CLIENT] Got consistency data for ", zap.Uint64("start", heightRange.StartHeight), zap.Uint64("end", heightRange.EndHeight), zap.Duration("took", database))

	b, t := processMissingBlocksAndTransactions(blocks, txs, heightRange.StartHeight, heightRange.EndHeight, window, hc.logger)
	hc.logger.Info("[Client] consistency check succeeded", zap.Uint64("start", heightRange.StartHeight), zap.Uint64("end", heightRange.EndHeight), zap.Duration("database", database), zap.Duration("all", time.Since(start)), zap.Int("missing_blocks_ranges", len(b)), zap.Int("missing_transactions_ranges", len(b)))
	return b, t, err
}

// processMissingBlocksAndTransactions takes blocks and transactions and joins them seeking for missing pieces
func processMissingBlocksAndTransactions(blocks, txs [][2]uint64, startHeight, endHeight, window uint64, logger *zap.Logger) (missingBlocks, missingTransactions [][2]uint64) {

	missingBlocks = [][2]uint64{}
	missingTransactions = [][2]uint64{}

	// initial blocks
	firstBlock := blocks[0]
	if firstBlock[0] > startHeight {
		missingBlocks = append(missingBlocks, [2]uint64{startHeight, firstBlock[0] - 1})
	}

	lenTx := len(txs)

	var prevBlockHeight uint64
	var txIndex int
	if startHeight != 0 {
		prevBlockHeight = startHeight - 1
	}

	for _, block := range blocks {
		// add missing blocks rage
		if block[0] != prevBlockHeight+1 {
			missingBlocks = append(missingBlocks, [2]uint64{prevBlockHeight + 1, block[0] - 1})
			prevBlockHeight = block[0]
			continue
		}
		prevBlockHeight = block[0]

		if txIndex+1 > lenTx {
			break
		}

		// check if current transaction == block
		tx := txs[txIndex]
		var missed bool
		if tx[0] > block[0] { // for non existing tx
			if block[1] > 0 {
				logger.Info("for non existing tx", zap.Any("tx", tx), zap.Any("block", block))
				missingTransactions = append(missingTransactions, [2]uint64{block[0], block[0]})
			}
			continue
		} else if tx[0] < block[0] {
			//  search for next equal height
			for {
				if txIndex > lenTx {
					missed = true
					break
				}

				tx = txs[txIndex]
				if tx[0] == block[0] {
					// success
					break
				} else if tx[0] > block[0] {
					missed = true
					break
				}

				txIndex++
			}
			txIndex++
		} else {
			txIndex++
		}

		if missed {
			// something goes wrong
			continue
		}

		if tx[0] != block[0] && tx[1] != block[1] {
			logger.Info("end", zap.Any("tx_index", txIndex), zap.Any("tx", tx), zap.Any("block", block))
			missingTransactions = append(missingTransactions, [2]uint64{block[0], block[0]})
		}

	}

	lenBlocks := len(blocks)
	lastBlock := blocks[lenBlocks-1]
	if lastBlock[0] != endHeight {
		missingBlocks = append(missingBlocks, [2]uint64{lastBlock[0] + 1, endHeight})
	}

	return groupRanges(missingBlocks, window), groupRanges(missingTransactions, window)
}

// GetRunningTransactions gets running transactions
func (hc *Client) GetRunningTransactions(ctx context.Context) (run []Run, err error) {
	out := hc.runner.GetRunning()
	return out.Run, out.Err
}

// GetMissingTransactions gets missing transactions for given height range using CheckMissingTransactions.
// This may run very very long (aka synchronize entire chain). For that kind of operations async parameter got added and runner was created.
func (hc *Client) GetMissingTransactions(ctx context.Context, nv NetworkVersion, heightRange shared.HeightRange, window uint64, async bool, force bool) (run *Run, err error) {

	hc.logger.Info("[Client] GetMissingTransactions StartProcess", zap.Any("range", heightRange), zap.Any("network", nv))

	isNew, progress, err := hc.runner.StartProcess(nv, heightRange, force)
	if err != nil {
		return nil, err
	}

	if !isNew {
		hc.logger.Info("[Client] Already Exists", zap.Any("range", heightRange), zap.Any("progress", progress))
		return progress, err
	}

	if !async {
		err := hc.getMissingTransactions(ctx, nv, heightRange, window, nil)
		return nil, err
	}

	nCtx := progress.Ctx
	go hc.getMissingTransactions(nCtx, nv, heightRange, window, progress)

	hc.logger.Info("[Client] Returning Progress", zap.Any("range", heightRange), zap.Any("network", nv))
	<-time.After(time.Second)
	return progress, nil

}

// getMissingTransactions scrape missing transactions based on height
func (hc *Client) getMissingTransactions(ctx context.Context, nv NetworkVersion, heightRange shared.HeightRange, window uint64, progress *Run) (err error) {
	timer := metrics.NewTimer(callDurationGetMissing)
	defer timer.ObserveDuration()

	hc.logger.Info("[Client] GetMissingTransactions CheckMissingTransactions", zap.Any("range", heightRange), zap.Any("network", nv))
	now := time.Now()
	missingBlocks, missingTransactions, err := hc.CheckMissingTransactions(ctx, nv, heightRange, MissingDiffTypeSQLHybrid, 999)
	if err != nil {
		if progress != nil {
			progress.Report(shared.HeightRange{}, time.Since(now), []error{err}, true)
		}
		hc.logger.Error("[Client] GetMissingTransactions CheckMissingTransactions error", zap.Error(err), zap.Any("range", heightRange), zap.Any("network", nv))
		return fmt.Errorf("Error checking missing transactions:  %w ", err)
	}

	for _, blocks := range missingBlocks {

		// (lukanus): has to be blocking op
		missingRange := shared.HeightRange{
			Network:     nv.Network,
			ChainID:     nv.ChainID,
			StartHeight: blocks[0],
			EndHeight:   blocks[1],
		}
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
			hc.logger.Info("[Client] GetMissingTransactions missingBlocks GetTransactions success", zap.Any("range", missingRange), zap.Any("network", nv))
			progress.Report(missingRange, time.Since(now), nil, false)
		}
	}

	if len(missingBlocks) > 0 {
		hc.logger.Info("[Client] GetMissingTransactions CheckMissingTransactions #2", zap.Any("range", heightRange), zap.Any("network", nv))
		now = time.Now()
		missingBlocks, missingTransactions, err = hc.CheckMissingTransactions(ctx, nv, heightRange, MissingDiffTypeSQLHybrid, 999)
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

		missingRange := shared.HeightRange{
			Network:     nv.Network,
			ChainID:     nv.ChainID,
			StartHeight: transactions[0],
			EndHeight:   transactions[1]}

		hc.logger.Info("[Client] GetMissingTransactions missingTransactions GetTransactions", zap.Any("range", missingRange), zap.Any("network", nv))
		now := time.Now()
		_, err := hc.GetTransactions(ctx, nv, shared.HeightRange{
			Network:     nv.Network,
			ChainID:     nv.ChainID,
			StartHeight: transactions[0],
			EndHeight:   transactions[1]}, 1000, true)
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
