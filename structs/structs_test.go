package structs

import (
	"math/big"
	"testing"
)

func TestTransactionAmount_Add(t *testing.T) {

	tests := []struct {
		name string
		args []TransactionAmount
		want TransactionAmount
	}{
		{
			name: "test with same exp",
			args: []TransactionAmount{
				{Numeric: big.NewInt(12344)},
				{Numeric: big.NewInt(1)},
			},
			want: TransactionAmount{Numeric: big.NewInt(12345), Exp: 0},
		},
		{
			name: "test with different exp",
			args: []TransactionAmount{
				{Numeric: big.NewInt(6), Exp: 1},
				{Numeric: big.NewInt(12345)},
			},
			want: TransactionAmount{Numeric: big.NewInt(123456), Exp: 1},
		},
		{
			name: "test with different exp reverse order",
			args: []TransactionAmount{
				{Numeric: big.NewInt(12345)},
				{Numeric: big.NewInt(6), Exp: 1},
			},
			want: TransactionAmount{Numeric: big.NewInt(123456), Exp: 1},
		},
		{
			name: "test with multiple additions in a row",
			args: []TransactionAmount{
				{Numeric: big.NewInt(12345)},
				{Numeric: big.NewInt(6), Exp: 1},
				{Numeric: big.NewInt(7), Exp: 2},
			},
			want: TransactionAmount{Numeric: big.NewInt(1234567), Exp: 2},
		},
		{
			name: "test with different exps",
			args: []TransactionAmount{
				{Numeric: big.NewInt(123), Exp: 3},
				{Numeric: big.NewInt(456), Exp: 6},
			},
			want: TransactionAmount{Numeric: big.NewInt(123456), Exp: 6},
		},
		{
			name: "test with different exps reverse order",
			args: []TransactionAmount{
				{Numeric: big.NewInt(456), Exp: 6},
				{Numeric: big.NewInt(123), Exp: 3},
			},
			want: TransactionAmount{Numeric: big.NewInt(123456), Exp: 6},
		},
		{
			name: "test with overflow",
			args: []TransactionAmount{
				{Numeric: big.NewInt(9), Exp: 1},
				{Numeric: big.NewInt(9), Exp: 1},
			},
			want: TransactionAmount{Numeric: big.NewInt(18), Exp: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			answer := tt.args[0].Clone()

			for _, t := range tt.args[1:] {
				answer.Add(t)
			}

			if tt.want.Numeric.Cmp(answer.Numeric) != 0 {
				t.Errorf("want: %+v, got: %+v", tt.want.Numeric, answer.Numeric)
				return
			}

			if tt.want.Exp != answer.Exp {
				t.Errorf("want: %+v, got: %+v", tt.want.Exp, answer.Exp)
				return
			}
		})
	}
}

func TestTransactionAmount_Sub(t *testing.T) {

	tests := []struct {
		name string
		args []TransactionAmount
		want TransactionAmount
	}{
		{
			name: "test with same exp",
			args: []TransactionAmount{
				{Numeric: big.NewInt(12346)},
				{Numeric: big.NewInt(1)},
			},
			want: TransactionAmount{Numeric: big.NewInt(12345), Exp: 0},
		},
		{
			name: "test with same exp",
			args: []TransactionAmount{
				{Numeric: big.NewInt(12346)},
				{Numeric: big.NewInt(1)},
			},
			want: TransactionAmount{Numeric: big.NewInt(12345), Exp: 0},
		},
		{
			name: "test with negative answer",
			args: []TransactionAmount{
				{Numeric: big.NewInt(1)},
				{Numeric: big.NewInt(12346)},
			},
			want: TransactionAmount{Numeric: big.NewInt(-12345), Exp: 0},
		},
		{
			name: "test with subtracting negative",
			args: []TransactionAmount{
				{Numeric: big.NewInt(12346)},
				{Numeric: big.NewInt(-1)},
			},
			want: TransactionAmount{Numeric: big.NewInt(12347), Exp: 0},
		},
		{
			name: "test with different exp",
			args: []TransactionAmount{
				{Numeric: big.NewInt(12346)},
				{Numeric: big.NewInt(4), Exp: 1},
			},
			want: TransactionAmount{Numeric: big.NewInt(123456), Exp: 1},
		},
		{
			name: "test with multiple subtractions in a row",
			args: []TransactionAmount{
				{Numeric: big.NewInt(200)},
				{Numeric: big.NewInt(2), Exp: 1},
				{Numeric: big.NewInt(5), Exp: 2},
			},
			want: TransactionAmount{Numeric: big.NewInt(19975), Exp: 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			answer := tt.args[0].Clone()
			for _, t := range tt.args[1:] {
				answer.Sub(t)
			}

			if tt.want.Numeric.Cmp(answer.Numeric) != 0 {
				t.Errorf("want: %+v, got: %+v", tt.want.Numeric, answer.Numeric)
				return
			}

			if tt.want.Numeric.Cmp(answer.Numeric) != 0 {
				t.Errorf("want: %+v, got: %+v", tt.want.Numeric, answer.Numeric)
				return
			}

			if tt.want.Exp != answer.Exp {
				t.Errorf("want: %+v, got: %+v", tt.want.Exp, answer.Exp)
				return
			}
		})
	}
}
