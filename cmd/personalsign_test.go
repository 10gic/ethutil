package cmd

import (
	"github.com/ethereum/go-ethereum/common/hexutil"
	"testing"
)

func TestPersonalSign(t *testing.T) {
	tests := []struct {
		input1 string
		input2 string
		want   string
	}{
		{
			input1: "abc",
			input2: "0x47ab031333b76182b744e1e3b6ddb28604fdeb6ec8afdd4961335f81815c6f21",
			want:   "0x3ba90844d3f6e4b9bbb31a9e79b685d3873c10420528a9f67271a98b62230b323d8a4f0780f74513ac3296673b36938383c8723d6e47de8f6d785a0b0579f2001b",
		},
		{
			input1: "hello eth",
			input2: "0x4f66baf5a1c3a91b6cf8173cdb60d12496e1f572cee6f9f86bc507d87a9790d7",
			want:   "0xa2999d1ac51dece28dbfd6a1aa417fa48a3f9107f5fe9cdf5578859fb65763b52faf34dfb7f86069b1da162145b32949545c7cafef0017844bd3e7191d2236ae1c",
		},
		{
			input1: "hello eth",
			input2: "0x4f66baf5a1c3a91b6cf8173cdb60d12496e1f572cee6f9f86bc507d87a9790d7",
			want:   "0xa2999d1ac51dece28dbfd6a1aa417fa48a3f9107f5fe9cdf5578859fb65763b52faf34dfb7f86069b1da162145b32949545c7cafef0017844bd3e7191d2236ae1c",
		},
	}

	for i, tc := range tests {
		got, _ := personalSign([]byte(tc.input1), hexToPrivateKey(tc.input2))
		if tc.want != hexutil.Encode(got) {
			t.Fatalf("test %d: expected: %v, got: %v", i+1, tc.want, got)
		}
	}
}
