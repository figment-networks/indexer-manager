package connectivity

import (
	"reflect"
	"testing"

	"github.com/figment-networks/cosmos-indexer/manager/connectivity/structs"
	"github.com/stretchr/testify/require"
)

func TestManager_Register(t *testing.T) {
	type args struct {
		id             string
		kind           string
		connectionInfo structs.WorkerConnection
	}
	tests := []struct {
		name         string
		args         []args
		expectedKind string
		expected     []WorkerInfo
	}{
		{name: "happy path",
			expectedKind: "cosmos",
			args: []args{
				{id: "asdf",
					kind: "cosmos",
					connectionInfo: structs.WorkerConnection{
						Version:   "0.0.1",
						Type:      "asdf",
						Addresses: []structs.WorkerAddress{{Address: "something.test"}},
					},
				},
			},
			expected: []WorkerInfo{
				{
					State:      StateInitialized,
					NodeSelfID: "asdf",
					Type:       "cosmos",
					ConnectionInfo: structs.WorkerConnection{
						Version:   "0.0.1",
						Type:      "asdf",
						Addresses: []structs.WorkerAddress{{Address: "something.test"}},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManager()

			for _, a := range tt.args {
				m.Register(a.id, a.kind, a.connectionInfo)
			}

			var found bool
			workers := m.GetWorkers(tt.expectedKind)
			require.NotEmpty(t, workers)

			for _, exp := range tt.expected {
				found = false
				for _, w := range workers {
					if w.NodeSelfID == exp.NodeSelfID &&
						w.State == StateInitialized &&
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
