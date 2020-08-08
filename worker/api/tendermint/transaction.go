package tendermint

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/figment-networks/cosmos-indexer/structs"
	"github.com/google/uuid"

	shared "github.com/figment-networks/cosmos-indexer/structs"

	cStruct "github.com/figment-networks/cosmos-indexer/worker/connectivity/structs"

	log "github.com/sirupsen/logrus"
)

// SearchTx is making search api call
func (c *Client) SearchTx(ctx context.Context, taskID, runUUID uuid.UUID, r structs.HeightRange, page, perPage int, fin chan string) (count int64, err error) {

	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/tx_search", nil)
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
	log.Printf("Request Time: %s", time.Now().Sub(now).String())
	if err != nil {
		fin <- err.Error()
		return 0, err
	}

	decoder := json.NewDecoder(resp.Body)

	result := &GetTxSearchResponse{}
	if err = decoder.Decode(result); err != nil {
		err := fmt.Sprintf("unable to decode result body: %s", err.Error())
		fin <- err
		return 0, errors.New(err)
	}

	if result.Error.Message != "" {
		log.Println("err:", result.Error.Message)
		err := fmt.Sprintf("Error getting search: %s", result.Error.Message)
		fin <- err
		return 0, errors.New(err)
	}

	totalCount, err := strconv.ParseInt(result.Result.TotalCount, 10, 64)
	if err != nil {
		fin <- err.Error()
		return 0, err
	}

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

func rawToTransaction(in chan TxResponse, out chan cStruct.OutResp, cdc *codec.Codec) {

	readr := strings.NewReader("")
	dec := json.NewDecoder(readr)
	for txRaw := range in {
		tx := &auth.StdTx{}
		readr.Reset(txRaw.TxResult.Log)
		lf := []LogFormat{}
		err := dec.Decode(&lf)
		if err != nil {
		}

		base64Dec := base64.NewDecoder(base64.StdEncoding, strings.NewReader(txRaw.TxData))
		_, err = cdc.UnmarshalBinaryLengthPrefixedReader(base64Dec, tx, 0)

		outTX := cStruct.OutResp{
			ID:  txRaw.TaskID.TaskID,
			All: uint64(txRaw.All),
		}

		trans := shared.Transaction{
			Hash: txRaw.Hash,
			Memo: tx.GetMemo(),
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
			for _, ev := range logf.Events {
				tev := shared.TransactionEvent{
					Type: ev.Type,
				}

				for _, attr := range ev.Attributes {

					sub := shared.SubsetEvent{
						Module: attr.Module,
						Action: attr.Action,
					}

					if len(attr.Sender) > 0 {
						sub.Sender = attr.Sender
					}

					if len(attr.Recipient) > 0 {
						sub.Recipient = attr.Recipient
					}

					if len(attr.Validator) > 0 {
						sub.Validator = attr.Validator
					}

					if attr.Amount != "" {
						sub.Amount = &shared.TransactionAmount{Text: attr.Amount}
					}

					tev.Sub = append(tev.Sub, sub)
				}

				trans.Events = append(trans.Events, tev)
			}
		}

		outTX.Payload = trans
		out <- outTX

	}
}
