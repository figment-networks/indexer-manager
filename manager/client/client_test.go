package client

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"testing"

	clientMocks "github.com/figment-networks/cosmos-indexer/manager/client/mocks"
	"github.com/figment-networks/cosmos-indexer/manager/connectivity/structs"
	storeMocks "github.com/figment-networks/cosmos-indexer/manager/store/mocks"
	shared "github.com/figment-networks/cosmos-indexer/structs"
	"github.com/golang/mock/gomock"
	"go.uber.org/zap/zaptest"
)

func Test_getRanges(t *testing.T) {
	type args struct {
		in []uint64
	}
	tests := []struct {
		name       string
		args       args
		wantRanges [][2]uint64
	}{
		{name: "one range", args: args{in: []uint64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}}, wantRanges: [][2]uint64{{0, 9}}},
		{name: "two ranges", args: args{in: []uint64{1, 2, 3, 4, 6, 7, 8, 9}}, wantRanges: [][2]uint64{{1, 4}, {6, 9}}},
		{name: "two ranges with single element", args: args{in: []uint64{1, 2, 3, 4, 6, 8, 9}}, wantRanges: [][2]uint64{{1, 4}, {6, 6}, {8, 9}}},
		{name: "three ranges mixed", args: args{in: []uint64{1, 2, 11, 4, 6, 16, 8, 9, 3, 12, 13, 14, 15, 7}}, wantRanges: [][2]uint64{{1, 4}, {6, 9}, {11, 16}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotRanges := getRanges(tt.args.in); !reflect.DeepEqual(gotRanges, tt.wantRanges) {
				t.Errorf("getRanges() = %v, want %v", gotRanges, tt.wantRanges)
			}
		})
	}
}

func Test_groupRanges(t *testing.T) {
	type args struct {
		ranges [][2]uint64
		window uint64
	}
	tests := []struct {
		name    string
		args    args
		wantOut [][2]uint64
	}{
		{name: "ok", args: args{ranges: [][2]uint64{{1, 10}}, window: 2}, wantOut: [][2]uint64{{1, 3}, {4, 6}, {7, 9}, {10, 10}}},
		{name: "ok 2", args: args{ranges: [][2]uint64{{0, 100}}, window: 20}, wantOut: [][2]uint64{{0, 20}, {21, 41}, {42, 62}, {63, 83}, {84, 100}}},
		{name: "ok 3", args: args{ranges: [][2]uint64{{0, 45}, {46, 54}, {55, 100}}, window: 20}, wantOut: [][2]uint64{{0, 20}, {21, 41}, {42, 54}, {55, 75}, {76, 96}, {97, 100}}},
		{name: "ok 3", args: args{ranges: [][2]uint64{{0, 45}, {55, 100}}, window: 20}, wantOut: [][2]uint64{{0, 20}, {21, 41}, {42, 45}, {55, 75}, {76, 96}, {97, 100}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotOut := groupRanges(tt.args.ranges, tt.args.window); !reflect.DeepEqual(gotOut, tt.wantOut) {
				t.Errorf("groupRanges() = %v, want %v", gotOut, tt.wantOut)
			}
		})
	}
}

