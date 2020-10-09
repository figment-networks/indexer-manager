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
	missingBlocks, missingTransactions, err := hc.CheckMissingTransactions(ctx, nv, heightRange, 999)
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
			hc.logger.Info("[Client] GetMissingTransactions missingBlocks GetTransactions success", zap.Any("range", missingRange), zap.Any("network", nv))
			progress.Report(missingRange, time.Since(now), nil, false)
		}
	}

	if len(missingBlocks) > 0 {
		hc.logger.Info("[Client] GetMissingTransactions CheckMissingTransactions #2", zap.Any("range", heightRange), zap.Any("network", nv))
		now = time.Now()
		missingBlocks, missingTransactions, err = hc.CheckMissingTransactions(ctx, nv, heightRange, 999)
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
