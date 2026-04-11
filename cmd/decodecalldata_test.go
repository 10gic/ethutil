package cmd

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestDecodeCalldataWithABIFile(t *testing.T) {
	funcSig := "transfer(address,uint256)"
	to := "0x8F36975cdeA2e6E64f85719788C8EFBBe89DFBbb"
	amount := "1000000"

	inputData, err := buildTxInputData(funcSig, []string{to, amount})
	if err != nil {
		t.Fatalf("buildTxInputData failed: %v", err)
	}

	abiContent := `[{
		"type":"function",
		"name":"transfer",
		"inputs":[{"name":"to","type":"address"},{"name":"value","type":"uint256"}],
		"outputs":[{"name":"","type":"bool"}]
	}]`

	tmpFile, err := os.CreateTemp(t.TempDir(), "abi-*.json")
	if err != nil {
		t.Fatalf("CreateTemp failed: %v", err)
	}
	if _, err = tmpFile.WriteString(abiContent); err != nil {
		t.Fatalf("write tmp abi failed: %v", err)
	}
	if err = tmpFile.Close(); err != nil {
		t.Fatalf("close tmp abi failed: %v", err)
	}

	got, err := decodeCalldata(hexutil.Encode(inputData), tmpFile.Name(), "", func(_ string) ([]string, error) {
		t.Fatalf("lookup should not be called when --abi-file is specified")
		return nil, nil
	})
	if err != nil {
		t.Fatalf("decodeCalldata failed: %v", err)
	}

	if got.SigSource != "abi-file" {
		t.Fatalf("unexpected source, got %q", got.SigSource)
	}
	if got.Signature != "transfer(address,uint256)" {
		t.Fatalf("unexpected signature, got %q", got.Signature)
	}
	if got.Selector != "0xa9059cbb" {
		t.Fatalf("unexpected selector, got %q", got.Selector)
	}

	if got.Params["to"] != common.HexToAddress(to).Hex() {
		t.Fatalf("unexpected to, got %v", got.Params["to"])
	}
	if got.Params["value"] != amount {
		t.Fatalf("unexpected value, got %v", got.Params["value"])
	}
}

func TestDecodeCalldataWithoutABIFileFallbackToOnlineSignature(t *testing.T) {
	funcSig := "transfer(address,uint256)"
	to := "0x8F36975cdeA2e6E64f85719788C8EFBBe89DFBbb"
	amount := "1000000"

	inputData, err := buildTxInputData(funcSig, []string{to, amount})
	if err != nil {
		t.Fatalf("buildTxInputData failed: %v", err)
	}

	got, err := decodeCalldata(hexutil.Encode(inputData), "", "", func(_ string) ([]string, error) {
		return []string{
			"foo(uint256)",
			"transfer(address,uint256)",
		}, nil
	})
	if err != nil {
		t.Fatalf("decodeCalldata failed: %v", err)
	}

	if got.SigSource != "online" {
		t.Fatalf("unexpected source, got %q", got.SigSource)
	}
	if got.Signature != "transfer(address,uint256)" {
		t.Fatalf("unexpected signature, got %q", got.Signature)
	}
	if got.Params["arg0"] != common.HexToAddress(to).Hex() {
		t.Fatalf("unexpected arg0, got %v", got.Params["arg0"])
	}
	if got.Params["arg1"] != amount {
		t.Fatalf("unexpected arg1, got %v", got.Params["arg1"])
	}
}

