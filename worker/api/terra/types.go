package terra

import (
	"bytes"
	"encoding/json"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

// TxResponse is result of querying for a tx
type TxResponse struct {
	Hash   string `json:"txhash"`
	Height string `json:"height"`

	Logs []LogFormat `json:"logs"`

	TxResult ResponseDeliverTx `json:"tx_result"`
	// TxData is base64 encoded transaction data
	TxData    TxData `json:"tx"`
	GasWanted string `json:"gas_wanted"`
	GasUsed   string `json:"gas_used"`
	Timestamp string `json:"timestamp"`

	All    int64
	TaskID TxID
}

type TxData struct {
	Type  string      `json:"type"`
	Value TxDataValue `json:"value"`
}

type TxDataValue struct {
	Msg  []TxDataMessage `json:"msg"`
	Memo string          `json:"memo"`
	Fee  TxDataValueFee  `json:"fee"`
}

type TxDataValueFee struct {
	Amount []TxDataValueFeeAmount `json:amount`
	Gas    string                 `json:"gas"`
}

type TxDataValueFeeAmount struct {
	Amount string `json:amount`
	Denom  string `json:denom`
}

type TxDataMessage struct {
	Type  string                 `json:"type"`
	Value map[string]interface{} `json:"value"`
}

type TxID struct {
	RunID  uuid.UUID
	TaskID uuid.UUID
}

type ResponseDeliverTx struct {
	Log  string   `json:"log"`
	Tags []TxTags `json:"tags"`
}

type TxTags struct {
	Key   string `json:"key"`
	Value string `json:"value"`
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
	Txs        []TxResponse `json:"txs,omitempty"`
	TotalCount string       `json:"total_count,omitempty"`
}

type LogFormat struct {
	MsgIndex float64     `json:"msg_index,omitempty"`
	Success  bool        `json:"success,omitempty"`
	Log      string      `json:"log,omitempty"`
	Events   []LogEvents `json:"events,omitempty"`
}

type LogEvents struct {
	Type string `json:"type,omitempty"`
	//Attributes []string `json:"attributes"`
	Attributes []*LogEventsAttributes `json:"attributes,omitempty"`
}

type LogEventsAttributes struct {
	Module    string
	Action    string
	Amount    string
	Sender    []string
	Validator map[string][]string
	Recipient []string

	Voter  []string
	Feeder []string

	Denom []string

	Others map[string][]string
}

type kvHolder struct {
	Key   string `json:"key"`
	Value string `json:value`
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

		case "validator", "voter":
			if lea.Validator == nil {
				lea.Validator = make(map[string][]string)
			}
			v, ok := lea.Validator[kc.Key]
			if !ok {
				v = []string{}
			}
			v = append(v, kc.Value)
			lea.Validator[kc.Key] = v
		case "sender":
			lea.Sender = append(lea.Sender, kc.Value)
		case "recipient":
			lea.Recipient = append(lea.Recipient, kc.Value)
		case "feeder":
			lea.Feeder = append(lea.Feeder, kc.Value)
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
