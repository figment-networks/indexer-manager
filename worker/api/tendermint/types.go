package tendermint

import (
	"bytes"
	"encoding/json"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
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

// BlockMeta is block metadata
type BlockMeta struct {
	BlockID BlockID `json:"block_id"`
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
