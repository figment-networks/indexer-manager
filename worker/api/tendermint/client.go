package tendermint

import (
	"bytes"
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
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/figment-networks/cosmos-indexer/structs"
	"github.com/google/uuid"

	shared "github.com/figment-networks/cosmos-indexer/structs"

	cStruct "github.com/figment-networks/cosmos-indexer/worker/connectivity/structs"

	log "github.com/sirupsen/logrus"
)

// Client is a Tendermint RPC client for cosmos using figmentnetworks datahub
type Client struct {
	baseURL    string
	key        string
	httpClient *http.Client
	cdc        *codec.Codec

	inTx chan TxResponse
	out  chan cStruct.OutResp
	//conn *client.WSClient
}

// NewClient returns a new client for a given endpoint
func NewClient(url, key string, c *http.Client) *Client {
	if c == nil {
		c = &http.Client{
			Timeout: time.Second * 40,
		}
	}

	/* (lukanus): to use  ws in future "github.com/tendermint/tendermint/rpc/jsonrpc/client"
	conn, err := client.NewWS(addr, "/websocket")
	err = conn.Start()
	*/

	cli := &Client{
		baseURL:    url, //tendermint rpc url
		key:        key,
		httpClient: c,
		cdc:        makeCodec(),
		inTx:       make(chan TxResponse, 20),
		out:        make(chan cStruct.OutResp, 20),
	}

	go rawToTransaction(cli.inTx, cli.out, cli.cdc)
	go rawToTransaction(cli.inTx, cli.out, cli.cdc)
	go rawToTransaction(cli.inTx, cli.out, cli.cdc)
	go rawToTransaction(cli.inTx, cli.out, cli.cdc)

	return cli
}

func (c *Client) Out() chan cStruct.OutResp {
	return c.out
}

// SearchTx is making search api call
func (c *Client) SearchTx(ctx context.Context, taskID uuid.UUID, r structs.HeightRange, page, perPage int, fin chan string) (count int64, err error) {

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
		tx.TaskID = taskID
		c.inTx <- tx
	}
	fin <- ""
	return totalCount, nil
}

// GetBlock fetches most recent block from chain
func (c Client) GetBlock() (*Block, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/block", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")
	if c.key != "" {
		req.Header.Add("Authorization", c.key)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)

	var result *GetBlockResponse

	if err = decoder.Decode(&result); err != nil {
		log.WithError(err).Error("unable to decode result body")
		return nil, err
	}

	if result.Error.Message != "" {
		return nil, errors.New("error fetching block") //todo make more descriptive
	}

	return &result.Result.Block, nil
}

type LogFormat struct {
	MsgIndex float64     `json:"msg_index"`
	Success  bool        `json:"success"`
	Log      string      `json:"log"`
	Events   []LogEvents `json:"events"`
}

type LogEvents struct {
	Type string `json:"type"`
	//Attributes []string `json:"attributes"`
	Attributes []*LogEventsAttributes `json:"attributes"`

	//ParsedAttributes map[string]string `json:"parsedAttributes"`
}

type LogEventsAttributes struct {
	Module    string
	Action    string
	Amount    string
	Sender    []string
	Validator []string
	Recipient []string
	Others    map[string][]string
}

type kvHolder struct {
	Key   string `json:"key"`
	Value string `json:value`
}

func (lea *LogEventsAttributes) UnmarshalJSON(b []byte) error {
	//	lea = &LogEventsAttributes{}
	lea.Others = make(map[string][]string)

	dec := json.NewDecoder(bytes.NewReader(b))
	/*	_, err := dec.Token()
		if err != nil {
			return err
		}
	*/
	kc := &kvHolder{}
	for dec.More() {
		err := dec.Decode(kc)
		if err != nil {
			log.Println("ERROR!")
			return err
		}
		switch kc.Key {
		case "validator":
			lea.Validator = append(lea.Validator, kc.Value)
		case "sender":
			lea.Sender = append(lea.Sender, kc.Value)
		case "recipient":
			lea.Recipient = append(lea.Recipient, kc.Value)
		case "module":
			lea.Module = kc.Value
		case "action":
			lea.Action = kc.Value
		case "amount":
			lea.Amount = kc.Value
		default:
			log.Println("Unknown, ", kc.Key, kc.Value)
			k, ok := lea.Others[kc.Key]
			if !ok {
				k = []string{}
			}
			k = append(k, kc.Value)
			lea.Others[kc.Key] = k
		}
	}
	return nil
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
			ID:  txRaw.TaskID,
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

func makeCodec() *codec.Codec {
	var cdc = codec.New()
	bank.RegisterCodec(cdc)
	staking.RegisterCodec(cdc)
	distr.RegisterCodec(cdc)
	slashing.RegisterCodec(cdc)
	gov.RegisterCodec(cdc)
	crisis.RegisterCodec(cdc)
	auth.RegisterCodec(cdc)
	sdk.RegisterCodec(cdc)
	codec.RegisterCrypto(cdc)
	codec.RegisterEvidences(cdc)
	return cdc
}