func TestDecodeCalldataNoSignatureFound(t *testing.T) {
	funcSig := "transfer(address,uint256)"
	inputData, err := buildTxInputData(funcSig, []string{
		"0x8F36975cdeA2e6E64f85719788C8EFBBe89DFBbb",
		"1000000",
	})
	if err != nil {
		t.Fatalf("buildTxInputData failed: %v", err)
	}

	_, err = decodeCalldata(hexutil.Encode(inputData), "", "", func(_ string) ([]string, error) {
		return []string{}, nil
	})
	if err == nil {
		t.Fatalf("expect error, got nil")
	}
	if !strings.Contains(err.Error(), "no function signature found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecodeCalldataInvalidHex(t *testing.T) {
	_, err := decodeCalldata("0xabc", "", "", func(_ string) ([]string, error) { return nil, nil })
	if err == nil {
		t.Fatalf("expect error, got nil")
	}
	if !strings.Contains(err.Error(), "at least 4 bytes") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecodeCalldataJSONOutputShape(t *testing.T) {
	funcSig := "transfer(address,uint256)"
	to := "0x8F36975cdeA2e6E64f85719788C8EFBBe89DFBbb"
	amount := "1000000"

	inputData, err := buildTxInputData(funcSig, []string{to, amount})
	if err != nil {
		t.Fatalf("buildTxInputData failed: %v", err)
	}

	got, err := decodeCalldata(hexutil.Encode(inputData), "", "", func(_ string) ([]string, error) {
		return []string{"transfer(address,uint256)"}, nil
	})
	if err != nil {
		t.Fatalf("decodeCalldata failed: %v", err)
	}

	data, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("json marshal failed: %v", err)
	}

	var decoded map[string]any
	if err = json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}
	params, ok := decoded["params"].(map[string]any)
	if !ok {
		t.Fatalf("params is not object, got %T", decoded["params"])
	}
	if _, ok = params["arg0"]; !ok {
		t.Fatalf("arg0 not found in params")
	}
	if _, ok = params["arg1"]; !ok {
		t.Fatalf("arg1 not found in params")
	}
}

func TestDecodeCalldataWithFuncSig(t *testing.T) {
	funcSig := "transfer(address,uint256)"
	to := "0x8F36975cdeA2e6E64f85719788C8EFBBe89DFBbb"
	amount := "1000000"

	inputData, err := buildTxInputData(funcSig, []string{to, amount})
	if err != nil {
		t.Fatalf("buildTxInputData failed: %v", err)
	}

	got, err := decodeCalldata(hexutil.Encode(inputData), "", funcSig, func(_ string) ([]string, error) {
		t.Fatalf("lookup should not be called when --func-sig is specified")
		return nil, nil
	})
	if err != nil {
		t.Fatalf("decodeCalldata failed: %v", err)
	}

	if got.SigSource != "func-sig" {
		t.Fatalf("unexpected source, got %q", got.SigSource)
	}
	if got.Signature != "transfer(address,uint256)" {
		t.Fatalf("unexpected signature, got %q", got.Signature)
	}
	if got.Params["arg0"] != common.HexToAddress(to).Hex() {
		t.Fatalf("unexpected arg0, got %v", got.Params["arg0"])
	}
	if got.Params["arg1"] != amount {
		t.Fatalf("unexpected arg1, got %v", got.Params["arg1"])
	}
}

