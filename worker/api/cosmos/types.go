package cosmos

import (
	"bytes"
	"encoding/json"
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
}

// ResponseDeliverTx result
type ResponseDeliverTx struct {
	Log       string  `json:"log"`
	GasWanted string  `json:"gasWanted"`
	GasUsed   string  `json:"gasUsed"`
	Tags      []TxTag `json:"tags"`
}

// TxTag is tag from cosmos
type TxTag struct {
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
}

// BlockID info
type BlockID struct {
	Hash string `json:"hash"`
}

// Block is cosmos block data
type Block struct {
	Header BlockHeader `json:"header"`
}

// BlockHeader structures
type BlockHeader struct {
	Height  string `json:"height"`
	ChainID string `json:"chain_id"`
	Time    string `json:"time"`
	NumTxs  string `json:"num_txs"`
}

// Error is api error
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data"`
}

// ResultTxSearch of searching for txs
type ResultTxSearch struct {
	Txs        []TxResponse `json:"txs"`
	TotalCount string       `json:"total_count"`
}

// GetTxSearchResponse cosmos response for search
type GetTxSearchResponse struct {
	ID     string         `json:"id"`
	RPC    string         `json:"jsonrpc"`
	Result ResultTxSearch `json:"result"`
	Error  Error          `json:"error"`
}

// GetBlockResponse cosmos response from block
type GetBlockResponse struct {
	ID     string      `json:"id"`
	RPC    string      `json:"jsonrpc"`
	Result ResultBlock `json:"result"`
	Error  Error       `json:"error"`
}

// GetBlockchainResponse cosmos response from blockchain
type GetBlockchainResponse struct {
	ID     string           `json:"id"`
	RPC    string           `json:"jsonrpc"`
	Result ResultBlockchain `json:"result"`
	Error  Error            `json:"error"`
}

// LogFormat format of logs from cosmos
type LogFormat struct {
	MsgIndex float64     `json:"msg_index"`
	Success  bool        `json:"success"`
	Log      string      `json:"log"`
	Events   []LogEvents `json:"events"`
}

// LogEvents format of events from logs cosmos
type LogEvents struct {
	Type string `json:"type"`
	//Attributes []string `json:"attributes"`
	Attributes []*LogEventsAttributes `json:"attributes"`
}

// LogEventsAttributes enhanced format of event attributes
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

// UnmarshalJSON LogEvents into a different format,
// to be able to parse it later more easily
// thats fulfillment of json.Unmarshaler inferface
func (lea *LogEventsAttributes) UnmarshalJSON(b []byte) error {

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
				lea.Validator = map[string][]string{}
			}
			v, ok := lea.Validator[kc.Key]
			if !ok {
				v = []string{}
			}
			lea.Validator[kc.Key] = append(v, kc.Value)

		case "withdraw_address":
			if lea.Withdraw == nil {
				lea.Withdraw = map[string][]string{}
			}
			// for now it's only address
			v, ok := lea.Withdraw["address"]
			if !ok {
				v = []string{}
			}
			lea.Withdraw[kc.Key] = append(v, kc.Value)
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
			if lea.Others == nil {
				lea.Others = map[string][]string{}
			}

			k, ok := lea.Others[kc.Key]
			if !ok {
				k = []string{}
			}
			lea.Others[kc.Key] = append(k, kc.Value)
		}
	}
	return nil
}
