package connectivity

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestManager_Register(t *testing.T) {
	type args struct {
		id      string
		kind    string
		address WorkerAddress
	}
	tests := []struct {
		name         string
		args         []args
		expectedKind string
		expected     []WorkerInfo
	}{
		{name: "happy path ",
			expectedKind: "cosmos",
			args: []args{
				{id: "asdf", kind: "cosmos", address: WorkerAddress{Domain: "something.test"}},
			},
			expected: []WorkerInfo{
				{
					State:      StateInitialized,
					NodeSelfID: "asdf",
					Type:       "cosmos",
					Address:    WorkerAddress{Domain: "something.test"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewManager()

			for _, a := range tt.args {
				m.Register(a.id, a.kind, a.address)
			}

			var found bool
			workers := m.GetWorkers(tt.expectedKind)
			require.NotEmpty(t, workers)

			for _, exp := range tt.expected {
				found = false
				for _, w := range workers {
					if w.NodeSelfID == exp.NodeSelfID &&
						w.State == StateInitialized &&
						w.Address.Domain == exp.Address.Domain &&
						w.Address.IP.String() == exp.Address.IP.String() &&
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