func TestClient_GetTransactions(t *testing.T) {
	InitMetrics()

	type calledSend struct {
		expectedParams  *TaskRequestComparator
		expectedReturns *TaskResponseComparator
	}
	type calls struct {
		calledSend []calledSend
	}
	type args struct {
		nv          NetworkVersion
		heightRange shared.HeightRange
		batchLimit  uint64
		silent      bool
	}

	tests := []struct {
		name    string
		args    args
		calls   calls
		want    []shared.Transaction
		wantErr bool
	}{
		{
			name: "happy path",
			args: args{
				nv:          NetworkVersion{Network: "network", ChainID: "chain_id", Version: "version"},
				heightRange: shared.HeightRange{StartHeight: 1, EndHeight: 3000},
				batchLimit:  1000,
			},
			// TODO(lukanus):  check storage as well
			want: []shared.Transaction{},
			calls: calls{
				calledSend: []calledSend{
					{
						expectedParams: &TaskRequestComparator{
							t: []structs.TaskRequest{
								{Type: "GetTransactions", Network: "network", ChainID: "chain_id", Version: "version", Payload: []byte(`{"Epoch":"","Hash":"","StartHeight":1,"EndHeight":1000}`)},
								{Type: "GetTransactions", Network: "network", ChainID: "chain_id", Version: "version", Payload: []byte(`{"Epoch":"","Hash":"","StartHeight":1001,"EndHeight":2000}`)},
								{Type: "GetTransactions", Network: "network", ChainID: "chain_id", Version: "version", Payload: []byte(`{"Epoch":"","Hash":"","StartHeight":2001,"EndHeight":3000}`)},
							},
						},
						expectedReturns: &TaskResponseComparator{
							t: []structs.TaskResponse{
								{Type: "GetTransactions", Version: "version", Order: 1, Payload: []byte(`{"Epoch":"","Hash":"","StartHeight":1,"EndHeight":1001}`)},
								{Type: "END", Version: "version", Order: 2, Final: true},
								{Type: "GetTransactions", Version: "version", Order: 3, Payload: []byte(`{"Epoch":"","Hash":"","StartHeight":1,"EndHeight":1001}`)},
								{Type: "END", Version: "version", Order: 4, Final: true},
								{Type: "GetTransactions", Version: "version", Order: 5, Payload: []byte(`{"Epoch":"","Hash":"","StartHeight":1,"EndHeight":1001}`)},
								{Type: "END", Version: "version", Order: 6, Final: true},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockClientCtrl := gomock.NewController(t)
			clientMock := clientMocks.NewMockTaskSender(mockClientCtrl)
			defer mockClientCtrl.Finish()

			for _, se := range tt.calls.calledSend {
				testAwait := &structs.Await{
					Resp: make(chan *structs.TaskResponse, len(se.expectedReturns.t)),
				}

				propagateToAwait(testAwait, se.expectedReturns.t)
				go clientMock.EXPECT().Send(se.expectedParams).Times(1).Return(testAwait, nil)
			}

			mockStoreCtrl := gomock.NewController(t)
			storeMock := storeMocks.NewMockDataStore(mockStoreCtrl)
			defer mockStoreCtrl.Finish()

			hc := NewClient(storeMock, zaptest.NewLogger(t), nil)
			hc.LinkSender(clientMock)

			ctx := context.Background()
			got, err := hc.GetTransactions(ctx, tt.args.nv, tt.args.heightRange, tt.args.batchLimit, tt.args.silent)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.GetTransactions() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.GetTransactions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func propagateToAwait(aw *structs.Await, trs []structs.TaskResponse) {
	for _, t := range trs {
		task := t
		aw.Resp <- &task
	}
}

type TaskRequestComparator struct {
	t []structs.TaskRequest
}

func (trc *TaskRequestComparator) Matches(x interface{}) bool {
	trs, ok := x.([]structs.TaskRequest)
	if !ok {
		return false
	}

	if len(trs) != len(trc.t) {
		return false
	}

	for i, tr := range trc.t {
		t := trs[i]

		if !(t.ChainID == tr.ChainID && t.Type == tr.Type &&
			t.Version == tr.Version && t.Network == tr.Network &&
			bytes.Equal(t.Payload, tr.Payload)) {
			return false
		}
	}

	return true
}

func (trc *TaskRequestComparator) String() string {
	return fmt.Sprintf("is %+v ", trc.t)
}

type TaskResponseComparator struct {
	t []structs.TaskResponse
}

func (trc *TaskResponseComparator) Matches(x interface{}) bool {
	trs, ok := x.([]structs.TaskResponse)
	if !ok {
		return false
	}

	if len(trs) != len(trc.t) {
		return false
	}

	for i, tr := range trc.t {
		t := trs[i]

		if !(t.Error == tr.Error && t.Type == tr.Type &&
			t.Version == tr.Version && t.Final == tr.Final &&
			t.Order == tr.Order && bytes.Equal(t.Payload, tr.Payload)) {
			return false
		}
	}

	return true
}

func (trc *TaskResponseComparator) String() string {
	return fmt.Sprintf("is %+v ", trc.t)
}