func TestRecursiveDecodeNestedCalldata(t *testing.T) {
	// Build inner calldata: transfer(address,uint256)
	innerCalldata, err := buildTxInputData("transfer(address,uint256)", []string{
		"0x8F36975cdeA2e6E64f85719788C8EFBBe89DFBbb",
		"1000000",
	})
	if err != nil {
		t.Fatalf("buildTxInputData for inner failed: %v", err)
	}

	// Build outer calldata: multicall(bytes[]) containing the inner calldata
	outerCalldata, err := buildTxInputData("multicall(bytes[])", []string{
		"[" + hexutil.Encode(innerCalldata) + "]",
	})
	if err != nil {
		t.Fatalf("buildTxInputData for outer failed: %v", err)
	}

	// Mock lookup returns signatures for both selectors
	mockLookup := func(selector string) ([]string, error) {
		switch selector {
		case "0xac9650d8":
			return []string{"multicall(bytes[])"}, nil
		case "0xa9059cbb":
			return []string{"transfer(address,uint256)"}, nil
		default:
			return nil, nil
		}
	}

	got, err := decodeCalldata(hexutil.Encode(outerCalldata), "", "", mockLookup)
	if err != nil {
		t.Fatalf("decodeCalldata failed: %v", err)
	}

	// Apply recursive decode
	applyRecursiveDecode(got, mockLookup, 0, maxRecursiveDepth)

	if got.Signature != "multicall(bytes[])" {
		t.Fatalf("unexpected outer signature, got %q", got.Signature)
	}

	// arg0 should be a []any with one nested DecodedCalldataOutput
	arr, ok := got.Params["arg0"].([]any)
	if !ok {
		t.Fatalf("expected arg0 to be []any, got %T", got.Params["arg0"])
	}
	if len(arr) != 1 {
		t.Fatalf("expected 1 element in arg0, got %d", len(arr))
	}

	nested, ok := arr[0].(*DecodedCalldataOutput)
	if !ok {
		t.Fatalf("expected nested DecodedCalldataOutput, got %T", arr[0])
	}
	if nested.Signature != "transfer(address,uint256)" {
		t.Fatalf("unexpected nested signature, got %q", nested.Signature)
	}
	if nested.Selector != "0xa9059cbb" {
		t.Fatalf("unexpected nested selector, got %q", nested.Selector)
	}
	if nested.Params["arg0"] != common.HexToAddress("0x8F36975cdeA2e6E64f85719788C8EFBBe89DFBbb").Hex() {
		t.Fatalf("unexpected nested arg0, got %v", nested.Params["arg0"])
	}
	if nested.Params["arg1"] != "1000000" {
		t.Fatalf("unexpected nested arg1, got %v", nested.Params["arg1"])
	}

	// Nested result should carry rawData; outer should not
	if got.RawData != "" {
		t.Fatalf("outer result should have empty rawData, got %q", got.RawData)
	}
	if nested.RawData == "" {
		t.Fatalf("nested result should have rawData set")
	}
	expectedRawData := hexutil.Encode(innerCalldata)
	if !strings.EqualFold(nested.RawData, expectedRawData) {
		t.Fatalf("unexpected nested rawData, got %q, want %q", nested.RawData, expectedRawData)
	}

	// Verify JSON output includes nested structure
	data, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatalf("json marshal failed: %v", err)
	}
	jsonStr := string(data)
	if !strings.Contains(jsonStr, "transfer(address,uint256)") {
		t.Fatalf("JSON output missing nested signature:\n%s", jsonStr)
	}
	if !strings.Contains(jsonStr, "rawData") {
		t.Fatalf("JSON output missing rawData field:\n%s", jsonStr)
	}
}

func TestRecursiveDecodeNoBytesParams(t *testing.T) {
	// transfer(address,uint256) has no bytes params, recursive should be no-op
	inputData, err := buildTxInputData("transfer(address,uint256)", []string{
		"0x8F36975cdeA2e6E64f85719788C8EFBBe89DFBbb",
		"1000000",
	})
	if err != nil {
		t.Fatalf("buildTxInputData failed: %v", err)
	}

	mockLookup := func(_ string) ([]string, error) {
		return []string{"transfer(address,uint256)"}, nil
	}

	got, err := decodeCalldata(hexutil.Encode(inputData), "", "", mockLookup)
	if err != nil {
		t.Fatalf("decodeCalldata failed: %v", err)
	}

	// Apply recursive — should not change anything
	applyRecursiveDecode(got, mockLookup, 0, maxRecursiveDepth)

	if got.Params["arg0"] != common.HexToAddress("0x8F36975cdeA2e6E64f85719788C8EFBBe89DFBbb").Hex() {
		t.Fatalf("unexpected arg0, got %v", got.Params["arg0"])
	}
	if got.Params["arg1"] != "1000000" {
		t.Fatalf("unexpected arg1, got %v", got.Params["arg1"])
	}
}

