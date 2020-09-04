package cosmos

import (
	"bytes"
	"encoding/json"

	"github.com/google/uuid"
)

// TxResponse is result of querying for a tx
type TxResponse struct {
	Hash   string  `json:"hash"`
	Height string  `json:"height"`
	Index  float64 `json:"index"`

	TxResult ResponseDeliverTx `json:"tx_result"`
	// TxData is base64 encoded transaction data
	TxData string `json:"tx"`

	All int64

	TaskID TxID
}

type TxID struct {
	RunID  uuid.UUID
	TaskID uuid.UUID
}

type ResponseDeliverTx struct {
	Log       string   `json:"log"`
	GasWanted string   `json:"gasWanted"`
	GasUsed   string   `json:"gasUsed"`
	Tags      []TxTags `json:"tags"`
}

type TxTags struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ResultBlock is result of fetching block
type ResultBlock struct {
	Block     Block     `json:"block"`
	BlockMeta BlockMeta `json:"block_meta"`
}

// ResultBlock is result of fetching block
type ResultBlockchain struct {
	LastHeight string      `json:"last_height"`
	BlockMetas []BlockMeta `json:"block_metas"`
}

// BlockMeta is block metadata
type BlockMeta struct {
	BlockID BlockID     `json:"block_id"`
	Header  BlockHeader `json:"header"`
	NumTxs  string      `json:"num_txs"`
}

type BlockID struct {
	Hash string `json:"hash"`
}

// Block is cosmos block data
type Block struct {
	Header BlockHeader `json:"header"`
}

type BlockHeader struct {
	Height  string `json:"height"`
	ChainID string `json:"chain_id"`
	Time    string `json:"time"`
	NumTxs  string `json:"num_txs"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

// Result of searching for txs
type ResultTxSearch struct {
	Txs        []TxResponse `json:"txs"`
	TotalCount string       `json:"total_count"`
}

type GetTxSearchResponse struct {
	ID     string         `json:"id"`
	RPC    string         `json:"jsonrpc"`
	Result ResultTxSearch `json:"result"`
	Error  Error          `json:"error"`
}
type GetBlockResponse struct {
	ID     string      `json:"id"`
	RPC    string      `json:"jsonrpc"`
	Result ResultBlock `json:"result"`
	Error  Error       `json:"error"`
}

type GetBlockchainResponse struct {
	ID     string           `json:"id"`
	RPC    string           `json:"jsonrpc"`
	Result ResultBlockchain `json:"result"`
	Error  Error            `json:"error"`
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
}

type LogEventsAttributes struct {
	Module         string
	Action         string
	Amount         string
	Sender         []string
	Validator      map[string][]string
	Withdraw       map[string][]string
	Recipient      []string
	CompletionTime string
	Commission     []string
	Others         map[string][]string
}

type kvHolder struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (lea *LogEventsAttributes) UnmarshalJSON(b []byte) error {
	lea.Others = make(map[string][]string)

	dec := json.NewDecoder(bytes.NewReader(b))
	kc := &kvHolder{}
	for dec.More() {
		err := dec.Decode(kc)
		if err != nil {
			return err
		}
		switch kc.Key {
		case "validator", "destination_validator", "source_validator":
			if lea.Validator == nil {
				lea.Validator = make(map[string][]string)
			}
			v, ok := lea.Validator[kc.Key]
			if !ok {
				v = []string{}
			}
			v = append(v, kc.Value)
			lea.Validator[kc.Key] = v

		case "withdraw_address":
			if lea.Withdraw == nil {
				lea.Withdraw = make(map[string][]string)
			}
			// for now it's only address
			v, ok := lea.Withdraw["address"]
			if !ok {
				v = []string{}
			}
			v = append(v, kc.Value)
			lea.Withdraw[kc.Key] = v
		case "sender":
			lea.Sender = append(lea.Sender, kc.Value)
		case "recipient":
			lea.Recipient = append(lea.Recipient, kc.Value)
		case "module":
			lea.Module = kc.Value
		case "action":
			lea.Action = kc.Value
		case "completion_time":
			lea.CompletionTime = kc.Value
		case "amount":
			lea.Amount = kc.Value
		default:
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
