package tendermint

import "github.com/google/uuid"

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
	Block Block `json:"block"`
}

// Block is cosmos block data
type Block struct {
	Header BlockHeader `json:"header"`
}

type BlockHeader struct {
	Height string `json:"height"`
}

type GetTxSearchResponse struct {
	ID     string         `json:"id"`
	RPC    string         `json:"jsonrpc"`
	Result ResultTxSearch `json:"result"`
	Error  Error          `json:"error"`
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

type GetBlockResponse struct {
	ID     string      `json:"id"`
	RPC    string      `json:"jsonrpc"`
	Result ResultBlock `json:"result"`
	Error  Error       `json:"error"`
}