func TestRecursiveDecodeFailedLookup(t *testing.T) {
	// Build inner calldata
	innerCalldata, err := buildTxInputData("transfer(address,uint256)", []string{
		"0x8F36975cdeA2e6E64f85719788C8EFBBe89DFBbb",
		"1000000",
	})
	if err != nil {
		t.Fatalf("buildTxInputData failed: %v", err)
	}

	// Build outer: doSomething(bytes) with inner as the bytes param
	outerCalldata, err := buildTxInputData("doSomething(bytes)", []string{
		hexutil.Encode(innerCalldata),
	})
	if err != nil {
		t.Fatalf("buildTxInputData for outer failed: %v", err)
	}

	// Mock lookup: outer selector resolves, but inner selector returns nothing
	outerSelector := hexutil.Encode(outerCalldata[:4])
	mockLookup := func(selector string) ([]string, error) {
		if selector == outerSelector {
			return []string{"doSomething(bytes)"}, nil
		}
		return []string{}, nil // inner selector not found
	}

	got, err := decodeCalldata(hexutil.Encode(outerCalldata), "", "", mockLookup)
	if err != nil {
		t.Fatalf("decodeCalldata failed: %v", err)
	}

	applyRecursiveDecode(got, mockLookup, 0, maxRecursiveDepth)

	// arg0 should remain as hex string since inner decode failed
	hexStr, ok := got.Params["arg0"].(string)
	if !ok {
		t.Fatalf("expected arg0 to be string, got %T", got.Params["arg0"])
	}
	if !strings.HasPrefix(hexStr, "0x") {
		t.Fatalf("expected hex string, got %q", hexStr)
	}
}

func TestRecursiveDecodeDepthLimit(t *testing.T) {
	// Build level 2 (deepest): transfer(address,uint256)
	level2, err := buildTxInputData("transfer(address,uint256)", []string{
		"0x8F36975cdeA2e6E64f85719788C8EFBBe89DFBbb",
		"1000000",
	})
	if err != nil {
		t.Fatalf("buildTxInputData level2 failed: %v", err)
	}

	// Build level 1: doSomething(bytes) wrapping level 2
	level1, err := buildTxInputData("doSomething(bytes)", []string{
		hexutil.Encode(level2),
	})
	if err != nil {
		t.Fatalf("buildTxInputData level1 failed: %v", err)
	}

	// Build level 0: doSomething(bytes) wrapping level 1
	level0, err := buildTxInputData("doSomething(bytes)", []string{
		hexutil.Encode(level1),
	})
	if err != nil {
		t.Fatalf("buildTxInputData level0 failed: %v", err)
	}

	outerSelector := hexutil.Encode(level0[:4])
	innerSelector := hexutil.Encode(level2[:4])
	mockLookup := func(selector string) ([]string, error) {
		switch selector {
		case outerSelector:
			return []string{"doSomething(bytes)"}, nil
		case innerSelector:
			return []string{"transfer(address,uint256)"}, nil
		default:
			return []string{}, nil
		}
	}

	got, err := decodeCalldata(hexutil.Encode(level0), "", "", mockLookup)
	if err != nil {
		t.Fatalf("decodeCalldata failed: %v", err)
	}

	// Use maxDepth=1: should decode level 1 but NOT level 2
	applyRecursiveDecode(got, mockLookup, 0, 1)

	// Level 1 should be decoded
	nested1, ok := got.Params["arg0"].(*DecodedCalldataOutput)
	if !ok {
		t.Fatalf("expected level 1 to be decoded, got %T", got.Params["arg0"])
	}
	if nested1.Signature != "doSomething(bytes)" {
		t.Fatalf("unexpected level 1 signature, got %q", nested1.Signature)
	}

	// Level 2 should remain as hex string (depth limit reached)
	hexStr, ok := nested1.Params["arg0"].(string)
	if !ok {
		t.Fatalf("expected level 2 to be hex string, got %T", nested1.Params["arg0"])
	}
	if !strings.HasPrefix(hexStr, "0x") {
		t.Fatalf("expected hex string at level 2, got %q", hexStr)
	}
}

func TestLooksLikeCalldata(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect bool
	}{
		{"empty", "", false},
		{"no 0x prefix", "a9059cbb", false},
		{"too short", "0xabcd", false},
		{"bare selector only (no payload)", "0xa9059cbb", false},
		{"address (20 bytes)", "0x8F36975cdeA2e6E64f85719788C8EFBBe89DFBbb", false},
		{"selector + 32 bytes, first word starts with 4 zero bytes", "0xa9059cbb" + "00000000" + strings.Repeat("ab", 28), true},
		{"selector + 32 bytes, only 3 zero bytes prefix", "0xa9059cbb" + "000000ff" + strings.Repeat("ab", 28), false},
		{"selector + 32 bytes, first word NOT starting with 00", "0xa9059cbb" + "ff000000" + strings.Repeat("00", 28), false},
		{"selector + 31 bytes (not aligned)", "0xa9059cbb" + strings.Repeat("00", 31), false},
		{"selector + 64 bytes, leading zeros", "0xa9059cbb" + "00000000" + strings.Repeat("ab", 60), true},
		{"bytes32 value (no selector structure)", "0x" + strings.Repeat("ab", 32), false},
		{"real transfer calldata", "0xa9059cbb0000000000000000000000008f36975cdea2e6e64f85719788c8efbbe89dfbbb00000000000000000000000000000000000000000000000000000000000f4240", true},
		{"uint256 value (32 bytes, no selector+payload)", "0x" + strings.Repeat("00", 32), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := looksLikeCalldata(tt.input)
			if got != tt.expect {
				t.Errorf("looksLikeCalldata(%q) = %v, want %v", tt.input, got, tt.expect)
			}
		})
	}
}

