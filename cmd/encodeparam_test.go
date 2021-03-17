package cmd

import (
	"encoding/hex"
	"reflect"
	"testing"
)

func hex2byteArray(hexStr string) []byte {
	data, err := hex.DecodeString(hexStr)
	if err != nil {
		panic(err)
	}
	return data
}

func TestEncodeParameters(t *testing.T) {
	tests := []struct {
		input1 []string
		input2 []string
		want   []byte
	}{
		{
			input1: []string{"uint256"},
			input2: []string{"2"},
			want:   hex2byteArray("0000000000000000000000000000000000000000000000000000000000000002"),
		},
		{
			input1: []string{"bool"},
			input2: []string{"true"},
			want:   hex2byteArray("0000000000000000000000000000000000000000000000000000000000000001"),
		},
		{
			input1: []string{"bool"},
			input2: []string{"false"},
			want:   hex2byteArray("0000000000000000000000000000000000000000000000000000000000000000"),
		},
		{
			input1: []string{"address"},
			input2: []string{"0x8F36975cdeA2e6E64f85719788C8EFBBe89DFBbb"},
			want:   hex2byteArray("0000000000000000000000008f36975cdea2e6e64f85719788c8efbbe89dfbbb"),
		},
		{
			input1: []string{"uint256", "address", "bool"},
			input2: []string{"123", "0x8F36975cdeA2e6E64f85719788C8EFBBe89DFBbb", "true"},
			want:   hex2byteArray("000000000000000000000000000000000000000000000000000000000000007b0000000000000000000000008f36975cdea2e6e64f85719788c8efbbe89dfbbb0000000000000000000000000000000000000000000000000000000000000001"),
		},
		{
			input1: []string{"uint256", "address[]", "bool"},
			input2: []string{"123", "[0x8F36975cdeA2e6E64f85719788C8EFBBe89DFBbb, 0x8F36975cdeA2e6E64f85719788C8EFBBe89DFBbb]", "true"},
			want:   hex2byteArray("000000000000000000000000000000000000000000000000000000000000007b0000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000008f36975cdea2e6e64f85719788c8efbbe89dfbbb0000000000000000000000008f36975cdea2e6e64f85719788c8efbbe89dfbbb"),
		},
	}

	for i, tc := range tests {
		got, _ := encodeParameters(tc.input1, tc.input2)
		if !reflect.DeepEqual(tc.want, got) {
			// fmt.Printf("%v", hex.EncodeToString(got))
			t.Fatalf("test %d: expected: %v, got: %v", i+1, tc.want, got)
		}
	}
}
