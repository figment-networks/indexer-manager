package terra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/figment-networks/cosmos-indexer/structs"
	shared "github.com/figment-networks/cosmos-indexer/structs"
	cStruct "github.com/figment-networks/cosmos-indexer/worker/connectivity/structs"
	"go.uber.org/zap"

	log "github.com/sirupsen/logrus"
)

var curencyRegex = regexp.MustCompile("([0-9\\.\\,\\-\\s]+)([^0-9\\s]+)$")

/*


// SearchTx is making search api call
func (c *Client) SearchTx(ctx context.Context, wg *sync.WaitGroup, r structs.HeightRange, out chan cStruct.OutResp, page, perPage int, fin chan string) (count int64, err error) {
	defer wg.Done()
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

	rawToTransaction(ctx, c, result.Result.Txs, out, c.logger, c.cdc)
	return totalCount, nil
}
*/

// SearchTx is making search api call
func (c *Client) SearchTx(ctx context.Context, wg *sync.WaitGroup, r structs.HeightRange, out chan cStruct.OutResp, page, perPage int, fin chan string) (count int64, err error) {
	defer wg.Done()
	defer c.logger.Sync()

	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/txs", nil)
	if err != nil {
		fin <- err.Error()
		return 0, err
	}

	req.Header.Add("Content-Type", "application/json")
	if c.key != "" {
		req.Header.Add("Authorization", c.key)
	}
	q := req.URL.Query()

	s := strings.Builder{}

	if r.EndHeight > 0 && r.EndHeight != r.StartHeight {
		s.WriteString("GTE" + strconv.Itoa(int(r.StartHeight)) + ",")
		s.WriteString("LTE" + strconv.Itoa(int(r.EndHeight)))
	} else {
		s.WriteString(strconv.Itoa(int(r.StartHeight)))
	}
	if r.StartHeight > 0 {
		q.Add("tx.height", s.String())
	}

	q.Add("page", strconv.Itoa(page))
	q.Add("limit", strconv.Itoa(perPage))
	req.URL.RawQuery = q.Encode()

	//now := time.Now()
	resp, err := c.httpClient.Do(req)
	//log.Printf("Request Time: %s", time.Now().Sub(now).String())
	//log.Printf("asd %+v", err)
	if err != nil {
		fin <- err.Error()
		return 0, err
	}

	decoder := json.NewDecoder(resp.Body)

	result := &ResultTxSearch{}
	if err = decoder.Decode(result); err != nil {
		err := fmt.Sprintf("unable to decode result body: %s", err.Error())
		fin <- err
		return 0, errors.New(err)
	}
	/*
		if result.Error.Message != "" {
			log.Println("err:", result.Error.Message)
			err := fmt.Sprintf("Error getting search: %s", result.Error.Message)
			fin <- err
			return 0, errors.New(err)
		}*/

	log.Printf("all %+v", result)
	totalCount, err := strconv.ParseInt(result.TotalCount, 10, 64)
	if err != nil {
		log.Printf("totalCount %+v %+v", totalCount, err)
		fin <- err.Error()
		return 0, err
	}

	log.Printf("result.Result.Txs %+v", result.Txs)

	rawToTransaction(ctx, c, result.Txs, out, c.logger)
	return totalCount, nil
}

func rawToTransaction(ctx context.Context, c *Client, in []TxResponse, out chan cStruct.OutResp, logger *zap.Logger) {
	for _, txRaw := range in {

		outTX := cStruct.OutResp{Type: "Transaction"}

		var err error

		tTime, err := time.Parse(time.RFC3339, txRaw.Timestamp)
		trans := shared.Transaction{
			Hash: txRaw.Hash,
			Memo: txRaw.TxData.Value.Memo,
			Time: tTime,
		}

		trans.Height, err = strconv.ParseUint(txRaw.Height, 10, 64)
		if err != nil {
			outTX.Error = err
		}
		trans.GasWanted, err = strconv.ParseUint(txRaw.GasWanted, 10, 64)
		if err != nil {
			outTX.Error = err
		}
		trans.GasUsed, err = strconv.ParseUint(txRaw.GasUsed, 10, 64)
		if err != nil {
			outTX.Error = err
		}

		for _, logf := range txRaw.Logs {

			txMsg := txRaw.TxData.Value.Msg[int(logf.MsgIndex)]

			tev := shared.TransactionEvent{
				ID:   strconv.FormatFloat(logf.MsgIndex, 'f', -1, 64),
				Kind: txMsg.Type,
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

					if len(attr.Feeder) > 0 {
						sub.Feeder = attr.Feeder
					}

					if len(attr.Recipient) > 0 {
						sub.Recipient = attr.Recipient
					}

					if len(attr.Validator) > 0 {
						sub.Validator = attr.Validator
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
		outTX.Payload = trans
		out <- outTX
	}
}

func getCurrency(in string) []string {
	return curencyRegex.FindAllString(in, 2)

}
