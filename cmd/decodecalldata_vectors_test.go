package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestDecodeCalldataFromVectorFile(t *testing.T) {
	vectorFile := loadABITestVectors(t)

	for i, tc := range vectorFile.Vectors {
		tc := tc
		t.Run(fmt.Sprintf("%d_%s", i, tc.Signature), func(t *testing.T) {
			if tc.SkipDecodeTest != "" {
				t.Skipf("%s", tc.SkipDecodeTest)
			}

			lookupFn := func(_ string) ([]string, error) {
				return []string{tc.Signature}, nil
			}

			got, err := decodeCalldata(tc.Calldata, "", "", lookupFn)
			if err != nil {
				t.Fatalf("decodeCalldata failed: %v", err)
			}

			if got.Signature != tc.Signature {
				t.Fatalf("signature mismatch, expected %q, got %q", tc.Signature, got.Signature)
			}

			expectedSelector := "0x" + strings.ToLower(remove0xPrefix(tc.Calldata))[0:8]
			if got.Selector != expectedSelector {
				t.Fatalf("selector mismatch, expected %q, got %q", expectedSelector, got.Selector)
			}

			if !jsonValueEqual(got.Params, tc.Params) {
				gotJSON, _ := json.Marshal(got.Params)
				expectJSON, _ := json.Marshal(tc.Params)
				t.Fatalf("params mismatch,\nexpected: %s\ngot:      %s", expectJSON, gotJSON)
			}
		})
	}
}
