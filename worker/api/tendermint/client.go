package tendermint

import (
	"bytes"
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
	shared "github.com/figment-networks/cosmos-indexer/structs"

	log "github.com/sirupsen/logrus"
)

type OutTx struct {
	Tx    shared.Transaction
	Error error
	All   int64
}

// Client is a Tendermint RPC client for cosmos using figmentnetworks datahub
type Client struct {
	baseURL    string
	key        string
	httpClient *http.Client
	cdc        *codec.Codec

	//conn *client.WSClient
}

// NewClient returns a new client for a given endpoint
func NewClient(url, key string, c *http.Client) *Client {
	if c == nil {
		c = &http.Client{
			Timeout: time.Second * 10,
		}
	}

	/* (lukanus): to use  ws in future "github.com/tendermint/tendermint/rpc/jsonrpc/client"
	conn, err := client.NewWS(addr, "/websocket")
	err = conn.Start()
	*/

	return &Client{
		//tendermint rpc url
		baseURL:    url, //todo strip trailing '/'
		key:        key,
		httpClient: c,
		cdc:        makeCodec(),
	}
}

// SearchTx is making search api call
func (c *Client) SearchTx(r structs.HeightRange, page, perPage int, out chan OutTx) (count int64, err error) {
	log.Println("[SearchTx] StartHeight ", r.StartHeight)

	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/tx_search", nil)
	if err != nil {
		// TODO(lukanus): return error
	}

	req.Header.Add("Content-Type", "application/json")
	if c.key != "" {
		req.Header.Add("Authorization", c.key)
	}

	log.Printf("GOT %+v", r)
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

	//fmt.Sprintf(`"tx.height>=%d AND tx.height<=%d"`, r.StartHeight, r.EndHeight))

	q.Add("query", s.String())
	q.Add("page", strconv.Itoa(page))
	q.Add("per_page", strconv.Itoa(perPage))
	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, err
		// TODO(lukanus): return error
	}

	decoder := json.NewDecoder(resp.Body)

	result := &GetTxSearchResponse{}
	if err = decoder.Decode(result); err != nil {
		log.WithError(err).Error("unable to decode result body")
		return
	}

	if result.Error.Message != "" {
		log.Println("err:", result.Error.Message)

		return 0, fmt.Errorf("Error getting search: %s", result.Error.Message)
		// TODO(lukanus): return error
		//0, fmt.Errorf("error fetching transactions: %d %s", result.Error.Code, result.Error.Message)
	}

	totalCount, err := strconv.ParseInt(result.Result.TotalCount, 10, 64)
	if err != nil {
		return 0, err
		// TODO(lukanus): return error
	}

	in := make(chan TxResponse, 20)
	defer close(in)

	go rawToTransaction(in, out, totalCount, c.cdc)
	go rawToTransaction(in, out, totalCount, c.cdc)

	for _, tx := range result.Result.Txs {
		in <- tx
	}
	return totalCount, nil
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
	Module string
	Action []string
	Sender []string
	Others map[string][]string
}

type kvHolder struct {
	Key   string `json:"key"`
	Value string `json:value`
}

func (lea *LogEventsAttributes) UnmarshalJSON(b []byte) error {
	lea = &LogEventsAttributes{}
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
			return err
		}

		switch kc.Key {
		case "module":
			lea.Module = kc.Value
		case "sender":
			lea.Sender = append(lea.Sender, kc.Value)
		case "action":
			lea.Action = append(lea.Action, kc.Value)
		default:
			k, ok := lea.Others[kc.Key]
			if !ok {
				k = []string{}
			}
			k = append(k, kc.Value)
			lea.Others[kc.Key] = k
		}
	}
	/*
		_, err = dec.Token()
		if err != nil {
			return err
		}
	*/
	return nil
}

func rawToTransaction(in chan TxResponse, out chan OutTx, totalCount int64, cdc *codec.Codec) {

	readr := strings.NewReader("")
	dec := json.NewDecoder(readr)
	for txRaw := range in {
		log.Printf("Inc  %+v", txRaw)
		tx := &auth.StdTx{}
		readr.Reset(txRaw.TxResult.Log)
		lf := &[]LogFormat{}
		err := dec.Decode(lf)
		if err != nil {
			log.Printf("Err  %w", err)
		}

		log.Printf("lf %+v", lf)

		base64Dec := base64.NewDecoder(base64.StdEncoding, strings.NewReader(txRaw.TxData))
		_, err = cdc.UnmarshalBinaryLengthPrefixedReader(base64Dec, tx, 0)
		//_, err := cdc.UnmarshalBinaryLengthPrefixedReader(base64Dec, tx, 0)

		outTX := OutTx{
			All: totalCount,
			Tx: shared.Transaction{
				Hash: txRaw.Hash,
				Memo: tx.GetMemo(),
			},
		}

		outTX.Tx.Height, err = strconv.ParseUint(txRaw.Height, 10, 64)
		if err != nil {
			outTX.Error = err
		}
		outTX.Tx.GasWanted, err = strconv.ParseUint(txRaw.TxResult.GasWanted, 10, 64)
		if err != nil {
			outTX.Error = err
		}
		outTX.Tx.GasUsed, err = strconv.ParseUint(txRaw.TxResult.GasUsed, 10, 64)
		if err != nil {
			outTX.Error = err
		}

		msgs := tx.GetMsgs()
		for _, m := range msgs {
			outTX.Tx.Type += "  --  " + m.Type()

		}

		for _, s := range tx.GetSigners() {
			outTX.Tx.Sender += "  --  " + s.String()
		}

		log.Printf("Tx  %+v", tx)
		log.Printf("Transaction %+v", outTX)
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
