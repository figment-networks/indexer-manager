package structs

import "encoding/json"

type WorkerCompositeKey struct {
	Network string
	Version string
}

type WorkerInfo struct {
	Network string
	Version string
	ID      string
}

type TaskRequest struct {
	Network string
	Version string

	Type    string
	Payload json.RawMessage

	ResponseCh chan TaskResponse
}

type TaskErrorType string

type TaskError struct {
	Msg  string
	Type TaskErrorType
}

type TaskResponse struct {
	Version string
	Type    string
	Order   int64
	Final   bool
	Error   TaskError
	Payload json.RawMessage
}
