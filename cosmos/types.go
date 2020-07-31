package cosmos

// ResultTx is result of querying for a tx
type ResultTx struct {
	Hash     string            `json:"hash"`
	Height   string            `json:"height"`
	Index    uint32            `json:"index"`
	TxResult ResponseDeliverTx `json:"tx_result"`
	// TxData is base64 encoded transaction data
	TxData string `json:"tx"`
}

type ResponseDeliverTx struct {
	GasWanted string `json:"gasWanted"`
	GasUsed   string `json:"gasUsed"`
}

// ResultBlock is result of fetching block
type ResultBlock struct {
	Block Block `json:"block"`
}

// Block is cosmos block data
type Block struct {
	Header BlockHeader `json:"header`
}

type BlockHeader struct {
	Height string `json:"height"`
}
