package connectivity

import (
	"reflect"
	"testing"

	"github.com/figment-networks/cosmos-indexer/manager/connectivity/structs"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestManager_Register(t *testing.T) {
	type args struct {
		id             string
		kind           string
		chain          string
		connectionInfo structs.WorkerConnection
	}
	tests := []struct {
		name         string
		args         []args
		expectedKind string
		expected     []structs.WorkerInfo
	}{
		{name: "happy path",
			expectedKind: "cosmos",
			args: []args{
				{
					id:    "asdf",
					chain: "qwer",
					kind:  "cosmos",
					connectionInfo: structs.WorkerConnection{
						Version:   "0.0.1",
						Type:      "asdf",
						Addresses: []structs.WorkerAddress{{Address: "something.test"}},
					},
				},
			},
			expected: []structs.WorkerInfo{
				{
					State:      structs.StreamUnknown,
					NodeSelfID: "asdf",
					Type:       "cosmos",
					ConnectionInfo: []structs.WorkerConnection{{
						Version:   "0.0.1",
						Type:      "asdf",
						Addresses: []structs.WorkerAddress{{Address: "something.test"}},
					}},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			m := NewManager("test", logger)

			for _, a := range tt.args {
				m.Register(a.id, a.kind, a.chain, a.connectionInfo)
			}

			var found bool
			workers := m.GetWorkers(tt.expectedKind)
			require.NotEmpty(t, workers)

			for _, exp := range tt.expected {
				found = false
				for _, w := range workers {
					if w.NodeSelfID == exp.NodeSelfID &&
						w.State == structs.StreamUnknown &&
						reflect.DeepEqual(w.ConnectionInfo, exp.ConnectionInfo) &&
						!w.LastCheck.IsZero() {
						found = true
						break
					}
				}
				if !found {
					require.Failf(t, "Worker of given type: %s not found", tt.expectedKind)
				}
			}
		})
	}
}
