package cmd

import (
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"testing"
)

func TestEncodeParametersFromVectorFile(t *testing.T) {
	vectorFile := loadABITestVectors(t)

	for i, tc := range vectorFile.Vectors {
		tc := tc
		t.Run(fmt.Sprintf("%d_%s", i, tc.Signature), func(t *testing.T) {
			if tc.SkipEncodeTest != "" {
				t.Skipf("%s", tc.SkipEncodeTest)
			}

			args := paramsToEncodeArgs(tc.Params)

			encoded, err := buildTxInputData(tc.Signature, args)
			if err != nil {
				t.Fatalf("buildTxInputData failed: %v", err)
			}

			expectedHex := strings.ToLower(remove0xPrefix(tc.Calldata))
			gotHex := hex.EncodeToString(encoded)

			if expectedHex != gotHex {
				t.Fatalf("encode mismatch,\nexpected: 0x%s\ngot:      0x%s", expectedHex, gotHex)
			}
		})
	}
}

// paramsToEncodeArgs converts decoded params (map with arg0, arg1, ...)
// to the string slice that buildTxInputData accepts.
// Top-level args are not quoted (they are separate CLI arguments).
func paramsToEncodeArgs(params map[string]any) []string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	args := make([]string, 0, len(keys))
	for _, k := range keys {
		args = append(args, paramValueToStr(params[k], false))
	}
	return args
}

// paramValueToStr converts a decoded param value to the string format
// that buildTxInputData accepts. When nested is true (inside a tuple or
// array), strings containing commas or leading/trailing whitespace are
// wrapped in double quotes so splitTopLevel won't mis-split them.
func paramValueToStr(v any, nested bool) string {
	switch val := v.(type) {
	case string:
		if nested && needsQuoting(val) {
			return `"` + strings.ReplaceAll(val, `"`, `\"`) + `"`
		}
		return val
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%v", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case map[string]any:
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, paramValueToStr(val[k], true))
		}
		return "(" + strings.Join(parts, ", ") + ")"
	case []any:
		parts := make([]string, 0, len(val))
		for _, item := range val {
			parts = append(parts, paramValueToStr(item, true))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	default:
		return fmt.Sprintf("%v", v)
	}
}

// needsQuoting returns true if a string value needs double-quote wrapping
// to survive splitTopLevel without data loss.
func needsQuoting(s string) bool {
	return strings.ContainsAny(s, ",()[]") || s != strings.TrimSpace(s)
}
