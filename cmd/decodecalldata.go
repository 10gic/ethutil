package cmd

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"reflect"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
)

var decodeCalldataCmdABIFile string
var decodeCalldataCmdFuncSig string
var decodeCalldataCmdNonRecursive bool

func init() {
	decodeCalldataCmd.Flags().StringVarP(&decodeCalldataCmdABIFile, "abi-file", "", "", "the path of abi file")
	decodeCalldataCmd.Flags().StringVarP(&decodeCalldataCmdFuncSig, "func-sig", "", "", "the function signature, e.g. 'transfer(address,uint256)'")
	decodeCalldataCmd.Flags().BoolVarP(&decodeCalldataCmdNonRecursive, "non-recursive", "", false, "disable recursive decoding of nested calldata in bytes parameters")
}

type funcSigLookup func(selector string) ([]string, error)

type DecodedCalldataOutput struct {
	Selector   string         `json:"selector"`
	Signature  string         `json:"signature"`
	SigSource  string         `json:"sigSource"`
	Params     map[string]any `json:"params"`
	Candidates []string       `json:"candidates,omitempty"`
	RawData    string         `json:"rawData,omitempty"`
}

var tupleArrayRE = regexp.MustCompile(`^\(.+\)\[\d*\]$`)

const maxRecursiveDepth = 5

var decodeCalldataCmd = &cobra.Command{
	Use:   "decode-calldata <calldata>",
	Short: "Decode calldata, with optional --abi-file or --func-sig",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rc, err := decodeCalldata(args[0], decodeCalldataCmdABIFile, decodeCalldataCmdFuncSig, GetFuncSig)
		checkErr(err)

		if !decodeCalldataCmdNonRecursive {
			applyRecursiveDecode(rc, GetFuncSig, 0, maxRecursiveDepth)
		}

		data, err := json.MarshalIndent(rc, "", "  ")
		checkErr(err)
		fmt.Println(string(data))
	},
}

func decodeCalldata(calldata string, abiFile string, funcSig string, lookupFn funcSigLookup) (*DecodedCalldataOutput, error) {
	if lookupFn == nil {
		lookupFn = GetFuncSig
	}

	selector, payload, err := splitCalldata(calldata)
	if err != nil {
		return nil, err
	}

	if abiFile != "" {
		content, err := os.ReadFile(abiFile)
		if err != nil {
			return nil, fmt.Errorf("read abi file failed: %w", err)
		}
		contractABI, err := parseContractABI(content)
		if err != nil {
			return nil, err
		}
		return decodeCalldataWithABI(selector, payload, contractABI)
	}

	if funcSig != "" {
		rc, err := decodeCalldataWithSignature(selector, payload, funcSig, true)
		if err != nil {
			rc, err = decodeCalldataWithSignature(selector, payload, funcSig, false)
		}
		if err != nil {
			return nil, fmt.Errorf("decode with --func-sig %q failed: %w", funcSig, err)
		}
		// Verify selector matches the provided signature
		expectedSelector := "0x" + hex.EncodeToString(crypto.Keccak256([]byte(rc.Signature))[:4])
		if selector != expectedSelector {
			return nil, fmt.Errorf("selector %s does not match --func-sig %q (expected %s)", selector, funcSig, expectedSelector)
		}
		rc.SigSource = "func-sig"
		return rc, nil
	}

	sigs, err := lookupFn(selector)
	if err != nil {
		return nil, fmt.Errorf("lookup function signature failed: %w", err)
	}
	if len(sigs) == 0 {
		return nil, fmt.Errorf("no function signature found for selector %s", selector)
	}

	var firstErr error
	for _, sig := range sigs {
		rc, err := decodeCalldataWithSignature(selector, payload, sig, true)
		if err == nil {
			rc.Candidates = sigs
			return rc, nil
		}
		if len(sigs) == 1 {
			// If only one candidate signature is available, allow non-strict decoding
			// so that legacy/non-canonical calldata can still be inspected.
			rc, err = decodeCalldataWithSignature(selector, payload, sig, false)
			if err == nil {
				rc.Candidates = sigs
				return rc, nil
			}
		}
		if firstErr == nil {
			firstErr = fmt.Errorf("failed with signature %q: %w", sig, err)
		}
	}

	if firstErr != nil {
		return nil, fmt.Errorf("failed to decode calldata with %d candidate signatures: %w", len(sigs), firstErr)
	}
	return nil, fmt.Errorf("failed to decode calldata with %d candidate signatures", len(sigs))
}

func splitCalldata(calldata string) (string, []byte, error) {
	calldata = strings.TrimSpace(calldata)
	raw := remove0xPrefix(calldata)
	if len(raw) < 8 {
		return "", nil, fmt.Errorf("calldata must contain at least 4 bytes selector")
	}
	if !isValidHexString(raw) {
		return "", nil, fmt.Errorf("calldata must be hex string")
	}

	payload, err := hex.DecodeString(raw[8:])
	if err != nil {
		return "", nil, fmt.Errorf("decode calldata payload failed: %w", err)
	}

	return "0x" + strings.ToLower(raw[:8]), payload, nil
}

