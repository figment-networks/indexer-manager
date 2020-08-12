package coda

import (
	"encoding/json"
)

// GraphError contains the GraphQL error message
type GraphError struct {
	Message string `json:"message"`
}

// GraphResponse contains the GraphQL call response
type GraphResponse struct {
	Errors []GraphError    `json:"errors"`
	Data   json.RawMessage `json:"data"`
}

// Decode decodes the graph data into the target interface
func (r GraphResponse) Decode(dst interface{}) error {
	return json.Unmarshal(r.Data, dst)
}