func TestRecursiveDecodeMultipleInnerCalls(t *testing.T) {
	// Build two inner calls
	inner1, err := buildTxInputData("transfer(address,uint256)", []string{
		"0x8F36975cdeA2e6E64f85719788C8EFBBe89DFBbb",
		"1000000",
	})
	if err != nil {
		t.Fatalf("buildTxInputData inner1 failed: %v", err)
	}

	inner2, err := buildTxInputData("approve(address,uint256)", []string{
		"0xdAC17F958D2ee523a2206206994597C13D831ec7",
		"999",
	})
	if err != nil {
		t.Fatalf("buildTxInputData inner2 failed: %v", err)
	}

	// Build outer: multicall(bytes[]) containing both inner calls
	outer, err := buildTxInputData("multicall(bytes[])", []string{
		"[" + hexutil.Encode(inner1) + "," + hexutil.Encode(inner2) + "]",
	})
	if err != nil {
		t.Fatalf("buildTxInputData outer failed: %v", err)
	}

	mockLookup := func(selector string) ([]string, error) {
		switch selector {
		case "0xac9650d8":
			return []string{"multicall(bytes[])"}, nil
		case "0xa9059cbb":
			return []string{"transfer(address,uint256)"}, nil
		case "0x095ea7b3":
			return []string{"approve(address,uint256)"}, nil
		default:
			return []string{}, nil
		}
	}

	got, err := decodeCalldata(hexutil.Encode(outer), "", "", mockLookup)
	if err != nil {
		t.Fatalf("decodeCalldata failed: %v", err)
	}

	applyRecursiveDecode(got, mockLookup, 0, maxRecursiveDepth)

	// Verify JSON round-trip: marshal and unmarshal to check the full output shape
	data, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatalf("json marshal failed: %v", err)
	}

	var top map[string]any
	if err := json.Unmarshal(data, &top); err != nil {
		t.Fatalf("json unmarshal failed: %v", err)
	}

	// Top level should NOT have rawData
	if _, ok := top["rawData"]; ok {
		t.Fatalf("top-level should not have rawData")
	}

	params := top["params"].(map[string]any)
	arr := params["arg0"].([]any)
	if len(arr) != 2 {
		t.Fatalf("expected 2 inner calls, got %d", len(arr))
	}

	// Verify first inner call
	n1 := arr[0].(map[string]any)
	if n1["signature"] != "transfer(address,uint256)" {
		t.Fatalf("unexpected inner1 signature: %v", n1["signature"])
	}
	if n1["sigSource"] != "online" {
		t.Fatalf("unexpected inner1 sigSource: %v", n1["sigSource"])
	}
	if n1["rawData"] == nil || n1["rawData"] == "" {
		t.Fatalf("inner1 should have rawData")
	}

	// Verify second inner call
	n2 := arr[1].(map[string]any)
	if n2["signature"] != "approve(address,uint256)" {
		t.Fatalf("unexpected inner2 signature: %v", n2["signature"])
	}
	if n2["rawData"] == nil || n2["rawData"] == "" {
		t.Fatalf("inner2 should have rawData")
	}

	// Verify inner params via JSON
	n2Params := n2["params"].(map[string]any)
	if n2Params["arg1"] != "999" {
		t.Fatalf("unexpected inner2 arg1: %v", n2Params["arg1"])
	}
}
