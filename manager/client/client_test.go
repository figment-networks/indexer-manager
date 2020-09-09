package client

import (
	"reflect"
	"testing"
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
