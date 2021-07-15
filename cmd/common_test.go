package cmd

import (
	"github.com/shopspring/decimal"
	"testing"
)

func TestWei2Other(t *testing.T) {
	tests := []struct {
		sourceAmtInWei string
		targetUnit     string
		output         string
	}{
		{
			sourceAmtInWei: "1",
			targetUnit:     "wei",
			output:         "1",
		},
		{
			sourceAmtInWei: "123456789012345678",
			targetUnit:     "gwei",
			output:         "123456789.012345678",
		},
		{
			sourceAmtInWei: "123456789012345678",
			targetUnit:     "ether",
			output:         "0.123456789012345678",
		},
	}

	for i, tc := range tests {
		got := wei2Other(decimal.RequireFromString(tc.sourceAmtInWei), tc.targetUnit)
		if tc.output != got.String() {
			t.Fatalf("test %d: expected: %v, got: %v", i+1, tc.output, got)
		}
	}
}