func parseContractABI(content []byte) (abi.ABI, error) {
	trimmed := strings.TrimSpace(string(content))
	if len(trimmed) == 0 {
		return abi.ABI{}, fmt.Errorf("abi is empty")
	}

	if trimmed[0] == '[' {
		return abi.JSON(strings.NewReader(trimmed))
	}
	if trimmed[0] == '{' {
		var wrapper map[string]json.RawMessage
		if err := json.Unmarshal([]byte(trimmed), &wrapper); err != nil {
			return abi.ABI{}, fmt.Errorf("unmarshal fail: %w", err)
		}
		abiField, ok := wrapper["abi"]
		if !ok {
			return abi.ABI{}, fmt.Errorf("abi invalid: field `abi` not found")
		}
		return abi.JSON(bytes.NewReader(abiField))
	}

	return abi.ABI{}, fmt.Errorf("abi invalid")
}

func decodeCalldataWithABI(selector string, payload []byte, contractABI abi.ABI) (*DecodedCalldataOutput, error) {
	selectorBytes, err := hex.DecodeString(remove0xPrefix(selector))
	if err != nil {
		return nil, fmt.Errorf("decode selector failed: %w", err)
	}
	method, err := contractABI.MethodById(selectorBytes)
	if err != nil {
		return nil, fmt.Errorf("resolve method by selector failed: %w", err)
	}

	values, err := method.Inputs.UnpackValues(payload)
	if err != nil {
		return nil, fmt.Errorf("decode with abi-file failed: %w", err)
	}
	if !isExactABIDecode(method.Inputs, values, payload) {
		return nil, fmt.Errorf("decode with abi-file failed: payload does not match method inputs exactly")
	}

	argNames := buildArgNames(method.Inputs)
	params := make(map[string]any, len(values))
	for i, value := range values {
		argName := fmt.Sprintf("arg%d", i)
		if i < len(argNames) {
			argName = argNames[i]
		}
		params[argName] = normalizeDecodedValue(value)
	}

	return &DecodedCalldataOutput{
		Selector:  selector,
		Signature: method.Sig,
		SigSource: "abi-file",
		Params:    params,
	}, nil
}

func decodeCalldataWithSignature(selector string, payload []byte, signature string, requireExactMatch bool) (*DecodedCalldataOutput, error) {
	funcName, argTypes, err := parseFuncSignature(signature)
	if err != nil {
		return nil, fmt.Errorf("parse function signature failed: %w", err)
	}
	inputArgs, err := buildInputArgs(argTypes)
	if err != nil {
		return nil, fmt.Errorf("build input args failed: %w", err)
	}

	values, err := inputArgs.UnpackValues(payload)
	if err != nil {
		return nil, fmt.Errorf("decode by signature failed: %w", err)
	}
	if requireExactMatch && !isExactABIDecode(inputArgs, values, payload) {
		return nil, fmt.Errorf("decode by signature failed: payload does not match inputs exactly")
	}

	params := make(map[string]any, len(values))
	for i, value := range values {
		params[fmt.Sprintf("arg%d", i)] = normalizeDecodedValue(value)
	}

	return &DecodedCalldataOutput{
		Selector:  selector,
		Signature: fmt.Sprintf("%s(%s)", funcName, strings.Join(argTypes, ",")),
		SigSource: "online",
		Params:    params,
	}, nil
}

func isExactABIDecode(inputArgs abi.Arguments, values []any, payload []byte) bool {
	repacked, err := inputArgs.PackValues(values)
	if err != nil {
		return false
	}
	return bytes.Equal(repacked, payload)
}

func buildInputArgs(inputArgTypes []string) (abi.Arguments, error) {
	var args abi.Arguments
	for _, inputArgType := range inputArgTypes {
		inputArgType = strings.TrimSpace(inputArgType)

		var (
			typ abi.Type
			err error
		)

		isTuple := strings.HasPrefix(inputArgType, "(") && strings.HasSuffix(inputArgType, ")")
		isTupleArray := tupleArrayRE.MatchString(inputArgType)

		if isTupleArray {
			typ, err = BuildTupleArrayType(inputArgType)
		} else if isTuple {
			typ, err = buildTupleType(inputArgType)
		} else {
			typ, err = abi.NewType(typeNormalize(inputArgType), "", nil)
		}
		if err != nil {
			return nil, err
		}

		args = append(args, abi.Argument{Type: typ})
	}
	return args, nil
}

func buildArgNames(inputArgs abi.Arguments) []string {
	rc := make([]string, 0, len(inputArgs))
	used := map[string]int{}

	for index, inputArg := range inputArgs {
		name := strings.TrimSpace(inputArg.Name)
		if name == "" {
			name = fmt.Sprintf("arg%d", index)
		}

		if duplicateCount, ok := used[name]; ok {
			duplicateCount++
			used[name] = duplicateCount
			name = fmt.Sprintf("%s_%d", name, duplicateCount)
		} else {
			used[name] = 0
		}

		rc = append(rc, name)
	}
	return rc
}

