package cosmos

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
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
	log "github.com/sirupsen/logrus"
)

type TxLogError struct {
	Codespace string  `json:"codespace"`
	Code      float64 `json:"code"`
	Message   string  `json:"message"`
}

var curencyRegex = regexp.MustCompile("([0-9\\.\\,\\-\\s]+)([^0-9\\s]+)$")

/*
// SearchTx is making search api call
func (c *Client) SearchTx(ctx context.Context, wg *sync.WaitGroup, r structs.HeightRange, blocks map[uint64]shared.Block, out chan cStruct.OutResp, page, perPage int, fin chan string) (count int64, err error) {
	if wg != nil {
		defer wg.Done()
	}
	defer c.logger.Sync()

	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/tx_search", nil)
	if err != nil {
		if fin != nil {
			fin <- err.Error()
		}
		return 0, err
	}

	req.Header.Add("Content-Type", "application/json")
	if c.key != "" {
		req.Header.Add("Authorization", c.key)
	}

	q := req.URL.Query()

	s := strings.Builder{}

	s.WriteString(`"`)
	s.WriteString("tx.height>= ")
	s.WriteString(strconv.FormatUint(r.StartHeight, 10))

	if r.EndHeight > 0 && r.EndHeight != r.StartHeight {
		s.WriteString(" AND ")
		s.WriteString("tx.height<=")
		s.WriteString(strconv.FormatUint(r.EndHeight, 10))
	}
	s.WriteString(`"`)

	q.Add("query", s.String())
	q.Add("page", strconv.Itoa(page))
	q.Add("per_page", strconv.Itoa(perPage))
	req.URL.RawQuery = q.Encode()

	// (lukanus): do not block initial calls
	if r.EndHeight != 0 && r.StartHeight != 0 {
		err = c.rateLimitter.Wait(ctx)
		if err != nil {
			if fin != nil {
				fin <- err.Error()
			}
			return 0, err
		}
	}

	now := time.Now()
	resp, err := c.httpClient.Do(req)

	log.Debug("[COSMOS-API] Request Time (/tx_search)", zap.Duration("duration", time.Now().Sub(now)))
	if err != nil {
		if fin != nil {
			fin <- err.Error()
		}
		return 0, err
	}

	if resp.StatusCode > 399 { // ERROR
		serverError, _ := ioutil.ReadAll(resp.Body)

		c.logger.Error("[COSMOS-API] error getting response from server", zap.Int("code", resp.StatusCode), zap.Any("response", string(serverError)))
		err := fmt.Errorf("error getting response from server %d %s", resp.StatusCode, string(serverError))
		if fin != nil {
			fin <- err.Error()
		}
		return 0, err
	}

	rawRequestDuration.WithLabels("/tx_search", resp.Status).Observe(time.Since(now).Seconds())

	decoder := json.NewDecoder(resp.Body)

	result := &GetTxSearchResponse{}
	if err = decoder.Decode(result); err != nil {
		c.logger.Error("[COSMOS-API] unable to decode result body", zap.Error(err))
		err := fmt.Errorf("unable to decode result body %w", err)
		if fin != nil {
			fin <- err.Error()
		}
		return 0, err
	}

	if result.Error.Message != "" {
		c.logger.Error("[COSMOS-API] Error getting search", zap.Any("result", result.Error.Message))
		err := fmt.Errorf("Error getting search: %s", result.Error.Message)
		if fin != nil {
			fin <- err.Error()
		}
		return 0, err
	}

	totalCount, err := strconv.ParseInt(result.Result.TotalCount, 10, 64)
	if err != nil {
		c.logger.Error("[COSMOS-API] Error getting totalCount", zap.Error(err), zap.Any("result", result), zap.String("query", req.URL.RawQuery), zap.Any("request", r))
		if fin != nil {
			fin <- err.Error()
		}
		return 0, err
	}

	numberOfItemsTransactions.Observe(float64(totalCount))

	c.logger.Debug("[COSMOS-API] Converting requests ", zap.Int("number", len(result.Result.Txs)), zap.Int("blocks", len(blocks)))
	err = rawToTransaction(ctx, c, result.Result.Txs, blocks, out, c.logger, c.cdc)
	if err != nil {
		c.logger.Error("[COSMOS-API] Error getting rawToTransaction", zap.Error(err))
		if fin != nil {
			fin <- err.Error()
		}
	}
	c.logger.Debug("[COSMOS-API] Converted all requests ")

	if fin != nil {
		fin <- ""
	}
	return totalCount, nil
}
*/

