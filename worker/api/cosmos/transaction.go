package cosmos

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/figment-networks/cosmos-indexer/structs"
	shared "github.com/figment-networks/cosmos-indexer/structs"
	cStruct "github.com/figment-networks/cosmos-indexer/worker/connectivity/structs"
	"github.com/figment-networks/indexing-engine/metrics"
	"go.uber.org/zap"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

type TxLogError struct {
	Codespace string  `json:"codespace"`
	Code      float64 `json:"code"`
	Message   string  `json:"message"`
}

var curencyRegex = regexp.MustCompile("([0-9\\.\\,\\-\\s]+)([^0-9\\s]+)$")

// SearchTx is making search api call
func (c *Client) SearchTx(ctx context.Context, taskID, runUUID uuid.UUID, r structs.HeightRange, page, perPage int, fin chan string) (count int64, err error) {
	defer c.logger.Sync()

	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/tx_search", nil)
	if err != nil {
		fin <- err.Error()
	}

	req.Header.Add("Content-Type", "application/json")
	if c.key != "" {
		req.Header.Add("Authorization", c.key)
	}

	q := req.URL.Query()

	s := strings.Builder{}

	s.WriteString(`"`)
	s.WriteString("tx.height>= ")
	s.WriteString(strconv.Itoa(int(r.StartHeight)))

	if r.EndHeight > 0 && r.EndHeight != r.StartHeight {
		s.WriteString(" AND ")
		s.WriteString("tx.height<=")
		s.WriteString(strconv.Itoa(int(r.EndHeight)))
	}
	s.WriteString(`"`)

	q.Add("query", s.String())
	q.Add("page", strconv.Itoa(page))
	q.Add("per_page", strconv.Itoa(perPage))
	req.URL.RawQuery = q.Encode()

	now := time.Now()
	resp, err := c.httpClient.Do(req)

	log.Debug("[COSMOS-API] Request Time (/tx_search)", zap.Duration("duration", time.Now().Sub(now)))
	if err != nil {
		fin <- err.Error()
		return 0, err
	}
	rawRequestDuration.WithLabels("/tx_search", resp.Status).Observe(time.Since(now).Seconds())

	decoder := json.NewDecoder(resp.Body)

	result := &GetTxSearchResponse{}
	if err = decoder.Decode(result); err != nil {
		c.logger.Error("[COSMOS-API] unable to decode result body", zap.Error(err))
		err := fmt.Errorf("unable to decode result body %w", err)
		fin <- err.Error()
		return 0, err
	}

	if result.Error.Message != "" {
		//err := fmt.Sprintf("Error getting search: %s", result.Error.Message)
		err := fmt.Errorf("Error getting search: %s", result.Error.Message)
		c.logger.Error("[COSMOS-API] Error getting search", zap.Error(err))
		fin <- err.Error()
		return 0, err
	}

	totalCount, err := strconv.ParseInt(result.Result.TotalCount, 10, 64)
	if err != nil {
		c.logger.Error("[COSMOS-API] Error getting totalCount", zap.Error(err))
		fin <- err.Error()
		return 0, err
	}

	numberOfItemsTransactions.Observe(float64(totalCount))

	for _, tx := range result.Result.Txs {

		select {
		case <-ctx.Done():
			return totalCount, nil
		default:
		}

		tx.All = totalCount
		tx.TaskID.TaskID = taskID
		tx.TaskID.RunID = runUUID
		c.inTx <- tx
	}
	fin <- ""
	return totalCount, nil
}

