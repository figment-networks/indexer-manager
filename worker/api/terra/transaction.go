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
	"time"

	"github.com/google/uuid"

	"github.com/figment-networks/cosmos-indexer/structs"
	shared "github.com/figment-networks/cosmos-indexer/structs"
	cStruct "github.com/figment-networks/cosmos-indexer/worker/connectivity/structs"

	log "github.com/sirupsen/logrus"
)

var curencyRegex = regexp.MustCompile("([0-9\\.\\,\\-\\s]+)([^0-9\\s]+)$")

// SearchTx is making search api call
func (c *Client) SearchTx(ctx context.Context, taskID, runUUID uuid.UUID, r structs.HeightRange, page, perPage int, fin chan string) (count int64, err error) {

	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/txs", nil)
	if err != nil {
		fin <- err.Error()
	}

	req.Header.Add("Content-Type", "application/json")
	if c.key != "" {
		req.Header.Add("Authorization", c.key)
	}

	//log.Printf("GOT %+v", r)
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

	now := time.Now()
	resp, err := c.httpClient.Do(req)
	log.Printf("Request Time: %s", time.Now().Sub(now).String())
	log.Printf("asd %+v", err)
	if err != nil {
		fin <- err.Error()
		return 0, err
	}

	decoder := json.NewDecoder(resp.Body)

	result := &ResultTxSearch{}
	if err = decoder.Decode(result); err != nil {
		err := fmt.Sprintf("unable to decode result body: %s", err.Error())
		log.Printf("asd1 %+v", err)
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
	for _, tx := range result.Txs {
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

func rawToTransaction(ctx context.Context, c *Client, in chan TxResponse, out chan cStruct.OutResp) {

	for txRaw := range in {

		outTX := cStruct.OutResp{
			ID:    txRaw.TaskID.TaskID,
			RunID: txRaw.TaskID.RunID,
			All:   uint64(txRaw.All),
			Type:  "Transaction",
		}

		var err error

		//"2020-05-03T06:52:40Z",
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