// SearchTx is making search api call
func (c *Client) SearchTx(ctx context.Context, r structs.HeightRange, blocks map[uint64]shared.Block, out chan cStruct.OutResp, page, perPage int, fin chan string) {
	defer c.logger.Sync()

	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/tx_search", nil)
	if err != nil {
		fin <- err.Error()
		return
	}

	req.Header.Add("Content-Type", "application/json")
	if c.key != "" {
		req.Header.Add("Authorization", c.key)
	}

	q := req.URL.Query()

	s := strings.Builder{}

	s.WriteString(`"`)
	s.WriteString("tx.height>= ")
	s.WriteString(strconv.FormatUint(r.StartHeight, 10))

	if r.EndHeight > 0 && r.EndHeight != r.StartHeight {
		s.WriteString(" AND ")
		s.WriteString("tx.height<=")
		s.WriteString(strconv.FormatUint(r.EndHeight, 10))
	}
	s.WriteString(`"`)

	q.Add("query", s.String())
	q.Add("page", strconv.Itoa(page))
	q.Add("per_page", strconv.Itoa(perPage))
	req.URL.RawQuery = q.Encode()

	// (lukanus): do not block initial calls
	if r.EndHeight != 0 && r.StartHeight != 0 {
		err = c.rateLimitter.Wait(ctx)
		if err != nil {
			fin <- err.Error()
			return
		}
	}

	now := time.Now()
	resp, err := c.httpClient.Do(req)

	log.Debug("[COSMOS-API] Request Time (/tx_search)", zap.Duration("duration", time.Now().Sub(now)))
	if err != nil {
		fin <- err.Error()
		return
	}

	if resp.StatusCode > 399 { // ERROR
		serverError, _ := ioutil.ReadAll(resp.Body)

		c.logger.Error("[COSMOS-API] error getting response from server", zap.Int("code", resp.StatusCode), zap.Any("response", string(serverError)))
		err := fmt.Errorf("error getting response from server %d %s", resp.StatusCode, string(serverError))
		fin <- err.Error()
		return
	}

	rawRequestDuration.WithLabels("/tx_search", resp.Status).Observe(time.Since(now).Seconds())

	decoder := json.NewDecoder(resp.Body)

	result := &GetTxSearchResponse{}
	if err = decoder.Decode(result); err != nil {
		c.logger.Error("[COSMOS-API] unable to decode result body", zap.Error(err))
		err := fmt.Errorf("unable to decode result body %w", err)
		fin <- err.Error()
		return
	}

	if result.Error.Message != "" {
		c.logger.Error("[COSMOS-API] Error getting search", zap.Any("result", result.Error.Message))
		err := fmt.Errorf("Error getting search: %s", result.Error.Message)
		fin <- err.Error()
		return
	}

	totalCount, err := strconv.ParseInt(result.Result.TotalCount, 10, 64)
	if err != nil {
		c.logger.Error("[COSMOS-API] Error getting totalCount", zap.Error(err), zap.Any("result", result), zap.String("query", req.URL.RawQuery), zap.Any("request", r))
		fin <- err.Error()
		return
	}

	numberOfItemsTransactions.Observe(float64(totalCount))

	c.logger.Debug("[COSMOS-API] Converting requests ", zap.Int("number", len(result.Result.Txs)), zap.Int("blocks", len(blocks)))
	err = rawToTransaction(ctx, c, result.Result.Txs, blocks, out, c.logger, c.cdc)
	if err != nil {
		c.logger.Error("[COSMOS-API] Error getting rawToTransaction", zap.Error(err))
		fin <- err.Error()
	}
	c.logger.Debug("[COSMOS-API] Converted all requests ")

	fin <- ""
	return
}

func rawToTransaction(ctx context.Context, c *Client, in []TxResponse, blocks map[uint64]shared.Block, out chan cStruct.OutResp, logger *zap.Logger, cdc *codec.Codec) error {
	readr := strings.NewReader("")
	dec := json.NewDecoder(readr)
	for _, txRaw := range in {
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

		outTX := cStruct.OutResp{Type: "Transaction"}
		block := blocks[hInt]

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

					if len(attr.Withdraw) > 0 {
						sub.Withdraw = attr.Withdraw
					}

					if len(attr.Validator) > 0 {
						sub.Validator = attr.Validator
					}

					if len(attr.Validator) > 0 {
						for k, v := range attr.Others {
							logger.Info("[COSMOS-API] Found unknown event attribute ", zap.String("key", k), zap.Strings("values", v))
						}
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
		timer.ObserveDuration()
	}

	return nil
}

func getCurrency(in string) []string {
	return curencyRegex.FindAllString(in, 2)
}