func rawToTransaction(ctx context.Context, c *Client, in chan TxResponse, out chan cStruct.OutResp, logger *zap.Logger, cdc *codec.Codec) {
	readr := strings.NewReader("")
	dec := json.NewDecoder(readr)
	for txRaw := range in {
		timer := metrics.NewTimer(transactionConversionDuration)

		tx := &auth.StdTx{}

		readr.Reset(txRaw.TxResult.Log)
		lf := []LogFormat{}
		txErr := TxLogError{}
		err := dec.Decode(&lf)
		if err != nil {
			// (lukanus): Try to fallback to known error format
			readr.Reset(txRaw.TxResult.Log)
			errin := dec.Decode(&txErr)
			if errin != nil {
				logger.Error("[COSMOS-API] Problem decoding raw transaction (json)", zap.Error(err), zap.String("content_log", txRaw.TxResult.Log), zap.Any("content", txRaw))

				out <- cStruct.OutResp{
					ID:    txRaw.TaskID.TaskID,
					RunID: txRaw.TaskID.RunID,
					Error: fmt.Errorf("[COSMOS-API] Problem decoding raw transaction (json) %w", err),
				}
				continue
			}
		}

		base64Dec := base64.NewDecoder(base64.StdEncoding, strings.NewReader(txRaw.TxData))
		_, err = cdc.UnmarshalBinaryLengthPrefixedReader(base64Dec, tx, 0)
		if err != nil {
			logger.Error("[COSMOS-API] Problem decoding raw transaction (cdc) ", zap.Error(err))
		}
		hInt, err := strconv.ParseUint(txRaw.Height, 10, 64)
		if err != nil {
			logger.Error("[COSMOS-API] Problem parsing height", zap.Error(err))
		}

		// (lukanus): it has embedded cache in int
		block, err := c.GetBlock(ctx, shared.HeightHash{Height: hInt})
		if err != nil {
			logger.Error("[COSMOS-API] Problem getting block at height", zap.Uint64("height", hInt), zap.Error(err))
		}

		outTX := cStruct.OutResp{
			ID:    txRaw.TaskID.TaskID,
			RunID: txRaw.TaskID.RunID,
			All:   uint64(txRaw.All),
			Type:  "Transaction",
		}

		trans := shared.Transaction{
			Hash:      txRaw.Hash,
			Memo:      tx.GetMemo(),
			Time:      block.Time,
			BlockHash: block.Hash,
		}

		trans.Height, err = strconv.ParseUint(txRaw.Height, 10, 64)
		if err != nil {
			outTX.Error = err
		}
		trans.GasWanted, err = strconv.ParseUint(txRaw.TxResult.GasWanted, 10, 64)
		if err != nil {
			outTX.Error = err
		}
		trans.GasUsed, err = strconv.ParseUint(txRaw.TxResult.GasUsed, 10, 64)
		if err != nil {
			outTX.Error = err
		}

		for _, logf := range lf {
			tev := shared.TransactionEvent{
				ID: strconv.FormatFloat(logf.MsgIndex, 'f', -1, 64),
			}
			for _, ev := range logf.Events {
				sub := shared.SubsetEvent{
					Type: ev.Type,
				}
				for _, attr := range ev.Attributes {
					sub.Module = attr.Module
					sub.Action = attr.Action

					if len(attr.Sender) > 0 {
						sub.Sender = attr.Sender
					}

					if len(attr.Recipient) > 0 {
						sub.Recipient = attr.Recipient
					}

					if len(attr.Validator) > 0 {
						sub.Validator = attr.Validator
					}
					if attr.CompletionTime != "" {
						cTime, _ := time.Parse(time.RFC3339Nano, attr.CompletionTime)
						sub.Completion = &cTime
					}

					if attr.Amount != "" {
						sliced := getCurrency(attr.Amount)

						sub.Amount = &shared.TransactionAmount{
							Text: attr.Amount,
						}

						if len(sliced) == 2 {
							sub.Amount.Currency = sliced[1]
							sub.Amount.Numeric, _ = strconv.ParseFloat(sliced[0], 64)
						} else {
							sub.Amount.Numeric, _ = strconv.ParseFloat(attr.Amount, 64)
						}
					}
				}
				tev.Sub = append(tev.Sub, sub)
			}
			trans.Events = append(trans.Events, tev)
		}

		if txErr.Message != "" {
			tev := shared.TransactionEvent{
				Sub: []shared.SubsetEvent{{
					Module: txErr.Codespace,
					Error:  &shared.SubsetEventError{Message: txErr.Message},
				}},
			}
			trans.Events = append(trans.Events, tev)
		}

		outTX.Payload = trans
		out <- outTX
		out <- cStruct.OutResp{
			ID:         txRaw.TaskID.TaskID,
			RunID:      txRaw.TaskID.RunID,
			Additional: true,
			Type:       "Block",
			Payload:    block,
		}

		timer.ObserveDuration()
	}
}

func getCurrency(in string) []string {
	return curencyRegex.FindAllString(in, 2)
}