func normalizeDecodedValue(v any) any {
	if v == nil {
		return nil
	}

	switch data := v.(type) {
	case common.Address:
		return data.Hex()
	case *common.Address:
		return data.Hex()
	case *big.Int:
		return data.String()
	case big.Int:
		return data.String()
	case []byte:
		return hexutil.Encode(data)
	case string:
		return data
	case bool:
		return data
	}

	rv := reflect.ValueOf(v)
	if !rv.IsValid() {
		return nil
	}

	switch rv.Kind() {
	case reflect.Ptr:
		if rv.IsNil() {
			return nil
		}
		return normalizeDecodedValue(rv.Elem().Interface())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", rv.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", rv.Uint())
	case reflect.Slice:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			data := make([]byte, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				data[i] = byte(rv.Index(i).Uint())
			}
			return hexutil.Encode(data)
		}
		arr := make([]any, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			arr = append(arr, normalizeDecodedValue(rv.Index(i).Interface()))
		}
		return arr
	case reflect.Array:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			data := make([]byte, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				data[i] = byte(rv.Index(i).Uint())
			}
			return hexutil.Encode(data)
		}
		arr := make([]any, 0, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			arr = append(arr, normalizeDecodedValue(rv.Index(i).Interface()))
		}
		return arr
	case reflect.Struct:
		obj := make(map[string]any)
		rt := rv.Type()
		for i := 0; i < rv.NumField(); i++ {
			field := rt.Field(i)
			if !field.IsExported() || !rv.Field(i).CanInterface() {
				continue
			}
			obj[strings.ToLower(field.Name[:1])+field.Name[1:]] = normalizeDecodedValue(rv.Field(i).Interface())
		}
		return obj
	case reflect.Map:
		obj := make(map[string]any)
		for _, key := range rv.MapKeys() {
			mapValue := rv.MapIndex(key)
			obj[fmt.Sprintf("%v", key.Interface())] = normalizeDecodedValue(mapValue.Interface())
		}
		return obj
	default:
		return fmt.Sprintf("%v", v)
	}
}

func applyRecursiveDecode(output *DecodedCalldataOutput, lookupFn funcSigLookup, depth int, maxDepth int) {
	if output == nil || depth >= maxDepth {
		return
	}
	for key, val := range output.Params {
		output.Params[key] = walkAndDecodeNested(val, lookupFn, depth, maxDepth)
	}
}

func walkAndDecodeNested(val any, lookupFn funcSigLookup, depth int, maxDepth int) any {
	if depth >= maxDepth {
		return val
	}

	switch v := val.(type) {
	case string:
		if nested := maybeDecodeNestedCalldata(v, lookupFn, depth, maxDepth); nested != nil {
			return nested
		}
		return v
	case []any:
		for i, elem := range v {
			v[i] = walkAndDecodeNested(elem, lookupFn, depth, maxDepth)
		}
		return v
	case map[string]any:
		for k, elem := range v {
			v[k] = walkAndDecodeNested(elem, lookupFn, depth, maxDepth)
		}
		return v
	default:
		return val
	}
}

// looksLikeCalldata checks whether a hex string has the structural pattern of
// ABI-encoded calldata: a 4-byte selector followed by a payload whose length
// is a multiple of 32 bytes, and the first ABI word starts with leading zeros.
//
// The leading-zero heuristic works because the first ABI word is almost always
// a left-padded value (address, uint, offset pointer) where the high bytes are
// zero. This avoids wasting HTTP lookups on values that happen to be 32-byte
// aligned but are clearly not calldata (e.g. a raw uint256 or hash).
func looksLikeCalldata(hexStr string) bool {
	if !strings.HasPrefix(hexStr, "0x") {
		return false
	}
	hexBody := hexStr[2:]
	// At least 4 bytes selector + 32 bytes payload (one ABI word)
	if len(hexBody) < 8+64 {
		return false
	}
	payloadHexLen := len(hexBody) - 8
	// Payload must be a multiple of 32 bytes (64 hex chars)
	if payloadHexLen%64 != 0 {
		return false
	}
	// The first ABI word (bytes 4-36) should start with at least 4 zero bytes
	// (8 hex chars). In ABI encoding the first word is almost always left-padded:
	//   - address:        12 zero bytes prefix
	//   - small integers: many zero bytes prefix
	//   - offset pointer: 0x0000...0020 / 0x0000...0040 etc.
	// Requiring 4 zero bytes brings the false-positive rate on random data from
	// 1/256 (1 byte) down to ~1/4 billion, while still matching all realistic
	// calldata patterns.
	firstWordPrefix := hexBody[8:16]
	return firstWordPrefix == "00000000"
}

func maybeDecodeNestedCalldata(hexStr string, lookupFn funcSigLookup, depth int, maxDepth int) *DecodedCalldataOutput {
	if depth >= maxDepth {
		return nil
	}
	if !looksLikeCalldata(hexStr) {
		return nil
	}

	result, err := decodeCalldata(hexStr, "", "", lookupFn)
	if err != nil {
		return nil
	}

	result.RawData = hexStr
	applyRecursiveDecode(result, lookupFn, depth+1, maxDepth)
	return result
}
