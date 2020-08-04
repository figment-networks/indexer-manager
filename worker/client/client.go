package client

import "encoding/json"

type TaskRequest struct {
	Id      string
	Type    string
	Payload json.RawMessage

	ResponseCh chan TaskResponse
}

type TaskResponse struct {
	Version string
	Id      string
	Type    string
	Payload json.RawMessage
}

type IndexerClienter interface {
	In() chan<- TaskRequest
}

type IndexerClient struct {
	taskInput chan TaskRequest
}

func NewIndexerClient() *IndexerClient {
	return &IndexerClient{make(chan TaskRequest)}
}

func (ic *IndexerClient) In() chan<- TaskRequest {
	return ic.taskInput
}
